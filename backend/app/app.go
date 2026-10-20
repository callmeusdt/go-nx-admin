package app

import (
	"context"
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
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	fiberLog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go-nx-admin/internal/cache"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/handlers"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"go-nx-admin/internal/routes"
	"go-nx-admin/internal/storage"
)

type Options struct {
	EmbedFS               fs.FS
	ExtraModels           []interface{}
	ExtraRoutes           func(api fiber.Router, db *gorm.DB, enforcer *casbin.Enforcer)
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

func Run(opts Options) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	// Everything the process persists lives under these directories; create
	// them before anything (the JWT key, SQLite, logs, uploads) needs to write.
	for _, d := range RequiredDirs(cfg) {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("mkdir %s error: %v", d, err)
		}
	}
	if err := cfg.ResolveJWTSecret(); err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	for _, w := range cfg.Warnings() {
		log.Printf("config: %s", w)
	}
	if cfg.Session.Cookie {
		log.Fatal("cookie sessions require app.NewRuntime with explicit migration and bootstrap")
	}

	if err := cache.Init(cfg.Cache); err != nil {
		log.Fatal("cache init error: ", err)
	}

	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("open log file error: ", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	files, err := storage.NewFromConfig(cfg.Upload, true)
	if err != nil {
		log.Fatal("storage init error: ", err)
	}
	storage.Set(files)

	var db *gorm.DB
	switch cfg.Database.Driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{})
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	default:
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
	}
	if err != nil {
		log.Fatal("db connect error: ", err)
	}
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}

	if err := models.AutoMigrate(db, opts.ExtraModels...); err != nil {
		log.Fatal("migrate error: ", err)
	}
	if err := models.Seed(db); err != nil {
		log.Fatal("seed error: ", err)
	}
	if opts.AfterMigrate != nil {
		opts.AfterMigrate(db)
	}

	enforcer, err := middleware.InitCasbin(db)
	if err != nil {
		log.Fatal("casbin init error: ", err)
	}

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"message": err.Error()})
		},
	})
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		dbStatus := "connected"
		if sqlDB != nil && sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}
		return c.JSON(fiber.Map{
			"status": "ok",
			"db":     dbStatus,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	app.Use(fiberLog.New(fiberLog.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	models.ExtraPermissionDefs = opts.ExtraPermissions
	models.ExtraPermissionExpand = opts.ExtraPermissionExpand
	handlers.ExtraDashboardStats = opts.DashboardStats

	routes.Register(app, db, enforcer, opts.ExtraRoutes)
	models.AutoDiscoverAPIs(app, db)

	serveUploads(app, cfg.Upload.Dir)

	if opts.EmbedFS != nil {
		app.Use("/*", adaptor.HTTPHandler(StaticHandler(opts.EmbedFS)))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("server starting on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatal(err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	app.ShutdownWithContext(ctx)
}
