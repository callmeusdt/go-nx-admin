package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/handlers"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func runtimeOptions(t *testing.T) RuntimeOptions {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	var cfg Config
	cfg.Server.Host, cfg.Server.Port = "127.0.0.1", 19876
	cfg.JWT.Secret = strings.Repeat("test-runtime-key", 3)
	cfg.CORS.Origins = "https://admin.example.test"
	return RuntimeOptions{Config: cfg, DB: db, LogOutput: io.Discard, DisableMedia: true,
		Extensions: Options{EmbedFS: fstest.MapFS{"index.html": {Data: []byte("<html>BIM admin</html>")}}}}
}

func TestRuntimeRequiresExplicitMigration(t *testing.T) {
	opts := runtimeOptions(t)
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("empty schema accepted")
	}
	if opts.DB.Migrator().HasTable("admin_casbin_rule") {
		t.Fatal("runtime created policy table")
	}
	if opts.DB.Migrator().HasTable("admin_users") {
		t.Fatal("runtime created users table")
	}
}

func TestRuntimeLoadsWithoutSeedsAndCloses(t *testing.T) {
	opts := runtimeOptions(t)
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	// Schema initialization itself must not create a default login.
	var users, policies int64
	if err := opts.DB.Table("admin_users").Count(&users).Error; err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatal("migration seeded users")
	}
	opts.Extensions.AfterMigrate = func(*gorm.DB) { t.Fatal("serve invoked migration callback") }
	previous := config.AppConfig
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("second runtime accepted")
	}
	if err := opts.DB.Table("admin_casbin_rule").Count(&policies).Error; err != nil {
		t.Fatal(err)
	}
	if policies != 0 {
		t.Fatal("runtime seeded policy")
	}
	for path, want := range map[string]int{"/health": 200, "/bim/users": 200, "/missing.js": 404, "/api/unknown": 404, "/uploads/private.txt": 404, "/api/v1/auth/me": 401} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Accept", "text/html")
		res, err := r.App.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("%s: got %d want %d", path, res.StatusCode, want)
		}
	}
	for _, route := range r.App.GetRoutes() {
		if strings.HasPrefix(route.Path, "/api/v1/media") {
			t.Fatal("media route registered")
		}
	}
	// Explicit configuration, rather than an unrelated environment secret, signs JWTs.
	t.Setenv("NX_JWT_SECRET", "unrelated-environment-value")
	token, err := middleware.GenerateToken(1, "runtime-test", "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := middleware.ParseToken(token); err != nil {
		t.Fatal(err)
	}
	if _, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) { return []byte(opts.Config.JWT.Secret), nil }); err != nil {
		t.Fatal("explicit signing key was not used", err)
	}
	if _, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) { return []byte("unrelated-environment-value"), nil }); err == nil {
		t.Fatal("environment overrode explicit signing key")
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if config.AppConfig != previous {
		t.Fatal("configuration not restored")
	}
	r2, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := r2.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeRejectsUnsafeInputs(t *testing.T) {
	opts := runtimeOptions(t)
	for name, mutate := range map[string]func(*RuntimeOptions){
		"database":         func(o *RuntimeOptions) { o.DB = nil },
		"port":             func(o *RuntimeOptions) { o.Config.Server.Port = 0 },
		"host":             func(o *RuntimeOptions) { o.Config.Server.Host = "" },
		"key":              func(o *RuntimeOptions) { o.Config.JWT.Secret = "short" },
		"cors":             func(o *RuntimeOptions) { o.Config.CORS.Origins = "*" },
		"cors-host":        func(o *RuntimeOptions) { o.Config.CORS.Origins = "admin.example.test" },
		"cors-empty-entry": func(o *RuntimeOptions) { o.Config.CORS.Origins += "," },
		"uploads":          func(o *RuntimeOptions) { o.PublicUploads = true },
	} {
		t.Run(name, func(t *testing.T) {
			copy := opts
			mutate(&copy)
			if _, err := NewRuntime(copy); err == nil {
				t.Fatal("invalid options accepted")
			}
		})
	}
}

func TestRuntimeConstructorRestoresGlobalsOnExtensionPanic(t *testing.T) {
	opts := runtimeOptions(t)
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	previous := config.AppConfig
	opts.Extensions.ExtraRoutes = func(fiber.Router, *gorm.DB, *casbin.Enforcer) { panic("extension failed") }
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("extension failure accepted")
	}
	if config.AppConfig != previous {
		t.Fatal("failed constructor leaked settings")
	}
	opts.Extensions.ExtraRoutes = nil
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeWhitelistUsesExplicitConfig(t *testing.T) {
	opts := runtimeOptions(t)
	opts.Config.Login.IPWhitelistEnabled = true
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	if err := opts.DB.Create(&models.UserIPWhitelist{UserID: 42, IP: "192.0.2.1", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	t.Setenv("NX_LOGIN_IP_WHITELIST_ENABLED", "false")
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(context.Background())
	if ok, _ := handlers.CheckLoginIPWhitelist(opts.DB, 42, "192.0.2.2"); ok {
		t.Fatal("environment bypassed explicit whitelist")
	}
	if ok, _ := handlers.CheckLoginIPWhitelist(opts.DB, 42, "192.0.2.1"); !ok {
		t.Fatal("permitted address rejected")
	}
}

func TestLegacyConfigRetainsSecretEnvironmentAlias(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	t.Setenv("NX_JWT_SECRET", "")
	t.Setenv("NIUBI_JWT_SECRET", strings.Repeat("legacy-test", 4))
	cfg := config.Load()
	if cfg.JWT.Secret != strings.Repeat("legacy-test", 4) {
		t.Fatal("legacy key ignored")
	}
	t.Setenv("NX_JWT_SECRET", strings.Repeat("preferred-test", 4))
	if config.Load().JWT.Secret != strings.Repeat("preferred-test", 4) {
		t.Fatal("primary environment key did not win")
	}
}

func TestRuntimeShutdownTimeoutKeepsSingletonReserved(t *testing.T) {
	opts := runtimeOptions(t)
	opts.Extensions.EmbedFS = nil
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	r.App.Get("/slow", func(c *fiber.Ctx) error {
		close(entered)
		<-release
		return c.SendString("done")
	})
	go r.App.Listener(listener)
	go func() {
		defer close(finished)
		client := &http.Client{Timeout: 5 * time.Second}
		res, err := client.Get("http://" + listener.Addr().String() + "/slow")
		if err == nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request never entered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := r.Close(ctx); err == nil {
		t.Fatal("shutdown unexpectedly drained")
	}
	if err := r.Close(context.Background()); err == nil {
		t.Fatal("repeat close forgot failed drain")
	}
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("new runtime accepted with handler in flight")
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not finish")
	}
	// Test-only cleanup after proving the old request has ended. Production
	// deliberately retains its failed-close latch until process exit.
	r.closeErr = nil
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
