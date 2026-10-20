package handlers

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/dbutil"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ListUsers(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var users []models.User
		q := db.Preload("Role")
		if name := strings.TrimSpace(c.Query("username")); name != "" {
			q = dbutil.ContainsFold(q, "username", name)
		}
		if roleID := c.QueryInt("role_id", 0); roleID > 0 {
			q = q.Where("role_id = ?", roleID)
		}
		if err := q.Find(&users).Error; err != nil {
			return dbError(c, err)
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
			return dbError(c, err)
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
				"id":                   user.ID,
				"username":             user.Username,
				"status":               user.Status,
				"role_id":              user.RoleID,
				"role":                 user.Role,
				"mfa_enabled":          user.MFAEnabled,
				"must_change_password": user.MustChangePassword,
				"created_at":           user.CreatedAt,
				"updated_at":           user.UpdatedAt,
				"is_online":            false,
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
			Username           string `json:"username"`
			Password           string `json:"password"`
			RoleID             uint   `json:"role_id"`
			MustChangePassword bool   `json:"must_change_password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := checkPasswordPolicy(req.Password); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return internalError(c, err)
		}
		user := models.User{Username: req.Username, Password: string(hashed), RoleID: req.RoleID, Status: 1, MustChangePassword: req.MustChangePassword}
		if err := db.Create(&user).Error; err != nil {
			return dbError(c, err)
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
			// nil leaves the role as it is; an update that omits it must not
			// move the user to role 0.
			RoleID *uint `json:"role_id"`
			// nil leaves the flag as it is, so clients that do not know about
			// it can not clear a pending forced change by accident.
			MustChangePassword *bool `json:"must_change_password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		// The username is the account's identity and never changes after creation.
		updates := map[string]interface{}{}
		if req.RoleID != nil {
			if *req.RoleID == 0 {
				return c.Status(400).JSON(fiber.Map{"message": "invalid role"})
			}
			updates["role_id"] = *req.RoleID
		}
		if req.Password != "" {
			if err := checkPasswordPolicy(req.Password); err != nil {
				return c.Status(400).JSON(fiber.Map{"message": err.Error()})
			}
			hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				return internalError(c, err)
			}
			updates["password"] = string(hashed)
		}
		if req.MustChangePassword != nil {
			updates["must_change_password"] = *req.MustChangePassword
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		if len(updates) == 0 {
			return c.JSON(fiber.Map{"message": "ok"})
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
			return dbError(c, err)
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
		var exists int64
		if err := db.Model(&models.User{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return dbError(c, err)
		}
		if exists == 0 {
			return c.Status(404).JSON(fiber.Map{"message": "not found"})
		}
		if err := deleteUserCascade(db, uint(id)); err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListRoles(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var roles []models.Role
		if err := db.Find(&roles).Error; err != nil {
			return dbError(c, err)
		}
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
			return dbError(c, err)
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
		var role models.Role
		if err := db.First(&role, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "role not found"})
		}
		// Casbin policies and issued tokens refer to the role by slug, so it is
		// fixed at creation like a username.
		if req.Slug != "" && req.Slug != role.Slug {
			return c.Status(400).JSON(fiber.Map{"message": "role slug cannot be changed"})
		}
		if err := db.Model(&models.Role{}).Where("id = ?", id).Update("name", req.Name).Error; err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

var errRoleInUse = errors.New("role is assigned to users")

// deleteRole removes a role together with everything keyed by it. The Casbin
// policies are keyed by slug, so leaving them behind would hand the old
// permissions to any later role created with the same slug.
func deleteRole(db *gorm.DB, e *casbin.Enforcer, role models.Role) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var inUse int64
		if err := tx.Model(&models.User{}).Where("role_id = ?", role.ID).Count(&inUse).Error; err != nil {
			return err
		}
		if inUse > 0 {
			return errRoleInUse
		}
		if err := tx.Where("role_id = ?", role.ID).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
	if err != nil {
		return err
	}
	_, err = e.RemoveFilteredPolicy(0, role.Slug)
	return err
}

func DeleteRole(db *gorm.DB, e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var role models.Role
		if err := db.First(&role, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "role not found"})
		}
		if role.IsSystem {
			return c.Status(400).JSON(fiber.Map{"message": "system role cannot be deleted"})
		}
		if err := deleteRole(db, e, role); err != nil {
			if errors.Is(err, errRoleInUse) {
				return c.Status(409).JSON(fiber.Map{"message": errRoleInUse.Error()})
			}
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListMenus(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var all []models.Menu
		if err := db.Order("sort asc").Order("id asc").Find(&all).Error; err != nil {
			return dbError(c, err)
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

// checkMenuParent rejects a parent that does not exist or that would put the
// menu under itself or one of its own descendants (a cycle never renders and
// would send a recursive walk into an endless loop). selfID is 0 for a new menu.
func checkMenuParent(db *gorm.DB, selfID uint, parentID *uint) (string, error) {
	if parentID == nil {
		return "", nil
	}
	if *parentID == 0 {
		return "invalid parent", nil
	}
	var all []models.Menu
	if err := db.Select("id", "parent_id").Find(&all).Error; err != nil {
		return "", err
	}
	parentOf := make(map[uint]*uint, len(all))
	for _, m := range all {
		parentOf[m.ID] = m.ParentID
	}
	if _, ok := parentOf[*parentID]; !ok {
		return "parent not found", nil
	}
	seen := map[uint]bool{}
	for cur := parentID; cur != nil && !seen[*cur]; cur = parentOf[*cur] {
		if selfID != 0 && *cur == selfID {
			return "menu cannot be its own ancestor", nil
		}
		seen[*cur] = true
	}
	return "", nil
}

func CreateMenu(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var menu models.Menu
		if err := c.BodyParser(&menu); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		menu.ID = 0
		menu.Children = nil
		if msg, err := checkMenuParent(db, 0, menu.ParentID); err != nil {
			return dbError(c, err)
		} else if msg != "" {
			return c.Status(400).JSON(fiber.Map{"message": msg})
		}
		if err := db.Create(&menu).Error; err != nil {
			return dbError(c, err)
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
		if msg, err := checkMenuParent(db, uint(id), req.ParentID); err != nil {
			return dbError(c, err)
		} else if msg != "" {
			return c.Status(400).JSON(fiber.Map{"message": msg})
		}
		if err := db.Model(&models.Menu{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": req.Name, "path": req.Path, "icon": req.Icon,
			"permission": req.Permission, "sort": req.Sort, "parent_id": req.ParentID,
		}).Error; err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

// menuSubtreeIDs returns id and all its descendants in one query. A visited set
// keeps a corrupt (cyclic) tree from looping forever.
func menuSubtreeIDs(db *gorm.DB, id uint) ([]uint, error) {
	var all []models.Menu
	if err := db.Select("id", "parent_id").Find(&all).Error; err != nil {
		return nil, err
	}
	children := make(map[uint][]uint, len(all))
	for _, m := range all {
		if m.ParentID != nil {
			children[*m.ParentID] = append(children[*m.ParentID], m.ID)
		}
	}
	out := []uint{}
	seen := map[uint]bool{}
	for queue := []uint{id}; len(queue) > 0; {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		out = append(out, cur)
		queue = append(queue, children[cur]...)
	}
	return out, nil
}

func DeleteMenu(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		ids, err := menuSubtreeIDs(db, uint(id))
		if err != nil {
			return dbError(c, err)
		}
		// Role grants point at menu ids; leaving them behind would attach the
		// old grant to whatever menu later reuses the id.
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("menu_id IN ?", ids).Delete(&models.RoleMenu{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", ids).Delete(&models.Menu{}).Error
		}); err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListAuditLogs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var logs []models.AuditLog
		query := db.Model(&models.AuditLog{})
		if operator := strings.TrimSpace(c.Query("operator")); operator != "" {
			query = dbutil.ContainsFold(query, "operator", operator)
		}
		if path := strings.TrimSpace(c.Query("path")); path != "" {
			query = dbutil.ContainsFold(query, "path", path)
		}
		if method := strings.TrimSpace(c.Query("method")); method != "" {
			query = query.Where("method = ?", method)
		}
		if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
			query = dbutil.ContainsFold(query, "ip", ip)
		}
		if start := strings.TrimSpace(c.Query("start")); start != "" {
			query = query.Where("created_at >= ?", start)
		}
		if end := strings.TrimSpace(c.Query("end")); end != "" {
			query = query.Where("created_at <= ?", end)
		}
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return dbError(c, err)
		}
		page, pageSize, offset := pagination(c)
		if err := query.Order("created_at desc").Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"data": logs, "total": total, "page": page, "page_size": pageSize})
	}
}

func ListApis(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var apis []models.Api
		if err := db.Order(clause.OrderByColumn{Column: clause.Column{Name: "group"}}).Order("id desc").Find(&apis).Error; err != nil {
			return dbError(c, err)
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
			return dbError(c, err)
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
			return dbError(c, err)
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
			query = dbutil.ContainsFold(query, "username", username)
		}
		if status := strings.TrimSpace(c.Query("status")); status != "" {
			query = query.Where("status = ?", status)
		}
		if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
			query = dbutil.ContainsFold(query, "ip", ip)
		}
		if geo := strings.TrimSpace(c.Query("geo")); geo != "" {
			query = dbutil.ContainsFold(query, "geo", geo)
		}
		if start := strings.TrimSpace(c.Query("start")); start != "" {
			query = query.Where("created_at >= ?", start)
		}
		if end := strings.TrimSpace(c.Query("end")); end != "" {
			query = query.Where("created_at <= ?", end)
		}
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return dbError(c, err)
		}
		page, pageSize, offset := pagination(c)
		if err := query.Order("created_at desc").Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
			return dbError(c, err)
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
		if err := db.Order("updated_at desc").Find(&users).Error; err != nil {
			return dbError(c, err)
		}
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

		query := db.Model(&models.Menu{}).Order("admin_menus.sort asc").Order("admin_menus.id asc")
		if !role.IsSystem {
			query = query.Joins("JOIN admin_role_menus ON admin_role_menus.menu_id = admin_menus.id AND admin_role_menus.role_id = ?", roleID)
		}

		var all []models.Menu
		if err := query.Find(&all).Error; err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"data": buildMenuTree(all)})
	}
}

func RoleMenuIDs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var rows []models.RoleMenu
		if err := db.Where("role_id = ?", roleID).Find(&rows).Error; err != nil {
			return dbError(c, err)
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
			return dbError(c, err)
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
		if ipRuleMatches(rule.IP, parsed) {
			return true, ""
		}
	}
	return false, "IP not allowed"
}

// ipRuleMatches reports whether a stored rule (a single address or a CIDR
// range) covers ip. Addresses are compared as parsed values, so equivalent
// IPv6 spellings match.
func ipRuleMatches(rule string, ip net.IP) bool {
	rule = strings.TrimSpace(rule)
	if strings.Contains(rule, "/") {
		_, network, err := net.ParseCIDR(rule)
		return err == nil && network.Contains(ip)
	}
	allowed := net.ParseIP(rule)
	return allowed != nil && allowed.Equal(ip)
}

// validIPRule accepts a single IP address or a CIDR range.
func validIPRule(rule string) bool {
	if strings.Contains(rule, "/") {
		_, _, err := net.ParseCIDR(rule)
		return err == nil
	}
	return net.ParseIP(rule) != nil
}

func ListUserIPWhitelists(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var rules []models.UserIPWhitelist
		if err := db.Where("user_id = ?", uint(id)).Order("id desc").Find(&rules).Error; err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"data": rules})
	}
}

func CreateUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var req struct {
			IP      string `json:"ip"`
			Remark  string `json:"remark"`
			Enabled *bool  `json:"enabled"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		req.IP = strings.TrimSpace(req.IP)
		if req.IP == "" {
			return c.Status(400).JSON(fiber.Map{"message": "ip required"})
		}
		if !validIPRule(req.IP) {
			return c.Status(400).JSON(fiber.Map{"message": "invalid ip or cidr"})
		}
		rule := models.UserIPWhitelist{UserID: uint(id), IP: req.IP, Remark: req.Remark}
		enabled := req.Enabled == nil || *req.Enabled
		// The column default is true and GORM substitutes it for a zero bool on
		// insert, so an explicit "disabled" is written as a follow-up update.
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
			if !enabled {
				return tx.Model(&rule).Update("enabled", false).Error
			}
			return nil
		}); err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"data": rule})
	}
}

func UpdateUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		wid, _ := strconv.ParseUint(c.Params("wid"), 10, 64)
		var req struct {
			IP      *string `json:"ip"`
			Remark  *string `json:"remark"`
			Enabled *bool   `json:"enabled"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		updates := map[string]interface{}{}
		if req.IP != nil {
			ip := strings.TrimSpace(*req.IP)
			if !validIPRule(ip) {
				return c.Status(400).JSON(fiber.Map{"message": "invalid ip or cidr"})
			}
			updates["ip"] = ip
		}
		if req.Remark != nil {
			updates["remark"] = *req.Remark
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if len(updates) == 0 {
			return c.JSON(fiber.Map{"message": "ok"})
		}
		// The rule must belong to the user in the URL, otherwise any rule id
		// could be edited through any user's path.
		res := db.Model(&models.UserIPWhitelist{}).Where("id = ? AND user_id = ?", wid, id).Updates(updates)
		if res.Error != nil {
			return dbError(c, res.Error)
		}
		if res.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"message": "not found"})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func DeleteUserIPWhitelist(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		wid, _ := strconv.ParseUint(c.Params("wid"), 10, 64)
		res := db.Where("id = ? AND user_id = ?", wid, id).Delete(&models.UserIPWhitelist{})
		if res.Error != nil {
			return dbError(c, res.Error)
		}
		if res.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"message": "not found"})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func ListRolePolicies(e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Params("slug")
		return c.JSON(fiber.Map{"data": e.GetFilteredPolicy(0, role)})
	}
}

// SaveRolePermissions replaces a role's policies with the expansion of the
// submitted permission keys. Raw policies are never accepted: a caller who may
// edit permissions could otherwise grant itself ".*", write a pattern that
// breaks matching for the role, or rewrite the system role.
func SaveRolePermissions(e *casbin.Enforcer, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fiber's Params string aliases the request buffer; the enforcer keeps
		// it in memory after the request ends, so it must be copied.
		slug := strings.Clone(c.Params("slug"))
		var req struct {
			Keys []string `json:"keys"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		var role models.Role
		if err := db.Where("slug = ?", slug).First(&role).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "role not found"})
		}
		if role.IsSystem {
			return c.Status(403).JSON(fiber.Map{"message": "system role permissions cannot be changed"})
		}
		known := map[string]bool{}
		for _, def := range models.AllPermissionDefs() {
			known[def.Key] = true
		}
		for _, key := range req.Keys {
			if !known[key] {
				return c.Status(400).JSON(fiber.Map{"message": "unknown permission: " + key})
			}
		}

		// Expansions are (method, path); the Casbin model stores (path, method).
		expanded := models.ExpandPermissionKeys(req.Keys)
		policies := make([][]string, 0, len(expanded))
		for _, p := range expanded {
			if len(p) >= 2 {
				policies = append(policies, []string{slug, p[1], p[0]})
			}
		}
		if _, err := e.RemoveFilteredPolicy(0, slug); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "internal error"})
		}
		if len(policies) > 0 {
			if _, err := e.AddPolicies(policies); err != nil {
				return c.Status(500).JSON(fiber.Map{"message": "internal error"})
			}
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

// MyPermissions reports the permission keys the caller's own role holds, so the
// UI can show or hide parts of a page. It only ever describes the caller, and
// hiding is a convenience: every API still enforces its own policy.
func MyPermissions(e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if role == "" {
			return c.Status(401).JSON(fiber.Map{"message": "unauthorized"})
		}
		raw := e.GetFilteredPolicy(0, role)
		for _, p := range raw {
			if len(p) >= 3 && p[1] == ".*" && p[2] == ".*" {
				return c.JSON(fiber.Map{"all": true, "granted": []string{}})
			}
		}
		granted := models.GetGrantedPermissionKeys(raw)
		if granted == nil {
			granted = []string{}
		}
		return c.JSON(fiber.Map{"all": false, "granted": granted})
	}
}
