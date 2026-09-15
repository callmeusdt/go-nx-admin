package middleware

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"go-nx-admin/internal/config"
)

func TestBrowserIPTrustBoundary(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	config.AppConfig = &config.Config{}
	app := fiber.New()
	for _, tc := range []struct {
		name, peer, header, want string
		trust                    bool
	}{
		{"untrusted loopback", "127.0.0.1", "198.51.100.8", "127.0.0.1", false},
		{"trusted proxy", "127.0.0.1", "198.51.100.8", "198.51.100.8", true},
		{"remote cannot forge", "192.0.2.5", "198.51.100.8", "192.0.2.5", true},
		{"invalid header", "127.0.0.1", "198.51.100.8, 192.0.2.5", "127.0.0.1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.AppConfig.Session.TrustLoopbackProxy = tc.trust
			var request fasthttp.RequestCtx
			request.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(tc.peer), Port: 1234})
			request.Request.Header.Set("X-BIM-Client-IP", tc.header)
			request.Request.Header.Set("CF-Connecting-IP", "203.0.113.9")
			request.Request.Header.Set("X-Forwarded-For", "203.0.113.9")
			ctx := app.AcquireCtx(&request)
			defer app.ReleaseCtx(ctx)
			if got := BrowserIP(ctx); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
