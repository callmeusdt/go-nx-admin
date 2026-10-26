package app

import "github.com/gofiber/fiber/v2"

// Actor is the identity set by the base authentication middleware for the
// current request. Business routes still require the base route protections.
type Actor struct {
	ID       uint
	Username string
	Role     string
	RoleID   uint
}

// CurrentActor reads the identity of an already authenticated request.
// Missing or malformed middleware values fail closed; request input is ignored.
func CurrentActor(c *fiber.Ctx) (Actor, bool) {
	if c == nil {
		return Actor{}, false
	}
	id, idOK := c.Locals("user_id").(uint)
	username, usernameOK := c.Locals("username").(string)
	role, roleOK := c.Locals("role").(string)
	roleID, roleIDOK := c.Locals("role_id").(uint)
	if !idOK || id == 0 || !usernameOK || username == "" || !roleOK || role == "" || !roleIDOK || roleID == 0 {
		return Actor{}, false
	}
	return Actor{ID: id, Username: username, Role: role, RoleID: roleID}, true
}
