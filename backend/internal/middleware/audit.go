package middleware

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

// auditErrorMax matches the audit_logs.error column (size:255). Longer text
// makes MySQL and PostgreSQL reject the whole row, losing the audit record.
const auditErrorMax = 255

// auditStatus is the status the client will actually receive. When a handler
// returns an error, Fiber's error handler has not run yet at the point the
// audit row is built, so the response still shows the default 200.
func auditStatus(current int, err error) int {
	if err == nil {
		return current
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return fe.Code
	}
	return fiber.StatusInternalServerError
}

// auditErrorText returns a bounded, single-line description of err that is safe
// to store: control characters are dropped and the cut never splits a rune.
func auditErrorText(err error) string {
	if err == nil {
		return ""
	}
	text := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, err.Error())
	if len(text) <= auditErrorMax {
		return text
	}
	text = text[:auditErrorMax]
	for !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	return text
}
