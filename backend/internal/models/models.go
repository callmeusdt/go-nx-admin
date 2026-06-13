package models

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Username         string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password         string    `gorm:"size:255;not null" json:"-"`
	Status           int       `gorm:"default:1" json:"status"`
	RoleID           uint      `gorm:"not null" json:"role_id"`
	Role             Role      `gorm:"foreignKey:RoleID" json:"role"`
	MFAEnabled       bool      `gorm:"default:false" json:"mfa_enabled"`
	MFASecret        string    `gorm:"size:64" json:"-"`
	MFARecoveryCodes string    `gorm:"size:512" json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
	ID          uint      `gorm:"primaryKey" json:"id"`
	Group       string    `gorm:"size:64;index" json:"group"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Path        string    `gorm:"size:255;not null" json:"path"`
	Method      string    `gorm:"size:16;not null" json:"method"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

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

func AutoMigrate(db *gorm.DB, extraModels ...interface{}) error {
	for _, m := range extraModels {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}
	db.Migrator().DropIndex(&OnlineUser{}, "idx_admin_online_users_user_id")
	return db.AutoMigrate(&User{}, &Role{}, &Menu{}, &RoleMenu{}, &Api{}, &LoginLog{}, &OnlineUser{}, &AuditLog{}, &UserIPWhitelist{}, &Media{}, &SystemConfig{})
}

func Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Role{}).Count(&count)
	if count > 0 {
		migrateBackendManageMenu(db)
		ensureSystemMenus(db)
		ensureApiSeeds(db)
		ensureAdminRoleMenus(db)
		return nil
	}

	adminRole := Role{Name: "超级管理员", Slug: "admin", IsSystem: true}
	if err := db.Create(&adminRole).Error; err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	adminUser := User{
		Username: "admin",
		Password: string(hashed),
		Status:   1,
		RoleID:   adminRole.ID,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	rootMenus := []Menu{
		{Name: "仪表盘", Path: "/dashboard", Icon: "LayoutDashboard", Permission: "dashboard:view", Sort: 1},
		{Name: "后台管理", Icon: "LayoutPanelLeft", Permission: "backend:manage", Sort: 99},
	}
	for i := range rootMenus {
		if err := db.Create(&rootMenus[i]).Error; err != nil {
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
		if err := db.Create(&children[i]).Error; err != nil {
			return err
		}
	}

	migrateBackendManageMenu(db)
	ensureSystemMenus(db)
	ensureApiSeeds(db)
	ensureAdminRoleMenus(db)

	log.Println("seed data created: admin/admin123")
	return nil
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
		var menu Menu
		if err := db.Where("path = ?", item.Path).First(&menu).Error; err == nil {
			db.Model(&menu).Updates(map[string]interface{}{
				"name":       item.Name,
				"icon":       item.Icon,
				"permission": item.Permission,
				"parent_id":  backend.ID,
				"sort":       item.Sort,
			})
			continue
		}
		db.Create(&item)
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
		var m Menu
		if err := db.Where("path = ?", item.Path).First(&m).Error; err == nil {
			db.Model(&m).Updates(map[string]interface{}{
				"name": item.Name, "icon": item.Icon,
				"permission": item.Permission, "parent_id": sysMenu.ID, "sort": item.Sort,
			})
			continue
		}
		db.Create(&item)
	}

	// 系统配置种子
	configItems := []SystemConfig{
		{Key: "site_name", Value: "NX Admin", Group: "站点信息", Type: "string", Label: "站点名称", Sort: 1},
		{Key: "site_logo", Value: "", Group: "站点信息", Type: "string", Label: "站点Logo URL", Sort: 2},
		{Key: "upload_max_size", Value: "10485760", Group: "上传设置", Type: "number", Label: "上传大小限制(字节)", Sort: 1},
		{Key: "upload_allow_types", Value: "jpg,jpeg,png,gif,pdf,doc", Group: "上传设置", Type: "string", Label: "允许上传类型", Sort: 2},
	}
	for _, item := range configItems {
		db.Where("key = ?", item.Key).FirstOrCreate(&SystemConfig{}, item)
	}
}

func ensureApiSeeds(db *gorm.DB) {
	items := []Api{
		{Group: "认证", Name: "当前用户", Path: "/api/v1/auth/me", Method: "GET"},
		{Group: "认证", Name: "修改资料", Path: "/api/v1/auth/profile", Method: "PUT"},
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
	var knownPaths []string
	for _, routes := range app.Stack() {
		for _, route := range routes {
			path := route.Path
			method := route.Method
			if !strings.HasPrefix(path, "/api/") {
				continue
			}
			if path == "/api/v1/auth/login" || path == "/api/v1/auth/verify-password" || path == "/api/v1/auth/captcha" {
				continue
			}
			key := path + "|" + method
			knownPaths = append(knownPaths, key)
			group := "api"
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) >= 3 {
				group = parts[2]
			}
			var existing Api
			if err := db.Where("path = ? AND method = ?", path, method).First(&existing).Error; err == nil {
				if existing.Group == "" {
					db.Model(&existing).Update("group", group)
				}
				continue
			}
			db.Create(&Api{
				Group: group,
				Name:  method + " " + path,
				Path:  path,
				Method: method,
			})
		}
	}
	if len(knownPaths) == 0 {
		return
	}
	db.Where("path LIKE ?", "/api/v1/%").Not("path||'|'||method IN (?)", knownPaths).Delete(&Api{})
	log.Printf("api auto-discover: synced %d endpoints", len(knownPaths))
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
