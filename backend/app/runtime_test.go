package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		"database": func(o *RuntimeOptions) { o.DB = nil },
		"port":     func(o *RuntimeOptions) { o.Config.Server.Port = 0 },
		"host":     func(o *RuntimeOptions) { o.Config.Server.Host = "" },
		"key":      func(o *RuntimeOptions) { o.Config.JWT.Secret = "short" },
		"uploads":  func(o *RuntimeOptions) { o.PublicUploads = true },
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

func TestConfigLoadReportsBadInputAndValidates(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("CONFIG_PATH", "")
	for _, k := range []string{"NX_JWT_SECRET", "NX_JWT_SECRET_FILE", "NX_CORS_ORIGINS", "NX_SERVER_PORT", "NX_LOGIN_IP_WHITELIST_ENABLED"} {
		t.Setenv(k, "")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("an unresolved (empty) secret must not validate")
	}
	if cfg.Server.Host == "172.18.0.1" {
		t.Fatal("default host must not be a workstation-specific address")
	}
	// Unset secret + empty CORS is a valid setup once the secret is resolved.
	if err := cfg.ResolveJWTSecret(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty cors.origins must be allowed: %v", err)
	}

	// cors.origins is inert: any value must still load and validate, but warn.
	for _, v := range []string{"*", "not-an-origin", "https://admin.example.test"} {
		t.Setenv("NX_CORS_ORIGINS", v)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		cfg.ResolveJWTSecret()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("stale cors.origins=%q must not block startup: %v", v, err)
		}
		if len(cfg.Warnings()) == 0 {
			t.Fatalf("cors.origins=%q is ignored and must be reported", v)
		}
	}
	t.Setenv("NX_CORS_ORIGINS", "")

	t.Setenv("NX_JWT_SECRET", "short")
	cfg, _ = config.Load()
	if err := cfg.ResolveJWTSecret(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("an explicit but too-short secret must be rejected, not silently replaced")
	}
	t.Setenv("NX_JWT_SECRET", strings.Repeat("good-secret-", 3))
	cfg, err = config.Load()
	if err != nil || cfg.ResolveJWTSecret() != nil || cfg.Validate() != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if cfg.JWT.Secret != strings.Repeat("good-secret-", 3) {
		t.Fatal("explicit secret was replaced")
	}
	t.Setenv("NX_JWT_SECRET", "")

	for name, env := range map[string][2]string{
		"port":      {"NX_SERVER_PORT", "80a0"},
		"whitelist": {"NX_LOGIN_IP_WHITELIST_ENABLED", "flase"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env[0], env[1])
			if _, err := config.Load(); err == nil {
				t.Fatalf("%s=%q accepted", env[0], env[1])
			}
		})
	}
	t.Setenv("NX_SERVER_PORT", "")
	t.Setenv("NX_LOGIN_IP_WHITELIST_ENABLED", "1")
	if cfg, err := config.Load(); err != nil || !cfg.Login.IPWhitelistEnabled {
		t.Fatalf("'1' must parse as true: %v", err)
	}
	t.Setenv("NX_LOGIN_IP_WHITELIST_ENABLED", "")

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("server: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(); err == nil {
		t.Fatal("malformed YAML silently ignored")
	}
}

// The signing key is generated once and then reused, so restarts keep sessions.
func TestJWTSecretIsGeneratedOnceAndPersisted(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	load := func() *config.Config {
		cfg := &config.Config{}
		if err := cfg.ResolveJWTSecret(); err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	first := load()
	if len(first.JWT.Secret) < config.MinJWTSecretLen {
		t.Fatalf("generated key too short: %d", len(first.JWT.Secret))
	}
	info, err := os.Stat(filepath.Join(dir, "data", "jwt.secret"))
	if err != nil {
		t.Fatalf("key not persisted: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %v, want 0600", info.Mode().Perm())
	}
	if second := load(); second.JWT.Secret != first.JWT.Secret {
		t.Fatal("restart generated a different key, which would sign everyone out")
	}

	// A published default in an old config is ignored in favour of the file.
	old := &config.Config{JWT: config.JWTConfig{Secret: "nx-admin-jwt-secret-2024"}}
	if err := old.ResolveJWTSecret(); err != nil || old.JWT.Secret != first.JWT.Secret {
		t.Fatalf("published default not replaced by the persisted key: %v", err)
	}
	// An explicit secret is never overridden and never touches the file.
	explicit := &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("x", 40)}}
	if err := explicit.ResolveJWTSecret(); err != nil || explicit.JWT.Secret != strings.Repeat("x", 40) {
		t.Fatalf("explicit secret changed: %v", err)
	}

	// A corrupt file is an error, never silently overwritten.
	custom := filepath.Join(dir, "custom.secret")
	if err := os.WriteFile(custom, []byte("tiny\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := &config.Config{JWT: config.JWTConfig{SecretFile: custom}}
	if err := bad.ResolveJWTSecret(); err == nil {
		t.Fatal("unusable secret file accepted")
	}
	if b, _ := os.ReadFile(custom); string(b) != "tiny\n" {
		t.Fatal("unusable secret file was overwritten")
	}
}

func TestTokensNeedConfiguredKey(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	config.AppConfig = nil
	t.Setenv("NX_JWT_SECRET", strings.Repeat("env-must-not-be-used", 2))
	if _, err := middleware.GenerateToken(1, "u", "admin", 1); err == nil {
		t.Fatal("token signed without a configured key")
	}
	if _, err := middleware.ParseToken("a.b.c"); err == nil {
		t.Fatal("token parsed without a configured key")
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

// Run and NewRuntime must share one security floor.
func TestRuntimeUsesSharedConfigValidation(t *testing.T) {
	opts := runtimeOptions(t)
	opts.Config.JWT.Secret = "nx-admin-jwt-secret-2024"
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("published default secret accepted by NewRuntime")
	}
}

// CORS belongs to the reverse proxy. If the app also set Access-Control-*
// headers, the browser would see duplicated values ("*, *") and reject them,
// so the application must never emit any - whatever cors.origins says.
func TestApplicationNeverEmitsCORSHeaders(t *testing.T) {
	for name, origins := range map[string]string{"empty": "", "configured": "https://admin.example.test", "wildcard": "*"} {
		t.Run(name, func(t *testing.T) {
			h := newBearerHarness(t, func(o *RuntimeOptions) { o.Config.CORS.Origins = origins })
			for _, tc := range []struct{ method, path string }{
				{"GET", "/health"}, {"OPTIONS", "/api/v1/users"}, {"POST", "/api/v1/auth/login"}, {"GET", "/"},
			} {
				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.Header.Set("Origin", "https://evil.example")
				req.Header.Set("Access-Control-Request-Method", "GET")
				res, err := h.r.App.Test(req, 5000)
				if err != nil {
					t.Fatal(err)
				}
				for k := range res.Header {
					if strings.HasPrefix(strings.ToLower(k), "access-control-") {
						t.Fatalf("%s %s emitted %s: %v", tc.method, tc.path, k, res.Header[k])
					}
				}
				res.Body.Close()
			}
		})
	}
}
