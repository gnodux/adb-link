package config

import (
	"os"
	"path/filepath"
)

// Settings holds application configuration loaded from env vars.
type Settings struct {
	ConfigDir     string
	APIHost       string
	APIPort       int
	LogLevel      string
	LogDir        string
	Reload        bool
	AsyncQueryTTL int
}

// DefaultSettings returns settings with defaults, overridden by env vars.
func DefaultSettings() *Settings {
	s := &Settings{
		ConfigDir:     envOrDefault("ADB_LINK_CONFIG_DIR", "config"),
		APIHost:       envOrDefault("ADB_LINK_API_HOST", "0.0.0.0"),
		APIPort:       envOrDefaultInt("ADB_LINK_API_PORT", 8000),
		LogLevel:      envOrDefault("ADB_LINK_LOG_LEVEL", "INFO"),
		LogDir:        envOrDefault("ADB_LINK_LOG_DIR", "logs"),
		Reload:        false,
		AsyncQueryTTL: envOrDefaultInt("ADB_LINK_ASYNC_QUERY_TTL", 3600),
	}
	// Make config dir absolute if relative
	if !filepath.IsAbs(s.ConfigDir) {
		if wd, err := os.Getwd(); err == nil {
			s.ConfigDir = filepath.Join(wd, s.ConfigDir)
		}
	}
	return s
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	var n int
	for _, c := range v {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return defaultVal
		}
	}
	return n
}
