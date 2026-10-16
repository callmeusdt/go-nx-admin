package cache

import "go-nx-admin/internal/config"

func cfgDriver(driver string) config.CacheConfig {
	return config.CacheConfig{Driver: driver}
}
