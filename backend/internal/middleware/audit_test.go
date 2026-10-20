package middleware

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

func TestAuditStatusFollowsTheErrorHandler(t *testing.T) {
	for name, tc := range map[string]struct {
		current int
		err     error
		want    int
	}{
		"no error keeps the response status": {200, nil, 200},
		"handler status kept without error":  {404, nil, 404},
		"plain error becomes 500":            {200, errors.New("boom"), 500},
		"fiber error keeps its own code":     {200, fiber.NewError(418, "x"), 418},
		"wrapped fiber error":                {200, errors.Join(errors.New("ctx"), fiber.NewError(403, "no")), 403},
	} {
		if got := auditStatus(tc.current, tc.err); got != tc.want {
			t.Errorf("%s: got %d, want %d", name, got, tc.want)
		}
	}
}

func TestAuditErrorTextIsBoundedValidAndSingleLine(t *testing.T) {
	if auditErrorText(nil) != "" {
		t.Fatal("nil error must be empty")
	}
	if got := auditErrorText(errors.New("short")); got != "short" {
		t.Fatalf("short text altered: %q", got)
	}
	long := auditErrorText(errors.New(strings.Repeat("x", 600)))
	if len(long) != auditErrorMax {
		t.Fatalf("len = %d, want %d", len(long), auditErrorMax)
	}
	// 3-byte runes: a cut at byte 255 lands inside one and must back off.
	multi := auditErrorText(errors.New(strings.Repeat("错", 200)))
	if !utf8.ValidString(multi) || len(multi) > auditErrorMax {
		t.Fatalf("multi-byte cut broke the text: len=%d valid=%v", len(multi), utf8.ValidString(multi))
	}
	if got := auditErrorText(errors.New("a\nb\r\nc\x00d\te")); strings.ContainsAny(got, "\n\r\x00\t") {
		t.Fatalf("control characters survived: %q", got)
	}
}
