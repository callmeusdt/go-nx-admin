package handlers

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func ListUsers(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var users []models.User
		if err := db.Preload("Role").Find(&users).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}

		var onlineUsers []models.OnlineUser
		if middleware.CookieMode() {
			var sessions []models.BrowserSession
			if err := db.Where("purpose = ? AND expires_at > ?", "full", time.Now()).Order("created_at desc").Find(&sessions).Error; err != nil {
				return err
			}
			for _, session := range sessions {
				onlineUsers = append(onlineUsers, models.OnlineUser{ID: session.ID, UserID: session.UserID, Token: session.TokenHash, IP: session.IP, UserAgent: session.UserAgent, LoginAt: session.CreatedAt, UpdatedAt: session.CreatedAt})
			}
		} else if err := db.Find(&onlineUsers).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		currentToken, _ := c.Locals("token").(string)
		onlineByUserID := make(map[uint][]fiber.Map, len(onlineUsers))
		for _, item := range onlineUsers {
			onlineByUserID[item.UserID] = append(onlineByUserID[item.UserID], fiber.Map{
				"id":         item.ID,
				"ip":         item.IP,
				"user_agent": item.UserAgent,
				"login_at":   item.LoginAt,
				"updated_at": item.UpdatedAt,
				"is_current": currentToken != "" && item.Token == currentToken,
				"can_kick":   currentToken != "" && item.Token != currentToken,
			})
		}

		data := make([]fiber.Map, 0, len(users))
		for _, user := range users {
			item := fiber.Map{
				"id":          user.ID,
				"username":    user.Username,
				"status":      user.Status,
				"role_id":     user.RoleID,
				"role":        user.Role,
				"mfa_enabled": user.MFAEnabled,
				"created_at":  user.CreatedAt,
				"updated_at":  user.UpdatedAt,
				"is_online":   false,
			}
			if sessions, ok := onlineByUserID[user.ID]; ok && len(sessions) > 0 {
				latest := sessions[0]
				item["is_online"] = true
				item["online_sessions"] = sessions
				item["online_session_count"] = len(sessions)
				item["online_ip"] = latest["ip"]
				item["online_user_agent"] = latest["user_agent"]
				item["online_login_at"] = latest["login_at"]
			}
			data = append(data, item)
		}
		return c.JSON(fiber.Map{"data": data})
	}
}

func CreateUser(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			RoleID   uint   `json:"role_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := models.User{Username: req.Username, Password: string(hashed), RoleID: req.RoleID, Status: 1}
		if err := db.Create(&user).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": user})
	}
}

func UpdateUser(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req struct {
			Username string `json:"username"`
			Password string `json:"password,omitempty"`
			Status   *int   `json:"status"`
			RoleID   uint   `json:"role_id"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		updates := map[string]interface{}{"username": req.Username, "role_id": req.RoleID}
		if req.Password != "" {
			hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			updates["password"] = string(hashed)
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
			if middleware.CookieMode() {
				return tx.Where("user_id = ?", id).Delete(&models.BrowserSession{}).Error
			}
			return nil
		})
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteUser(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		if currentUserID, ok := c.Locals("user_id").(uint); ok && uint(id) == currentUserID {
			return c.Status(400).JSON(fiber.Map{"message": "cannot delete current user"})
		}
		db.Delete(&models.User{}, id)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListRoles(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var roles []models.Role
		db.Find(&roles)
		return c.JSON(fiber.Map{"data": roles})
	}
}

func CreateRole(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		role := models.Role{Name: req.Name, Slug: req.Slug, IsSystem: false}
		if err := db.Create(&role).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": role})
	}
}

func UpdateRole(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Model(&models.Role{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": req.Name, "slug": req.Slug,
		}).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteRole(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var role models.Role
		if err := db.First(&role, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "role not found"})
		}
		if role.IsSystem {
			return c.Status(400).JSON(fiber.Map{"message": "system role cannot be deleted"})
		}
		db.Delete(&role)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListMenus(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var all []models.Menu
		if err := db.Order("sort asc").Order("id asc").Find(&all).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": buildMenuTree(all)})
	}
}

func buildMenuTree(all []models.Menu) []models.Menu {
	childrenByParent := make(map[uint][]models.Menu)
	roots := make([]models.Menu, 0)
	for _, menu := range all {
		if menu.ParentID == nil {
			roots = append(roots, menu)
			continue
		}
		childrenByParent[*menu.ParentID] = append(childrenByParent[*menu.ParentID], menu)
	}

	var fillChildren func(items []models.Menu) []models.Menu
	fillChildren = func(items []models.Menu) []models.Menu {
		for i := range items {
			children := childrenByParent[items[i].ID]
			if len(children) > 0 {
				items[i].Children = fillChildren(children)
			}
		}
		return items
	}

	return fillChildren(roots)
}

func CreateMenu(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var menu models.Menu
		if err := c.BodyParser(&menu); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Create(&menu).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": menu})
	}
}

func UpdateMenu(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req models.Menu
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Model(&models.Menu{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": req.Name, "path": req.Path, "icon": req.Icon,
			"permission": req.Permission, "sort": req.Sort, "parent_id": req.ParentID,
		}).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteMenu(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var collectIDs func(uint) ([]uint, error)
		collectIDs = func(menuID uint) ([]uint, error) {
			ids := []uint{menuID}
			var children []models.Menu
			if err := db.Where("parent_id = ?", menuID).Find(&children).Error; err != nil {
				return nil, err
			}
			for _, child := range children {
				childIDs, err := collectIDs(child.ID)
				if err != nil {
					return nil, err
				}
				ids = append(ids, childIDs...)
			}
			return ids, nil
		}

		ids, err := collectIDs(uint(id))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		if err := db.Delete(&models.Menu{}, ids).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListAuditLogs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var logs []models.AuditLog
		query := db.Model(&models.AuditLog{})
		if operator := strings.TrimSpace(c.Query("operator")); operator != "" {
			query = query.Where("operator LIKE ?", "%"+operator+"%")
		}
		if path := strings.TrimSpace(c.Query("path")); path != "" {
			query = query.Where("path LIKE ?", "%"+path+"%")
		}
		if method := strings.TrimSpace(c.Query("method")); method != "" {
			query = query.Where("method = ?", method)
		}
		if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
			query = query.Where("ip LIKE ?", "%"+ip+"%")
		}
		if start := strings.TrimSpace(c.Query("start")); start != "" {
			query = query.Where("created_at >= ?", start)
		}
		if end := strings.TrimSpace(c.Query("end")); end != "" {
			query = query.Where("created_at <= ?", end)
		}
		var total int64
		query.Count(&total)
		page, pageSize, offset := pagination(c)
		if err := query.Order("created_at desc").Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": logs, "total": total, "page": page, "page_size": pageSize})
	}
}

func ListApis(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var apis []models.Api
		if err := db.Order("`group` asc").Order("id desc").Find(&apis).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": apis})
	}
}

func CreateApi(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var api models.Api
		if err := c.BodyParser(&api); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Create(&api).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": api})
	}
}

func UpdateApi(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req models.Api
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Model(&models.Api{}).Where("id = ?", id).Updates(map[string]interface{}{
			"group": req.Group, "name": req.Name, "path": req.Path,
			"method": req.Method, "description": req.Description,
		}).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteApi(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		db.Delete(&models.Api{}, id)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListLoginLogs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var logs []models.LoginLog
		query := db.Model(&models.LoginLog{})
		if username := strings.TrimSpace(c.Query("username")); username != "" {
			query = query.Where("username LIKE ?", "%"+username+"%")
		}
		if status := strings.TrimSpace(c.Query("status")); status != "" {
			query = query.Where("status = ?", status)
		}
		if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
			query = query.Where("ip LIKE ?", "%"+ip+"%")
		}
		if geo := strings.TrimSpace(c.Query("geo")); geo != "" {
			query = query.Where("geo LIKE ?", "%"+geo+"%")
		}
		if start := strings.TrimSpace(c.Query("start")); start != "" {
			query = query.Where("created_at >= ?", start)
		}
		if end := strings.TrimSpace(c.Query("end")); end != "" {
			query = query.Where("created_at <= ?", end)
		}
		var total int64
		query.Count(&total)
		page, pageSize, offset := pagination(c)
		if err := query.Order("created_at desc").Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": logs, "total": total, "page": page, "page_size": pageSize})
	}
}

func pagination(c *fiber.Ctx) (int, int, int) {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize, (page - 1) * pageSize
}

func ListOnlineUsers(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			page, size, offset := pagination(c)
			query := db.Table("admin_browser_sessions s").Joins("JOIN admin_users u ON u.id=s.user_id").Joins("JOIN admin_roles r ON r.id=u.role_id").Where("s.purpose = ? AND s.expires_at > ?", "full", time.Now())
			var total int64
			if err := query.Count(&total).Error; err != nil {
				return err
			}
			var items []map[string]any
			if err := query.Select("s.id,s.user_id,u.username,r.slug AS role,s.ip,s.user_agent,s.created_at AS login_at,s.created_at AS updated_at").Order("s.created_at desc,s.id desc").Limit(size).Offset(offset).Find(&items).Error; err != nil {
				return err
			}
			return c.JSON(fiber.Map{"data": items, "total": total, "page": page, "page_size": size})
		}
		var users []models.OnlineUser
		db.Order("updated_at desc").Find(&users)
		return c.JSON(fiber.Map{"data": users})
	}
}

func KickOnlineUser(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		currentToken, _ := c.Locals("token").(string)
		if middleware.CookieMode() {
			session := c.Locals("browser_session").(models.BrowserSession)
			if session.ID == uint(id) {
				return c.Status(400).JSON(fiber.Map{"code": "CURRENT_SESSION"})
			}
			if err := db.Delete(&models.BrowserSession{}, id).Error; err != nil {
				return err
			}
			return c.JSON(fiber.Map{"message": "ok"})
		}
		var online models.OnlineUser
		if err := db.First(&online, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "session not found"})
		}
		if currentToken != "" && online.Token == currentToken {
			return c.Status(400).JSON(fiber.Map{"message": "cannot kick current session"})
		}
		db.Delete(&online)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func MyMenus(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID := c.Locals("role_id").(uint)
		var role models.Role
		if err := db.First(&role, roleID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "role not found"})
		}

		query := db.Model(&models.Menu{}).Order("sort asc").Order("id asc")
		if !role.IsSystem {
			query = query.Joins("JOIN admin_role_menus ON admin_role_menus.menu_id = admin_menus.id AND admin_role_menus.role_id = ?", roleID)
		}

		var all []models.Menu
		if err := query.Find(&all).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": buildMenuTree(all)})
	}
}

func RoleMenuIDs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var rows []models.RoleMenu
		if err := db.Where("role_id = ?", roleID).Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		ids := make([]uint, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.MenuID)
		}
		return c.JSON(fiber.Map{"data": ids})
	}
}

func SaveRoleMenus(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req struct {
			MenuIDs []uint `json:"menu_ids"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
				return err
			}
			for _, menuID := range req.MenuIDs {
				if err := tx.Create(&models.RoleMenu{RoleID: uint(roleID), MenuID: menuID}).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func isLoginIPWhitelistEnabled() bool {
	if config.AppConfig != nil {
		return config.AppConfig.Login.IPWhitelistEnabled
	}
	return os.Getenv("NX_LOGIN_IP_WHITELIST_ENABLED") != "false"
}

func CheckLoginIPWhitelist(db *gorm.DB, userID uint, ip string) (bool, string) {
	if !isLoginIPWhitelistEnabled() {
		return true, ""
	}
	var rules []models.UserIPWhitelist
	if err := db.Where("user_id = ? AND enabled = ?", userID, true).Find(&rules).Error; err != nil {
		return false, "IP rules unavailable"
	}
	if len(rules) == 0 {
		return true, ""
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false, "invalid client IP"
	}
	for _, rule := range rules {
		if rule.IP == ip {
			return true, ""
		}
	}
	return false, "IP not allowed"
}

func ListUserIPWhitelists(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var rules []models.UserIPWhitelist
		if err := db.Where("user_id = ?", uint(id)).Order("id desc").Find(&rules).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": rules})
	}
}

func CreateUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var rule models.UserIPWhitelist
		if err := c.BodyParser(&rule); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		rule.ID = 0
		rule.UserID = uint(id)
		if rule.IP == "" {
			return c.Status(400).JSON(fiber.Map{"message": "ip required"})
		}
		if err := db.Create(&rule).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"data": rule})
	}
}

func UpdateUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, _ := strconv.ParseUint(c.Params("wid"), 10, 64)
		var req models.UserIPWhitelist
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Model(&models.UserIPWhitelist{}).Where("id = ?", wid).Updates(map[string]interface{}{
			"ip": req.IP, "remark": req.Remark, "enabled": req.Enabled,
		}).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, _ := strconv.ParseUint(c.Params("wid"), 10, 64)
		db.Delete(&models.UserIPWhitelist{}, wid)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListRolePolicies(e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Params("slug")
		return c.JSON(fiber.Map{"data": e.GetFilteredPolicy(0, role)})
	}
}

func SaveRolePolicies(e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Params("slug")
		var req struct {
			Keys     []string   `json:"keys"`
			Policies [][]string `json:"policies"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}

		policies := req.Policies
		if len(req.Keys) > 0 {
			policies = models.ExpandPermissionKeys(req.Keys)
		}

		e.RemoveFilteredPolicy(0, role)
		for _, policy := range policies {
			if len(policy) < 2 {
				continue
			}
			e.AddPolicy(role, policy[0], policy[1])
		}
		if err := e.SavePolicy(); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListRolePermissions(e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Params("slug")
		raw := e.GetFilteredPolicy(0, role)
		granted := models.GetGrantedPermissionKeys(raw)
		return c.JSON(fiber.Map{
			"permissions": models.AllPermissionDefs(),
			"granted":     granted,
		})
	}
}
