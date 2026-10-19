package app

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrAdminAlreadyInitialized = errors.New("app: administrators already initialized")

var bootstrapTables = []string{"admin_users", "admin_roles", "admin_menus", "admin_role_menus", "admin_casbin_rule"}

// bootstrapMu serializes bootstrap calls inside one process; SQLite has no
// table-level lock that blocks a concurrent reader from seeing an empty schema.
var bootstrapMu sync.Mutex

// BootstrapAdmin initializes an explicitly migrated, empty admin schema on
// PostgreSQL, MySQL/MariaDB or SQLite. Credentials come from the caller; it
// never resets existing accounts.
func BootstrapAdmin(db *gorm.DB, username, password string) error {
	if db == nil {
		return errors.New("app: bootstrap requires a database")
	}
	dialect := db.Dialector.Name()
	switch dialect {
	case "postgres", "mysql", "sqlite":
	default:
		return fmt.Errorf("app: bootstrap does not support %q", dialect)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,63}$`).MatchString(username) || len(password) < 8 || len(password) > 32 {
		return errors.New("app: username must be 3-64 account characters and password 8-32 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("app: password hashing failed")
	}
	if dialect == "mysql" {
		// A named lock held on one pinned connection spans the whole transaction;
		// MySQL cannot lock empty tables transactionally.
		return db.Connection(func(conn *gorm.DB) error {
			var got int
			if err := conn.Raw("SELECT GET_LOCK('nx_bootstrap_admin', 30)").Scan(&got).Error; err != nil {
				return err
			}
			if got != 1 {
				return errors.New("app: bootstrap lock timeout")
			}
			defer conn.Exec("DO RELEASE_LOCK('nx_bootstrap_admin')")
			return bootstrapAdmin(conn, dialect, username, string(hash))
		})
	}
	if dialect == "sqlite" {
		bootstrapMu.Lock()
		defer bootstrapMu.Unlock()
	}
	return bootstrapAdmin(db, dialect, username, string(hash))
}

func bootstrapAdmin(db *gorm.DB, dialect, username, hash string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Serialize bootstrap in the target schema, including concurrent calls.
		switch dialect {
		case "postgres":
			if err := tx.Exec("LOCK TABLE " + strings.Join(bootstrapTables, ", ") + " IN EXCLUSIVE MODE").Error; err != nil {
				return err
			}
		case "sqlite":
			// A no-op write takes SQLite's write lock before the emptiness check.
			if err := tx.Exec("UPDATE admin_roles SET id = id WHERE 1 = 0").Error; err != nil {
				return err
			}
		}
		for _, table := range bootstrapTables {
			var count int64
			if err := tx.Table(table).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return ErrAdminAlreadyInitialized
			}
		}
		role := models.Role{Name: "System administrator", Slug: "admin", IsSystem: true}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		user := models.User{Username: username, Password: hash, Status: 1, RoleID: role.ID}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')").Error; err != nil {
			return err
		}
		menus := []models.Menu{
			{Name: "仪表盘", Path: "/dashboard", Icon: "LayoutDashboard", Sort: 1},
			{Name: "管理员账号", Path: "/users", Icon: "Users", Sort: 2},
			{Name: "角色管理", Path: "/roles", Icon: "Shield", Sort: 3},
			{Name: "操作日志", Path: "/audit-logs", Icon: "FileText", Sort: 4},
			{Name: "登录日志", Path: "/login-logs", Icon: "LogIn", Sort: 5},
		}
		for i := range menus {
			if err := tx.Create(&menus[i]).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.RoleMenu{RoleID: role.ID, MenuID: menus[i].ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
