package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"go-nx-admin/internal/config"
)

// RunE must return startup failures; library code that calls log.Fatal cannot
// be embedded or tested.
func TestRunEReturnsStartupErrors(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	t.Setenv("CONFIG_PATH", "")
	for _, k := range []string{"NX_JWT_SECRET", "NX_JWT_SECRET_FILE", "NX_DB_DRIVER", "NX_DB_DSN", "NX_SERVER_PORT"} {
		t.Setenv(k, "")
	}

	t.Run("malformed config", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("server: [unclosed"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunE(Options{}); err == nil {
			t.Fatal("malformed config returned no error")
		}
	})
	t.Run("cookie mode needs NewRuntime", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		cfg := "session:\n  cookie: true\n  origin: https://admin.example.test\n"
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunE(Options{}); err == nil || !strings.Contains(err.Error(), "NewRuntime") {
			t.Fatalf("expected a NewRuntime hint, got %v", err)
		}
	})
	t.Run("unusable database", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		cfg := "database:\n  driver: mysql\n  dsn: \"nobody@tcp(127.0.0.1:1)/none?timeout=1s\"\n"
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RunE(Options{}); err == nil {
			t.Fatal("unreachable database returned no error")
		}
	})
}

// A full start/stop cycle on a random free port; shutdown must release the
// single-runtime guard so the process can start another one.
func TestRunEServesAndStopsOnContextCancel(t *testing.T) {
	previous := config.AppConfig
	defer func() { config.AppConfig = previous }()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("CONFIG_PATH", "")
	for _, k := range []string{"NX_JWT_SECRET", "NX_JWT_SECRET_FILE", "NX_DB_DRIVER", "NX_DB_DSN"} {
		t.Setenv(k, "")
	}
	t.Setenv("NX_SERVER_PORT", "39517")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runWith(ctx, Options{}) }()

	var healthy bool
	for i := 0; i < 100 && !healthy; i++ {
		time.Sleep(50 * time.Millisecond)
		healthy = probeHealth("http://127.0.0.1:39517/health")
	}
	if !healthy {
		cancel()
		t.Fatalf("server never became healthy: %v", <-done)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("clean shutdown returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunE did not stop after the context was cancelled")
	}
	runtimeGuard.Lock()
	active := runtimeGuard.active
	runtimeGuard.Unlock()
	if active {
		t.Fatal("shutdown left the single-runtime guard held")
	}
	for _, name := range []string{"data/jwt.secret", "data/nx.db", "logs/app.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s after a fresh start: %v", name, err)
		}
	}
}

// Server errors must not echo SQL, paths or driver messages to the client.
func TestErrorHandlerHidesInternalDetails(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/boom", func(c *fiber.Ctx) error {
		return errors.New(`SELECT * FROM admin_users WHERE password='x': no such table`)
	})
	app.Get("/teapot", func(c *fiber.Ctx) error { return fiber.NewError(418, "short and stout") })
	for path, want := range map[string]int{"/boom": 500, "/teapot": 418} {
		res, err := app.Test(httptest.NewRequest("GET", path, nil), 5000)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("%s: status %d, want %d", path, res.StatusCode, want)
		}
		if strings.Contains(string(body), "admin_users") || strings.Contains(string(body), "stout") {
			t.Fatalf("%s leaked internal detail: %s", path, body)
		}
	}
}

func probeHealth(url string) bool {
	client := &http.Client{Timeout: time.Second}
	res, err := client.Get(url)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == 200
}
