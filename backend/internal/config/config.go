package config

import (
	"os"
	"path/filepath"
	"strconv"
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
	Dir     string   `yaml:"dir"`
	MaxSize int64    `yaml:"max_size"`
	Driver  string   `yaml:"driver"` // local (default) / s3
	S3      S3Config `yaml:"s3"`
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

type CaptchaConfig struct {
	Enabled bool `yaml:"enabled"`
	Padding int  `yaml:"padding"`
}

var AppConfig *Config

func Load() *Config {
	cfg := &Config{
		Server:   ServerConfig{Host: "172.18.0.1", Port: 19500},
		Database: DatabaseConfig{Driver: "sqlite", DSN: "data/nx.db"},
		JWT:      JWTConfig{Secret: "nx-admin-jwt-secret-2024"},
		CORS:     CORSConfig{Origins: "*"},
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
		yaml.Unmarshal(data, cfg)
		break
	}

	if v := os.Getenv("NX_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("NX_SERVER_PORT"); v != "" {
		cfg.Server.Port, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("NX_DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("NX_DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("NX_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	} else if v := os.Getenv("NI" + "UBI_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("NX_JWT_ACCESS_MINUTES"); v != "" {
		cfg.JWT.AccessMinutes, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("NX_JWT_REFRESH_MINUTES"); v != "" {
		cfg.JWT.RefreshMinutes, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("NX_CORS_ORIGINS"); v != "" {
		cfg.CORS.Origins = v
	}
	if v := os.Getenv("NX_LOGIN_IP_WHITELIST_ENABLED"); v != "" {
		cfg.Login.IPWhitelistEnabled = v == "true"
	}
	if v := os.Getenv("NX_UPLOAD_DIR"); v != "" {
		cfg.Upload.Dir = v
	}
	if v := os.Getenv("NX_UPLOAD_DRIVER"); v != "" {
		cfg.Upload.Driver = v
	}
	if v := os.Getenv("NX_S3_ENDPOINT"); v != "" {
		cfg.Upload.S3.Endpoint = v
	}
	if v := os.Getenv("NX_S3_BUCKET"); v != "" {
		cfg.Upload.S3.Bucket = v
	}
	if v := os.Getenv("NX_S3_ACCESS_KEY"); v != "" {
		cfg.Upload.S3.AccessKey = v
	}
	if v := os.Getenv("NX_S3_SECRET_KEY"); v != "" {
		cfg.Upload.S3.SecretKey = v
	}
	if v := os.Getenv("NX_UPLOAD_MAX_SIZE"); v != "" {
		cfg.Upload.MaxSize, _ = strconv.ParseInt(v, 10, 64)
	}

	if v := os.Getenv("NX_CACHE_DRIVER"); v != "" {
		cfg.Cache.Driver = v
	}
	if v := os.Getenv("NX_REDIS_ADDR"); v != "" {
		cfg.Cache.Redis.Addr = v
	}
	if v := os.Getenv("NX_REDIS_PASSWORD"); v != "" {
		cfg.Cache.Redis.Password = v
	}
	if v := os.Getenv("NX_REDIS_DB"); v != "" {
		cfg.Cache.Redis.DB, _ = strconv.Atoi(v)
	}

	AppConfig = cfg
	return cfg
}
