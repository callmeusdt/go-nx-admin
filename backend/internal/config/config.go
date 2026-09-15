package config

import (
	"os"
	"path/filepath"
	"strconv"

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
	if v := os.Getenv("NX_CORS_ORIGINS"); v != "" {
		cfg.CORS.Origins = v
	}
	if v := os.Getenv("NX_LOGIN_IP_WHITELIST_ENABLED"); v != "" {
		cfg.Login.IPWhitelistEnabled = v == "true"
	}
	if v := os.Getenv("NX_UPLOAD_DIR"); v != "" {
		cfg.Upload.Dir = v
	}
	if v := os.Getenv("NX_UPLOAD_MAX_SIZE"); v != "" {
		cfg.Upload.MaxSize, _ = strconv.ParseInt(v, 10, 64)
	}

	AppConfig = cfg
	return cfg
}
