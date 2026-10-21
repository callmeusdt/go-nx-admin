package handlers

import (
	"errors"
	"log"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// dbError answers a failed database operation. Driver text carries table and
// column names, constraint names and sometimes statement fragments, so it is
// logged for the operator and never sent to the client. Expected conflicts get
// a stable short message instead of a generic failure.
//
// app.OpenDatabase turns on gorm.Config.TranslateError. A caller-supplied
// *gorm.DB (NewRuntime) may not have, so the common driver codes are matched as
// a fallback and the result is the same either way.
func dbError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueViolation(err):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "already exists"})
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "referenced by other records"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "not found"})
	}
	return internalError(c, err)
}

// internalError logs the cause and returns a generic 500.
func internalError(c *fiber.Ctx, err error) error {
	log.Printf("%s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "internal error"})
}

// isUniqueViolation recognises a duplicate-key failure from the raw driver
// error for MySQL (1062), PostgreSQL (23505) and SQLite (UNIQUE constraint).
func isUniqueViolation(err error) bool {
	var my *mysql.MySQLError
	if errors.As(err, &my) {
		return my.Number == 1062
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.Code == "23505"
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "constraint failed: UNIQUE")
}
