package middleware

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"go-nx-admin/internal/config"
)

func TestClientIP(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	app := fiber.New()
	cloudflare := []string{"127.0.0.1", "203.0.113.0/24"}
	for _, tc := range []struct {
		name, peer, header, value, want string
		trusted                         []string
	}{
		{"direct peer ignores forged XFF", "198.51.100.7", "X-Forwarded-For", "1.2.3.4", "198.51.100.7", nil},
		{"direct peer ignores forged CF header", "198.51.100.7", "CF-Connecting-IP", "1.2.3.4", "198.51.100.7", nil},
		{"loopback proxy, X-Real-IP", "127.0.0.1", "X-Real-IP", "198.51.100.8", "198.51.100.8", nil},
		{"loopback proxy, CF-Connecting-IP", "127.0.0.1", "CF-Connecting-IP", "198.51.100.8", "198.51.100.8", nil},
		{"loopback proxy, XFF single", "127.0.0.1", "X-Forwarded-For", "198.51.100.8", "198.51.100.8", nil},
		{"XFF client-supplied prefix is not trusted", "127.0.0.1", "X-Forwarded-For", "9.9.9.9, 198.51.100.8", "198.51.100.8", nil},
		{"XFF skips trusted hops from the right", "127.0.0.1", "X-Forwarded-For", "9.9.9.9, 198.51.100.8, 203.0.113.5", "198.51.100.8", cloudflare},
		{"XFF all trusted falls to leftmost", "127.0.0.1", "X-Forwarded-For", "203.0.113.9, 203.0.113.5", "203.0.113.9", cloudflare},
		{"configured CIDR proxy", "172.18.0.2", "X-Real-IP", "198.51.100.8", "198.51.100.8", []string{"172.18.0.0/16"}},
		{"bare IP entry", "172.18.0.2", "X-Real-IP", "198.51.100.8", "198.51.100.8", []string{"172.18.0.2"}},
		{"configured list replaces loopback default", "127.0.0.1", "X-Real-IP", "198.51.100.8", "127.0.0.1", []string{"172.18.0.0/16"}},
		{"garbage header falls back to peer", "127.0.0.1", "X-Real-IP", "not-an-ip", "127.0.0.1", nil},
		{"garbage XFF hop falls back to peer", "127.0.0.1", "X-Forwarded-For", "198.51.100.8, bogus", "127.0.0.1", nil},
		{"missing header falls back to peer", "127.0.0.1", "X-Real-IP", "", "127.0.0.1", nil},
		{"ipv6 loopback proxy", "::1", "X-Real-IP", "2001:db8::1", "2001:db8::1", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.AppConfig = &config.Config{Server: config.ServerConfig{ClientIPHeader: tc.header, TrustedProxies: tc.trusted}}
			var request fasthttp.RequestCtx
			request.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(tc.peer), Port: 1234})
			if tc.value != "" {
				request.Request.Header.Set(tc.header, tc.value)
			}
			ctx := app.AcquireCtx(&request)
			defer app.ReleaseCtx(ctx)
			if got := ClientIP(ctx); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}

	t.Run("only the configured header is read", func(t *testing.T) {
		config.AppConfig = &config.Config{Server: config.ServerConfig{ClientIPHeader: "X-Real-IP"}}
		var request fasthttp.RequestCtx
		request.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})
		request.Request.Header.Set("CF-Connecting-IP", "1.2.3.4")
		request.Request.Header.Set("X-Forwarded-For", "1.2.3.4")
		ctx := app.AcquireCtx(&request)
		defer app.ReleaseCtx(ctx)
		if got := ClientIP(ctx); got != "127.0.0.1" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("empty header setting ignores all headers", func(t *testing.T) {
		config.AppConfig = &config.Config{}
		var request fasthttp.RequestCtx
		request.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})
		request.Request.Header.Set("X-Forwarded-For", "1.2.3.4")
		ctx := app.AcquireCtx(&request)
		defer app.ReleaseCtx(ctx)
		if got := ClientIP(ctx); got != "127.0.0.1" {
			t.Fatalf("got %q", got)
		}
	})
}
