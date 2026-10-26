package app

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestCurrentActorReadsAuthenticatedLocalsOnly(t *testing.T) {
	valid := map[string]any{"user_id": uint(7), "username": "operator", "role": "admin", "role_id": uint(2)}
	tests := []struct {
		name   string
		locals map[string]any
		wantOK bool
	}{
		{name: "authenticated", locals: valid, wantOK: true},
		{name: "untrusted header only"},
		{name: "missing role", locals: map[string]any{"user_id": uint(7), "username": "operator", "role_id": uint(2)}},
		{name: "zero user id", locals: map[string]any{"user_id": uint(0), "username": "operator", "role": "admin", "role_id": uint(2)}},
		{name: "zero role id", locals: map[string]any{"user_id": uint(7), "username": "operator", "role": "admin", "role_id": uint(0)}},
		{name: "wrong user id type", locals: map[string]any{"user_id": 7, "username": "operator", "role": "admin", "role_id": uint(2)}},
		{name: "wrong role id type", locals: map[string]any{"user_id": uint(7), "username": "operator", "role": "admin", "role_id": 2}},
		{name: "wrong username type", locals: map[string]any{"user_id": uint(7), "username": []byte("operator"), "role": "admin", "role_id": uint(2)}},
		{name: "wrong role type", locals: map[string]any{"user_id": uint(7), "username": "operator", "role": []byte("admin"), "role_id": uint(2)}},
		{name: "empty username", locals: map[string]any{"user_id": uint(7), "username": "", "role": "admin", "role_id": uint(2)}},
		{name: "empty role", locals: map[string]any{"user_id": uint(7), "username": "operator", "role": "", "role_id": uint(2)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := fiber.New()
			var got Actor
			var ok bool
			server.Get("/actor", func(c *fiber.Ctx) error {
				for key, value := range tc.locals {
					c.Locals(key, value)
				}
				got, ok = CurrentActor(c)
				return c.SendStatus(204)
			})
			request := httptest.NewRequest("GET", "/actor", nil)
			request.Header.Set("X-User-ID", "999")
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want=%v", ok, tc.wantOK)
			}
			if ok && !reflect.DeepEqual(got, Actor{ID: 7, Username: "operator", Role: "admin", RoleID: 2}) {
				t.Fatalf("unexpected actor: %+v", got)
			}
			if !ok && got != (Actor{}) {
				t.Fatalf("rejected actor leaked fields: %+v", got)
			}
		})
	}
}
