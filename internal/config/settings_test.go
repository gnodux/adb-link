package config

import (
	"os"
	"testing"
)

func TestDefaultSettings_AllDefaults(t *testing.T) {
	// Clear all env vars that could affect settings
	for _, key := range []string{
		"ADB_LINK_CONFIG_DIR", "ADB_LINK_API_HOST", "ADB_LINK_API_PORT",
		"ADB_LINK_LOG_LEVEL", "ADB_LINK_LOG_DIR", "ADB_LINK_ASYNC_QUERY_TTL",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	s := DefaultSettings()
	if s.APIHost != "0.0.0.0" {
		t.Errorf("APIHost = %q, want %q", s.APIHost, "0.0.0.0")
	}
	if s.APIPort != 8000 {
		t.Errorf("APIPort = %d, want %d", s.APIPort, 8000)
	}
	if s.LogLevel != "INFO" {
		t.Errorf("LogLevel = %q, want %q", s.LogLevel, "INFO")
	}
	if s.AsyncQueryTTL != 3600 {
		t.Errorf("AsyncQueryTTL = %d, want %d", s.AsyncQueryTTL, 3600)
	}
}

func TestDefaultSettings_EnvOverride_ConfigDir(t *testing.T) {
	t.Setenv("ADB_LINK_CONFIG_DIR", "/custom/config")
	s := DefaultSettings()
	if s.ConfigDir != "/custom/config" {
		t.Errorf("ConfigDir = %q, want %q", s.ConfigDir, "/custom/config")
	}
}

func TestDefaultSettings_EnvOverride_APIPort(t *testing.T) {
	t.Setenv("ADB_LINK_API_PORT", "9090")
	s := DefaultSettings()
	if s.APIPort != 9090 {
		t.Errorf("APIPort = %d, want %d", s.APIPort, 9090)
	}
}

func TestDefaultSettings_EnvOverride_APIHost(t *testing.T) {
	t.Setenv("ADB_LINK_API_HOST", "127.0.0.1")
	s := DefaultSettings()
	if s.APIHost != "127.0.0.1" {
		t.Errorf("APIHost = %q, want %q", s.APIHost, "127.0.0.1")
	}
}

func TestDefaultSettings_EnvOverride_LogLevel(t *testing.T) {
	t.Setenv("ADB_LINK_LOG_LEVEL", "DEBUG")
	s := DefaultSettings()
	if s.LogLevel != "DEBUG" {
		t.Errorf("LogLevel = %q, want %q", s.LogLevel, "DEBUG")
	}
}

func TestDefaultSettings_EnvOverride_AsyncQueryTTL(t *testing.T) {
	t.Setenv("ADB_LINK_ASYNC_QUERY_TTL", "7200")
	s := DefaultSettings()
	if s.AsyncQueryTTL != 7200 {
		t.Errorf("AsyncQueryTTL = %d, want %d", s.AsyncQueryTTL, 7200)
	}
}

func TestEnvOrDefault_ReturnsEnvValue(t *testing.T) {
	t.Setenv("TEST_ENV_KEY", "custom_value")
	if got := envOrDefault("TEST_ENV_KEY", "default"); got != "custom_value" {
		t.Errorf("got %q, want %q", got, "custom_value")
	}
}

func TestEnvOrDefault_ReturnsDefault(t *testing.T) {
	os.Unsetenv("TEST_ENV_MISSING")
	if got := envOrDefault("TEST_ENV_MISSING", "fallback"); got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestEnvOrDefaultInt_Valid(t *testing.T) {
	t.Setenv("TEST_INT_KEY", "42")
	if got := envOrDefaultInt("TEST_INT_KEY", 10); got != 42 {
		t.Errorf("got %d, want %d", got, 42)
	}
}

func TestEnvOrDefaultInt_Empty(t *testing.T) {
	os.Unsetenv("TEST_INT_EMPTY")
	if got := envOrDefaultInt("TEST_INT_EMPTY", 99); got != 99 {
		t.Errorf("got %d, want %d", got, 99)
	}
}

func TestEnvOrDefaultInt_Invalid(t *testing.T) {
	t.Setenv("TEST_INT_BAD", "abc")
	if got := envOrDefaultInt("TEST_INT_BAD", 55); got != 55 {
		t.Errorf("got %d, want %d", got, 55)
	}
}

func TestEnvOrDefaultInt_NegativeFallback(t *testing.T) {
	t.Setenv("TEST_INT_NEG", "12.5")
	if got := envOrDefaultInt("TEST_INT_NEG", 77); got != 77 {
		t.Errorf("got %d, want %d (float should fallback)", got, 77)
	}
}
