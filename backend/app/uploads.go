package app

import (
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// serveUploads mounts the upload directory with headers that stop a stored
// file from being interpreted as a page: no MIME sniffing, a sandbox CSP and,
// for everything that is not a plain image, a download disposition.
//
// Object keys start with "uploads/", so files live in <dir>/uploads and the
// stored URL "/uploads/..." maps onto that sub-directory. Mounting <dir> itself
// made every returned URL answer 404.
func serveUploads(app *fiber.App, dir string) {
	dir = filepath.Join(dir, "uploads")
	app.Use("/uploads", func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; media-src 'self'; style-src 'unsafe-inline'; sandbox")
		c.Set("Cross-Origin-Resource-Policy", "same-origin")
		return c.Next()
	})
	app.Static("/uploads", dir, fiber.Static{
		Download: false,
		ModifyResponse: func(c *fiber.Ctx) error {
			if t := string(c.Response().Header.ContentType()); !isInlineSafe(t) {
				c.Set("Content-Disposition", "attachment")
			}
			return nil
		},
	})
}

func isInlineSafe(contentType string) bool {
	for _, p := range []string{"image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "video/", "audio/", "application/pdf"} {
		if len(contentType) >= len(p) && contentType[:len(p)] == p {
			return true
		}
	}
	return false
}
