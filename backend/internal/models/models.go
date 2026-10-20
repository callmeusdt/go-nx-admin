package models

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type User struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Username         string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password         string `gorm:"size:255;not null" json:"-"`
	Status           int    `gorm:"default:1" json:"status"`
	RoleID           uint   `gorm:"not null" json:"role_id"`
	Role             Role   `gorm:"foreignKey:RoleID" json:"role"`
	MFAEnabled       bool   `gorm:"default:false" json:"mfa_enabled"`
	MFASecret        string `gorm:"size:64" json:"-"`
	MFARecoveryCodes string `gorm:"size:1024" json:"-"`
	// MustChangePassword confines the account to changing its password at the
	// next sign-in; it is cleared when the user sets a new one.
	MustChangePassword bool      `gorm:"default:false" json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (User) TableName() string { return "admin_users" }

type Role struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Slug      string    `gorm:"uniqueIndex;size:64;not null" json:"slug"`
	IsSystem  bool      `gorm:"default:false" json:"is_system"`
	Users     []User    `gorm:"foreignKey:RoleID" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Role) TableName() string { return "admin_roles" }

type Menu struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ParentID   *uint     `gorm:"index" json:"parent_id"`
	Path       string    `gorm:"size:255" json:"path"`
	Permission string    `gorm:"size:128" json:"permission"`
	Name       string    `gorm:"size:64;not null" json:"name"`
	Icon       string    `gorm:"size:64" json:"icon"`
	Sort       int       `gorm:"default:0" json:"sort"`
	Children   []Menu    `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Menu) TableName() string { return "admin_menus" }

type RoleMenu struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoleID    uint      `gorm:"index:idx_role_menu,unique;not null" json:"role_id"`
	MenuID    uint      `gorm:"index:idx_role_menu,unique;not null" json:"menu_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (RoleMenu) TableName() string { return "admin_role_menus" }

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Operator  string    `gorm:"size:64" json:"operator"`
	Title     string    `gorm:"size:128" json:"title"`
	Module    string    `gorm:"size:64" json:"module"`
	Path      string    `gorm:"size:255" json:"path"`
	Method    string    `gorm:"size:10" json:"method"`
	IP        string    `gorm:"size:45" json:"ip"`
	Status    int       `json:"status"`
	Duration  int64     `gorm:"comment:microseconds" json:"duration"`
	Error     string    `gorm:"size:255" json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "admin_audit_logs" }

type Api struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Group       string `gorm:"size:64;index" json:"group"`
	Name        string `gorm:"size:64;not null" json:"name"`
	Path        string `gorm:"size:255;not null" json:"path"`
	Method      string `gorm:"size:16;not null" json:"method"`
	Description string `gorm:"size:255" json:"description"`
	// Source is ApiSourceAuto for rows created by route discovery. Only those
	// are removed when their route disappears; hand-made rows are never touched.
	Source    string    `gorm:"size:16;not null;default:''" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApiSourceAuto marks an Api row owned by AutoDiscoverAPIs.
const ApiSourceAuto = "auto"

func (Api) TableName() string { return "admin_apis" }

type LoginLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:64;index" json:"username"`
	IP        string    `gorm:"size:45" json:"ip"`
	Geo       string    `gorm:"size:128" json:"geo"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`
	Status    string    `gorm:"size:16" json:"status"`
	Message   string    `gorm:"size:255" json:"message"`
	Duration  int64     `json:"duration"`
	CreatedAt time.Time `json:"created_at"`
}

func (LoginLog) TableName() string { return "admin_login_logs" }

type OnlineUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Username  string    `gorm:"size:64" json:"username"`
	Role      string    `gorm:"size:64" json:"role"`
	Token     string    `gorm:"uniqueIndex;size:512" json:"-"`
	IP        string    `gorm:"size:45" json:"ip"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`
	LoginAt   time.Time `json:"login_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// RefreshHash is the SHA-256 of the opaque refresh token. PrevRefreshHash
	// remembers the last rotated one so a replay can be detected as token theft.
	RefreshHash     string    `gorm:"index;size:64" json:"-"`
	PrevRefreshHash string    `gorm:"index;size:64" json:"-"`
	RefreshExpires  time.Time `json:"-"`
}

func (OnlineUser) TableName() string { return "admin_online_users" }

type UserIPWhitelist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	IP        string    `gorm:"size:45;not null" json:"ip"`
	Remark    string    `gorm:"size:255" json:"remark"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserIPWhitelist) TableName() string { return "admin_user_ip_whitelists" }

// dropLegacyOnlineUserIndex removes the old unique index on user_id (one session
// per user). Current databases have a plain index of the same name, which must
// stay, so only a unique one is dropped. A fresh database has none.
func dropLegacyOnlineUserIndex(db *gorm.DB) error {
	const name = "idx_admin_online_users_user_id"
	// The SQLite driver logs its PRAGMA while listing indexes; keep startup quiet.
	quiet := db.Session(&gorm.Session{Logger: logger.Discard})
	if !quiet.Migrator().HasTable(&OnlineUser{}) {
		return nil
	}
	indexes, err := quiet.Migrator().GetIndexes(&OnlineUser{})
	if err != nil {
		return fmt.Errorf("inspect indexes: %w", err)
	}
	for _, idx := range indexes {
		if idx.Name() != name {
			continue
		}
		if unique, ok := idx.Unique(); ok && unique {
			if err := db.Migrator().DropIndex(&OnlineUser{}, name); err != nil {
				return fmt.Errorf("drop legacy index: %w", err)
			}
		}
	}
	return nil
}

// AutoMigrate creates the base tables first, then the caller's extra models, so
// an extra model may reference a base table (for example admin_users) even on a
// brand-new database.
func AutoMigrate(db *gorm.DB, extraModels ...interface{}) error {
	if err := dropLegacyOnlineUserIndex(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(&User{}, &Role{}, &Menu{}, &RoleMenu{}, &Api{}, &LoginLog{}, &OnlineUser{}, &AuditLog{}, &UserIPWhitelist{}, &Media{}, &SystemConfig{}); err != nil {
		return err
	}
	for _, m := range extraModels {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}
	return nil
}

// Seed creates the first administrator on an empty database and, on every run,
// makes sure the system menus, API rows and admin grants exist.
func Seed(db *gorm.DB) error {
	initialPassword, err := seedFirstRun(db)
	if err != nil {
		return err
	}
	migrateBackendManageMenu(db)
	ensureSystemMenus(db)
	ensureApiSeeds(db)
	ensureAdminRoleMenus(db)
	if initialPassword != "" {
		log.Printf("seed data created: initial administrator admin / %s (shown once; it must be changed at first sign-in)", initialPassword)
	}
	return nil
}

// seedFirstRun creates the admin role, account and root menus in one
// transaction, so a crash can not leave a role without its account (which the
// next start would then treat as "already seeded"). If another instance seeds
// at the same moment the unique role slug makes this one back off. It returns
// the generated password, or "" when nothing was created.
func seedFirstRun(db *gorm.DB) (string, error) {
	var password string
	err := db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&Role{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		adminRole := Role{Name: "超级管理员", Slug: "admin", IsSystem: true}
		if err := tx.Create(&adminRole).Error; err != nil {
			return err
		}
		// Every deployment gets its own first password, shown once in the log,
		// and the account must replace it at first sign-in. A published default
		// would be a working credential on every fresh install.
		initial, err := randomPassword()
		if err != nil {
			return err
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(initial), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		adminUser := User{
			Username:           "admin",
			Password:           string(hashed),
			Status:             1,
			RoleID:             adminRole.ID,
			MustChangePassword: true,
		}
		if err := tx.Create(&adminUser).Error; err != nil {
			return err
		}

		rootMenus := []Menu{
			{Name: "仪表盘", Path: "/dashboard", Icon: "LayoutDashboard", Permission: "dashboard:view", Sort: 1},
			{Name: "后台管理", Icon: "LayoutPanelLeft", Permission: "backend:manage", Sort: 99},
		}
		for i := range rootMenus {
			if err := tx.Create(&rootMenus[i]).Error; err != nil {
				return err
			}
		}
		children := []Menu{
			{Name: "用户管理", Path: "/users", Icon: "Users", Permission: "users:list", ParentID: &rootMenus[1].ID, Sort: 1},
			{Name: "角色管理", Path: "/roles", Icon: "Shield", Permission: "roles:list", ParentID: &rootMenus[1].ID, Sort: 2},
			{Name: "菜单管理", Path: "/menus", Icon: "Menu", Permission: "menus:list", ParentID: &rootMenus[1].ID, Sort: 3},
			{Name: "API管理", Path: "/apis", Icon: "Route", Permission: "apis:list", ParentID: &rootMenus[1].ID, Sort: 4},
			{Name: "登录日志", Path: "/login-logs", Icon: "LogIn", Permission: "login-logs:list", ParentID: &rootMenus[1].ID, Sort: 5},
			{Name: "操作日志", Path: "/audit-logs", Icon: "FileText", Permission: "audit-logs:list", ParentID: &rootMenus[1].ID, Sort: 6},
		}
		for i := range children {
			if err := tx.Create(&children[i]).Error; err != nil {
				return err
			}
		}
		password = initial
		return nil
	})
	if err != nil {
		// Losing the race to a concurrent seeder is not a failure.
		var roles int64
		if db.Model(&Role{}).Count(&roles).Error == nil && roles > 0 {
			return "", nil
		}
		return "", err
	}
	return password, nil
}

// randomPassword returns 20 characters (~119 bits) of unambiguous characters,
// inside the 8-32 byte password policy.
func randomPassword() (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 20)
	limit := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf), nil
}

// ensureMenu creates a system menu when no menu uses its path. An existing menu
// is left exactly as the operator configured it: names, icons, order and parent
// are theirs to change, and a restart must not undo that.
func ensureMenu(db *gorm.DB, item Menu) {
	var n int64
	if err := db.Model(&Menu{}).Where("path = ?", item.Path).Count(&n).Error; err != nil {
		log.Printf("seed menu %s: %v", item.Path, err)
		return
	}
	if n > 0 {
		return
	}
	if err := db.Create(&item).Error; err != nil {
		log.Printf("seed menu %s: %v", item.Path, err)
	}
}

func ensureSystemMenus(db *gorm.DB) {
	var backend Menu
	if err := db.Where("permission = ?", "backend:manage").First(&backend).Error; err != nil {
		return
	}
	items := []Menu{
		{Name: "API管理", Path: "/apis", Icon: "Route", Permission: "apis:list", ParentID: &backend.ID, Sort: 4},
		{Name: "登录日志", Path: "/login-logs", Icon: "LogIn", Permission: "login-logs:list", ParentID: &backend.ID, Sort: 5},
		{Name: "操作日志", Path: "/audit-logs", Icon: "FileText", Permission: "audit-logs:list", ParentID: &backend.ID, Sort: 6},
	}
	for _, item := range items {
		ensureMenu(db, item)
	}
	var auditMenus []Menu
	db.Where("path = ?", "/audit-logs").Order("id asc").Find(&auditMenus)
	if len(auditMenus) > 1 {
		keep := auditMenus[0]
		for _, item := range auditMenus[1:] {
			db.Delete(&RoleMenu{}, "menu_id = ?", item.ID)
			db.Delete(&item)
		}
		db.Model(&keep).Updates(map[string]interface{}{"parent_id": backend.ID, "sort": 6})
	}
	var online Menu
	if result := db.Where("path = ?", "/online-users").Limit(1).Find(&online); result.Error == nil && result.RowsAffected > 0 {
		db.Delete(&RoleMenu{}, "menu_id = ?", online.ID)
		db.Delete(&online)
	}

	// 系统管理 (system:settings) - new parent for system features
	var sysMenu Menu
	if result := db.Where("permission = ?", "system:settings").Limit(1).Find(&sysMenu); result.Error == nil && result.RowsAffected == 0 {
		sysMenu = Menu{Name: "系统管理", Icon: "Settings", Permission: "system:settings", Sort: 98}
		db.Create(&sysMenu)
	}

	sysItems := []Menu{
		{Name: "系统配置", Path: "/system-config", Icon: "Sliders", Permission: "system-config:list", ParentID: &sysMenu.ID, Sort: 1},
		{Name: "媒体管理", Path: "/media", Icon: "Image", Permission: "media:list", ParentID: &sysMenu.ID, Sort: 2},
	}
	for _, item := range sysItems {
		ensureMenu(db, item)
	}

	// 系统配置种子
	configItems := []SystemConfig{
		{Key: "site_name", Value: "NX Admin", Group: "站点信息", Type: "string", Label: "站点名称", Sort: 1},
		{Key: "site_logo", Value: "", Group: "站点信息", Type: "string", Label: "站点Logo URL", Sort: 2},
		{Key: "upload_max_size", Value: "10485760", Group: "上传设置", Type: "number", Label: "上传大小限制(字节)", Sort: 1},
		{Key: "upload_allow_types", Value: "jpg,jpeg,png,gif,pdf,doc", Group: "上传设置", Type: "string", Label: "允许上传类型", Sort: 2},
	}
	for _, item := range configItems {
		// Match on the key alone: an operator may have edited the other fields,
		// and "key" is reserved in MySQL so it must be quoted by GORM.
		cond := clause.Eq{Column: clause.Column{Name: "key"}, Value: item.Key}
		if err := db.Where(cond).Attrs(item).FirstOrCreate(&SystemConfig{}).Error; err != nil {
			log.Printf("seed system config %s: %v", item.Key, err)
		}
	}
}

func ensureApiSeeds(db *gorm.DB) {
	items := []Api{
		{Group: "认证", Name: "当前用户", Path: "/api/v1/auth/me", Method: "GET"},
		{Group: "认证", Name: "修改密码", Path: "/api/v1/auth/password", Method: "POST"},
		{Group: "认证", Name: "我的权限", Path: "/api/v1/auth/permissions", Method: "GET"},
		{Group: "认证", Name: "注销登录", Path: "/api/v1/auth/logout", Method: "POST"},
		{Group: "认证", Name: "我的菜单", Path: "/api/v1/auth/menus", Method: "GET"},
		{Group: "认证", Name: "我的IP", Path: "/api/v1/auth/my-ip", Method: "GET"},
		{Group: "认证", Name: "配置MFA", Path: "/api/v1/auth/mfa/setup", Method: "POST"},
		{Group: "认证", Name: "启用MFA", Path: "/api/v1/auth/mfa/enable", Method: "POST"},
		{Group: "认证", Name: "关闭MFA", Path: "/api/v1/auth/mfa/disable", Method: "POST"},
	}
	for _, item := range items {
		db.Where("path = ? AND method = ?", item.Path, item.Method).FirstOrCreate(&Api{}, item)
	}
}

func AutoDiscoverAPIs(app *fiber.App, db *gorm.DB) {
	var existing []Api
	if err := db.Find(&existing).Error; err != nil {
		log.Printf("api auto-discover: list registered endpoints: %v", err)
		return
	}
	// Compare in Go: string concatenation differs per dialect (|| is OR in MySQL).
	byKey := make(map[string]Api, len(existing))
	for _, row := range existing {
		byKey[row.Path+"|"+row.Method] = row
	}

	known := map[string]bool{}
	var created []Api
	for _, routes := range app.Stack() {
		for _, route := range routes {
			path := route.Path
			method := route.Method
			if !strings.HasPrefix(path, "/api/") {
				continue
			}
			// HEAD and OPTIONS are Fiber's automatic companions of other routes,
			// not endpoints anyone grants or manages.
			if method == fiber.MethodHead || method == fiber.MethodOptions {
				continue
			}
			if path == "/api/v1/auth/login" || path == "/api/v1/auth/verify-password" || path == "/api/v1/auth/captcha" {
				continue
			}
			key := path + "|" + method
			if known[key] {
				continue
			}
			known[key] = true
			group := "api"
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) >= 3 {
				group = parts[2]
			}
			if row, ok := byKey[key]; ok {
				if row.Group == "" {
					db.Model(&Api{}).Where("id = ?", row.ID).Update("group", group)
				}
				continue
			}
			created = append(created, Api{Group: group, Name: method + " " + path, Path: path, Method: method, Source: ApiSourceAuto})
		}
	}
	if len(created) > 0 {
		if err := db.CreateInBatches(&created, 100).Error; err != nil {
			log.Printf("api auto-discover: register endpoints: %v", err)
		}
	}
	if len(known) == 0 {
		return
	}
	var stale []uint
	for _, row := range existing {
		if row.Source == ApiSourceAuto && strings.HasPrefix(row.Path, "/api/v1/") && !known[row.Path+"|"+row.Method] {
			stale = append(stale, row.ID)
		}
	}
	for len(stale) > 0 {
		n := len(stale)
		if n > 500 {
			n = 500
		}
		if err := db.Delete(&Api{}, stale[:n]).Error; err != nil {
			log.Printf("api auto-discover: remove stale endpoints: %v", err)
			break
		}
		stale = stale[n:]
	}
	log.Printf("api auto-discover: synced %d endpoints", len(known))
}

func migrateBackendManageMenu(db *gorm.DB) {
	db.Model(&Menu{}).
		Where("permission = ?", "system:manage").
		Updates(map[string]interface{}{
			"name":       "后台管理",
			"permission": "backend:manage",
			"icon":       "LayoutPanelLeft",
		})
}

func ensureAdminRoleMenus(db *gorm.DB) {
	var role Role
	if err := db.Where("slug = ?", "admin").First(&role).Error; err != nil {
		return
	}
	var menus []Menu
	if err := db.Find(&menus).Error; err != nil {
		return
	}
	for _, menu := range menus {
		db.FirstOrCreate(&RoleMenu{}, RoleMenu{RoleID: role.ID, MenuID: menu.ID})
	}
}
