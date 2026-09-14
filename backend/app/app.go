package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"go-nx-admin/internal/routes"
)

type Options struct {
	EmbedFS               fs.FS
	ExtraModels           []interface{}
	ExtraRoutes           func(api fiber.Router, db *gorm.DB, enforcer *casbin.Enforcer)
	AfterMigrate          func(db *gorm.DB)
	ExtraPermissions      []PermissionDef
	ExtraPermissionExpand func(string) [][]string
}

type PermissionDef = models.PermissionDef

func Run(opts Options) {
	cfg := config.Load()

	dirs := []string{"data", "logs", cfg.Upload.Dir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("mkdir %s error: %v", d, err)
		}
	}

	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("open log file error: ", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

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

	allowOrigins := cfg.CORS.Origins
	app.Use(cors.New(cors.Config{AllowOrigins: allowOrigins}))
	app.Use(fiberLog.New(fiberLog.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	models.ExtraPermissionDefs = opts.ExtraPermissions
	models.ExtraPermissionExpand = opts.ExtraPermissionExpand

	routes.Register(app, db, enforcer, opts.ExtraRoutes)
	models.AutoDiscoverAPIs(app, db)

	app.Static("/uploads", cfg.Upload.Dir)

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
