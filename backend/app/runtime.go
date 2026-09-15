package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLog "github.com/gofiber/fiber/v2/middleware/logger"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"go-nx-admin/internal/routes"
)

// Config is the public configuration surface for explicit runtime construction.
type Config = config.Config

// RuntimeOptions never reads environment/config files, opens a database, migrates,
// seeds users, or discovers APIs. The caller owns the supplied database pool.
type RuntimeOptions struct {
	Config        Config
	DB            *gorm.DB
	Extensions    Options
	LogOutput     io.Writer
	DisableMedia  bool
	PublicUploads bool
}

// Runtime owns the HTTP listener, not the caller's database or signal handling.
// The legacy handlers retain process-wide settings: only one explicit runtime
// may exist per process, and it must not be combined with Run.
type Runtime struct {
	App                 *fiber.App
	address             string
	previousConfig      *config.Config
	previousPermissions []PermissionDef
	previousExpand      func(string) [][]string
	closeMu             sync.Mutex
	closeErr            error
	once                sync.Once
}

var runtimeGuard struct {
	sync.Mutex
	active bool
}

func NewRuntime(opts RuntimeOptions) (result *Runtime, resultErr error) {
	if opts.DB == nil {
		return nil, errors.New("app: database is required")
	}
	if opts.Config.Server.Host == "" || opts.Config.Server.Port < 1 || opts.Config.Server.Port > 65535 {
		return nil, errors.New("app: explicit host and valid port are required")
	}
	if len(opts.Config.JWT.Secret) < 32 {
		return nil, errors.New("app: JWT secret must contain at least 32 bytes")
	}
	if opts.Config.CORS.Origins == "" || strings.Contains(opts.Config.CORS.Origins, "*") {
		return nil, errors.New("app: explicit CORS origins are required")
	}
	for _, origin := range strings.Split(opts.Config.CORS.Origins, ",") {
		u, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("app: invalid CORS origin")
		}
	}
	if opts.PublicUploads && (opts.DisableMedia || opts.Config.Upload.Dir == "") {
		return nil, errors.New("app: public uploads require media and an explicit directory")
	}
	runtimeGuard.Lock()
	defer runtimeGuard.Unlock()
	if runtimeGuard.active {
		return nil, errors.New("app: only one runtime per process is supported")
	}
	enforcer, err := middleware.LoadCasbin(opts.DB)
	if err != nil {
		return nil, fmt.Errorf("app: load existing policy: %w", err)
	}
	output := opts.LogOutput
	if output == nil {
		output = os.Stdout
	}
	r := &Runtime{previousConfig: config.AppConfig,
		previousPermissions: models.ExtraPermissionDefs, previousExpand: models.ExtraPermissionExpand,
		address: net.JoinHostPort(opts.Config.Server.Host, strconv.Itoa(opts.Config.Server.Port))}
	defer func() {
		if recover() != nil {
			config.AppConfig = r.previousConfig
			models.ExtraPermissionDefs = r.previousPermissions
			models.ExtraPermissionExpand = r.previousExpand
			result, resultErr = nil, errors.New("app: runtime construction failed")
		}
	}()
	r.App = fiber.New(fiber.Config{DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				status = fe.Code
			}
			return c.Status(status).JSON(fiber.Map{"message": http.StatusText(status)})
		}})
	config.AppConfig = &opts.Config
	models.ExtraPermissionDefs = opts.Extensions.ExtraPermissions
	models.ExtraPermissionExpand = opts.Extensions.ExtraPermissionExpand
	r.App.Use(fiberRecover.New())
	r.App.Use(cors.New(cors.Config{AllowOrigins: opts.Config.CORS.Origins}))
	r.App.Use(fiberLog.New(fiberLog.Config{Output: output, Format: "${status} ${method} ${path}\n"}))
	r.App.Get("/health", func(c *fiber.Ctx) error {
		db, err := opts.DB.DB()
		if err != nil || db.PingContext(c.UserContext()) != nil {
			return c.Status(503).JSON(fiber.Map{"status": "unavailable"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})
	routes.RegisterWithOptions(r.App, opts.DB, enforcer, opts.Extensions.ExtraRoutes, opts.DisableMedia)
	if opts.PublicUploads {
		r.App.Static("/uploads", opts.Config.Upload.Dir)
	}
	r.App.Use(func(c *fiber.Ctx) error {
		p := c.Path()
		if p == "/api" || strings.HasPrefix(p, "/api/") || p == "/uploads" || strings.HasPrefix(p, "/uploads/") {
			return c.Status(404).JSON(fiber.Map{"message": "Not Found"})
		}
		return c.Next()
	})
	if opts.Extensions.EmbedFS != nil {
		r.App.Use(adaptor.HTTPHandler(StaticHandler(opts.Extensions.EmbedFS)))
	}
	runtimeGuard.active = true
	return r, nil
}

func (r *Runtime) Listen() error { return r.App.Listen(r.address) }

func (r *Runtime) Close(ctx context.Context) error {
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	// fasthttp forgets listeners after a timed-out shutdown, so a subsequent
	// nil result cannot prove old handlers drained. Keep globals reserved until
	// process exit in this exceptional case rather than permit unsafe reuse.
	if r.closeErr != nil {
		return r.closeErr
	}
	err := r.App.ShutdownWithContext(ctx)
	if err != nil {
		r.closeErr = err
		return err
	}
	r.once.Do(func() {
		runtimeGuard.Lock()
		defer runtimeGuard.Unlock()
		config.AppConfig = r.previousConfig
		models.ExtraPermissionDefs = r.previousPermissions
		models.ExtraPermissionExpand = r.previousExpand
		runtimeGuard.active = false
	})
	return nil
}

// MigrateSchema is an explicit maintenance operation. It creates no default users.
func MigrateSchema(db *gorm.DB, extraModels ...interface{}) error {
	if db == nil {
		return errors.New("app: database is required")
	}
	if err := models.AutoMigrate(db, extraModels...); err != nil {
		return err
	}
	// The policy table belongs to the schema, but policy seeds remain explicit.
	return db.Table("admin_casbin_rule").AutoMigrate(&gormadapter.CasbinRule{})
}
