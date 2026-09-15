package app

import (
	"errors"
	"regexp"

	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrAdminAlreadyInitialized = errors.New("app: administrators already initialized")

// BootstrapAdmin initializes an explicitly migrated, empty PostgreSQL admin
// schema. Credentials come from the caller; it never resets existing accounts.
func BootstrapAdmin(db *gorm.DB, username, password string) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return errors.New("app: bootstrap requires PostgreSQL")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,63}$`).MatchString(username) || len(password) < 12 || len(password) > 72 {
		return errors.New("app: username must be 3-64 account characters and password 12-72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("app: password hashing failed")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// Serialize bootstrap in the target schema, including concurrent calls.
		if err := tx.Exec("LOCK TABLE admin_users, admin_roles, admin_menus, admin_role_menus, admin_casbin_rule IN EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		for _, table := range []string{"admin_users", "admin_roles", "admin_menus", "admin_role_menus", "admin_casbin_rule"} {
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
		user := models.User{Username: username, Password: string(hash), Status: 1, RoleID: role.ID}
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
