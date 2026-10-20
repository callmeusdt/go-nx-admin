package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	CORS     CORSConfig     `yaml:"cors"`
	Login    LoginConfig    `yaml:"login"`
	Upload   UploadConfig   `yaml:"upload"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	Session  SessionConfig  `yaml:"session"`
	Cache    CacheConfig    `yaml:"cache"`
	Site     SiteConfig     `yaml:"site"`
}

// SiteConfig brands the served SPA shell. Empty fields fall back to the
// built-in title and bull favicon.
type SiteConfig struct {
	Title   string `yaml:"title"`
	Favicon string `yaml:"favicon"` // http(s)/data URL, or a local file path served by the backend
}

type CacheConfig struct {
	Driver string      `yaml:"driver"` // memory (default) / redis
	Redis  RedisConfig `yaml:"redis"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	Prefix   string `yaml:"prefix"`
}

// Cookie is opt-in; existing deployments retain their Bearer transport.
type SessionConfig struct {
	Cookie             bool   `yaml:"cookie"`
	Origin             string `yaml:"origin"`
	TrustLoopbackProxy bool   `yaml:"trust_loopback_proxy"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	// TrustedProxies lists the proxies (IPs or CIDRs) allowed to tell us the
	// client address. Empty means loopback only.
	TrustedProxies []string `yaml:"trusted_proxies"`
	// ClientIPHeader is the one header read when the peer is a trusted proxy:
	// X-Forwarded-For, X-Real-IP or CF-Connecting-IP. Empty ignores all headers.
	ClientIPHeader string `yaml:"client_ip_header"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	// Lifetimes in minutes. Zero falls back to the defaults below.
	AccessMinutes  int `yaml:"access_minutes"`  // default 30
	RefreshMinutes int `yaml:"refresh_minutes"` // default 10080 (7 days)
	MFAMinutes     int `yaml:"mfa_minutes"`     // default 5
}

const (
	DefaultAccessMinutes  = 30
	DefaultRefreshMinutes = 7 * 24 * 60
	DefaultMFAMinutes     = 5
)

func minutesOrDefault(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func (j JWTConfig) AccessTTL() time.Duration {
	return time.Duration(minutesOrDefault(j.AccessMinutes, DefaultAccessMinutes)) * time.Minute
}

func (j JWTConfig) RefreshTTL() time.Duration {
	return time.Duration(minutesOrDefault(j.RefreshMinutes, DefaultRefreshMinutes)) * time.Minute
}

func (j JWTConfig) MFATTL() time.Duration {
	return time.Duration(minutesOrDefault(j.MFAMinutes, DefaultMFAMinutes)) * time.Minute
}

type CORSConfig struct {
	Origins string `yaml:"origins"`
}

type LoginConfig struct {
	IPWhitelistEnabled bool            `yaml:"ip_whitelist_enabled"`
	RateLimit          RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled       bool `yaml:"enabled"`
	MaxAttempts   int  `yaml:"max_attempts"`
	WindowMinutes int  `yaml:"window_minutes"`
	LockMinutes   int  `yaml:"lock_minutes"`
}

type UploadConfig struct {
	Dir     string `yaml:"dir"`
	MaxSize int64  `yaml:"max_size"`
	// AllowTypes lists the permitted file extensions (no dot, lower case).
	// Empty uses DefaultUploadTypes. Active content such as html, svg and js is
	// never accepted by default because uploads are served from the app origin.
	AllowTypes []string `yaml:"allow_types"`
	Driver     string   `yaml:"driver"` // local (default) / s3
	S3         S3Config `yaml:"s3"`
}

// S3Config targets AWS S3 or any S3-compatible service (MinIO, OSS, COS, R2).
type S3Config struct {
	Endpoint      string `yaml:"endpoint"` // host[:port], no scheme
	Region        string `yaml:"region"`
	Bucket        string `yaml:"bucket"`
	AccessKey     string `yaml:"access_key"`
	SecretKey     string `yaml:"secret_key"`
	UseSSL        bool   `yaml:"use_ssl"`
	PathStyle     bool   `yaml:"path_style"`
	Prefix        string `yaml:"prefix"`
	PublicBaseURL string `yaml:"public_base_url"`
}

// DefaultUploadTypes are inert formats only.
var DefaultUploadTypes = []string{
	"jpg", "jpeg", "png", "gif", "webp", "bmp",
	"pdf", "txt", "csv", "md", "doc", "docx", "xls", "xlsx", "ppt", "pptx",
	"zip", "gz", "tar", "7z",
	"mp3", "wav", "mp4", "webm", "mov",
}

type CaptchaConfig struct {
	Enabled bool `yaml:"enabled"`
	Padding int  `yaml:"padding"`
}

var AppConfig *Config

// Load reads configuration from the default locations and the environment.
// Unlike earlier versions it reports malformed YAML or environment values
// instead of silently falling back to defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Server:   ServerConfig{Host: "127.0.0.1", Port: 19500, ClientIPHeader: "X-Forwarded-For"},
		Database: DatabaseConfig{Driver: "sqlite", DSN: "data/nx.db"},
		Login: LoginConfig{
			IPWhitelistEnabled: true,
			RateLimit: RateLimitConfig{
				Enabled: true, MaxAttempts: 5,
				WindowMinutes: 5, LockMinutes: 15,
			},
		},
		Upload:  UploadConfig{Dir: "./uploads", MaxSize: 10 * 1024 * 1024},
		Captcha: CaptchaConfig{Enabled: true, Padding: 5},
		Cache:   CacheConfig{Driver: "memory", Redis: RedisConfig{Addr: "127.0.0.1:6379", Prefix: "nx:"}},
	}

	paths := []string{"config.yaml"}
	if execPath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(execPath), "config.yaml"))
	}
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", p, err)
		}
		break
	}

	env := envReader{}
	env.str("NX_SERVER_HOST", &cfg.Server.Host)
	env.integer("NX_SERVER_PORT", &cfg.Server.Port)
	env.str("NX_DB_DRIVER", &cfg.Database.Driver)
	env.str("NX_DB_DSN", &cfg.Database.DSN)
	env.str("NX_JWT_SECRET", &cfg.JWT.Secret)
	env.integer("NX_JWT_ACCESS_MINUTES", &cfg.JWT.AccessMinutes)
	env.integer("NX_JWT_REFRESH_MINUTES", &cfg.JWT.RefreshMinutes)
	env.integer("NX_JWT_MFA_MINUTES", &cfg.JWT.MFAMinutes)
	env.str("NX_CORS_ORIGINS", &cfg.CORS.Origins)
	env.boolean("NX_LOGIN_IP_WHITELIST_ENABLED", &cfg.Login.IPWhitelistEnabled)
	env.boolean("NX_LOGIN_RATE_LIMIT_ENABLED", &cfg.Login.RateLimit.Enabled)
	env.integer("NX_LOGIN_RATE_LIMIT_MAX_ATTEMPTS", &cfg.Login.RateLimit.MaxAttempts)
	env.boolean("NX_CAPTCHA_ENABLED", &cfg.Captcha.Enabled)
	env.str("NX_UPLOAD_DIR", &cfg.Upload.Dir)
	env.str("NX_UPLOAD_DRIVER", &cfg.Upload.Driver)
	env.int64("NX_UPLOAD_MAX_SIZE", &cfg.Upload.MaxSize)
	if v := os.Getenv("NX_UPLOAD_ALLOW_TYPES"); v != "" {
		cfg.Upload.AllowTypes = nil
		for _, entry := range strings.Split(v, ",") {
			if entry = strings.ToLower(strings.Trim(strings.TrimSpace(entry), ".")); entry != "" {
				cfg.Upload.AllowTypes = append(cfg.Upload.AllowTypes, entry)
			}
		}
	}
	env.str("NX_S3_ENDPOINT", &cfg.Upload.S3.Endpoint)
	env.str("NX_S3_REGION", &cfg.Upload.S3.Region)
	env.str("NX_S3_BUCKET", &cfg.Upload.S3.Bucket)
	env.str("NX_S3_ACCESS_KEY", &cfg.Upload.S3.AccessKey)
	env.str("NX_S3_SECRET_KEY", &cfg.Upload.S3.SecretKey)
	env.boolean("NX_S3_USE_SSL", &cfg.Upload.S3.UseSSL)
	env.boolean("NX_S3_PATH_STYLE", &cfg.Upload.S3.PathStyle)
	env.str("NX_S3_PREFIX", &cfg.Upload.S3.Prefix)
	env.str("NX_S3_PUBLIC_BASE_URL", &cfg.Upload.S3.PublicBaseURL)
	if v := os.Getenv("NX_TRUSTED_PROXIES"); v != "" {
		cfg.Server.TrustedProxies = nil
		for _, entry := range strings.Split(v, ",") {
			if entry = strings.TrimSpace(entry); entry != "" {
				cfg.Server.TrustedProxies = append(cfg.Server.TrustedProxies, entry)
			}
		}
	}
	if v, ok := os.LookupEnv("NX_CLIENT_IP_HEADER"); ok {
		cfg.Server.ClientIPHeader = v
	}
	env.str("NX_SITE_TITLE", &cfg.Site.Title)
	env.str("NX_SITE_FAVICON", &cfg.Site.Favicon)
	env.str("NX_CACHE_DRIVER", &cfg.Cache.Driver)
	env.str("NX_REDIS_ADDR", &cfg.Cache.Redis.Addr)
	env.str("NX_REDIS_PASSWORD", &cfg.Cache.Redis.Password)
	env.integer("NX_REDIS_DB", &cfg.Cache.Redis.DB)
	if env.err != nil {
		return nil, env.err
	}

	AppConfig = cfg
	return cfg, nil
}

// envReader applies environment overrides and remembers the first malformed
// value, so a typo such as NX_SERVER_PORT=80a0 fails loudly.
type envReader struct{ err error }

func (e *envReader) str(name string, dst *string) {
	if v := os.Getenv(name); v != "" {
		*dst = v
	}
}

func (e *envReader) integer(name string, dst *int) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil && e.err == nil {
		e.err = fmt.Errorf("config: %s must be an integer", name)
		return
	}
	*dst = n
}

func (e *envReader) int64(name string, dst *int64) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil && e.err == nil {
		e.err = fmt.Errorf("config: %s must be an integer", name)
		return
	}
	*dst = n
}

// boolean accepts the usual spellings; anything else is an error rather than
// a silent "false" that could switch a security control off.
func (e *envReader) boolean(name string, dst *bool) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	b, err := strconv.ParseBool(v)
	if err != nil && e.err == nil {
		e.err = fmt.Errorf("config: %s must be true or false", name)
		return
	}
	*dst = b
}
