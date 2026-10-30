package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/cache"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
)

type Options struct {
	EmbedFS               fs.FS
	ExtraModels           []interface{}
	ExtraRoutes           func(api fiber.Router, db *gorm.DB, enforcer *casbin.Enforcer)
	ExtraPublicRoutes     func(app *fiber.App, db *gorm.DB, enforcer *casbin.Enforcer)
	AfterMigrate          func(db *gorm.DB)
	ExtraPermissions      []PermissionDef
	ExtraPermissionExpand func(string) [][]string
	// DashboardStats adds or overrides figures returned by GET /dashboard/stats.
	// Keys are merged over the built-in ones; it runs under dashboard:view.
	DashboardStats func(db *gorm.DB) map[string]any
}

type PermissionDef = models.PermissionDef

// RequiredDirs are the directories Run writes into: data (SQLite file and the
// generated JWT key), logs, uploads, plus the parent of a custom SQLite path
// or secret file. A fresh deployment has none of them.
func RequiredDirs(cfg *config.Config) []string {
	dirs := []string{"data", "logs", cfg.Upload.Dir}
	if cfg.Database.Driver == "" || cfg.Database.Driver == "sqlite" {
		if d := filepath.Dir(sqlitePath(cfg.Database.DSN)); d != "" && d != "." {
			dirs = append(dirs, d)
		}
	}
	if cfg.JWT.SecretFile != "" {
		if d := filepath.Dir(cfg.JWT.SecretFile); d != "." {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// sqlitePath strips the "file:" prefix and "?query" that a SQLite DSN may carry.
func sqlitePath(dsn string) string {
	dsn = strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		dsn = dsn[:i]
	}
	return dsn
}

// Run is the process entry point: it blocks until SIGINT/SIGTERM and exits the
// process on a startup failure. Use RunE to receive the error instead.
func Run(opts Options) {
	if err := RunE(opts); err != nil {
		log.Fatal(err)
	}
}

// RunE loads configuration, prepares the database (migrate, seed, policies) and
// serves until SIGINT/SIGTERM. Beyond that start-up work it is NewRuntime with
// the settings a standalone server wants, so both entry points share one HTTP
// stack (error handler, /health, logging, static files, single-instance guard).
// Every failure is returned; nothing in here terminates the process.
func RunE(opts Options) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return runWith(ctx, opts)
}

func runWith(ctx context.Context, opts Options) (err error) {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Everything the process persists lives under these directories; create
	// them before anything (the JWT key, SQLite, logs, uploads) needs to write.
	for _, d := range RequiredDirs(cfg) {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s error: %w", d, err)
		}
	}
	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file error: %w", err)
	}
	defer logFile.Close()
	logOutput := io.MultiWriter(os.Stdout, logFile)
	previousLog := log.Writer()
	log.SetOutput(logOutput)
	defer log.SetOutput(previousLog)

	if err := cfg.ResolveJWTSecret(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	for _, w := range cfg.Warnings() {
		log.Printf("config: %s", w)
	}
	if cfg.Session.Cookie {
		return errors.New("cookie sessions require app.NewRuntime with explicit migration and bootstrap")
	}

	cacheStore, err := cache.New(cfg.Cache)
	if err != nil {
		return fmt.Errorf("cache init error: %w", err)
	}
	defer cacheStore.Close()

	db, err := OpenDatabase(cfg)
	if err != nil {
		return fmt.Errorf("db connect error: %w", err)
	}
	if pool, perr := db.DB(); perr == nil {
		defer pool.Close()
	}
	if err := models.AutoMigrate(db, opts.ExtraModels...); err != nil {
		return fmt.Errorf("migrate error: %w", err)
	}
	if err := models.Seed(db); err != nil {
		return fmt.Errorf("seed error: %w", err)
	}
	if opts.AfterMigrate != nil {
		opts.AfterMigrate(db)
	}
	// Seeds the admin policy and upgrades legacy rows; the runtime then loads
	// the stored policy without writing.
	if _, err := middleware.InitCasbin(db); err != nil {
		return fmt.Errorf("casbin init error: %w", err)
	}

	rt, err := NewRuntime(RuntimeOptions{
		Config:        *cfg,
		DB:            db,
		Extensions:    opts,
		LogOutput:     logOutput,
		Cache:         cacheStore,
		PublicUploads: true,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if cerr := rt.Close(shutdownCtx); cerr != nil && err == nil {
			err = cerr
		}
	}()
	models.AutoDiscoverAPIs(rt.App, db)

	listenErr := make(chan error, 1)
	go func() { listenErr <- rt.Listen() }()
	log.Printf("server starting on %s", rt.address)
	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
		log.Println("shutting down...")
		return nil
	}
}
