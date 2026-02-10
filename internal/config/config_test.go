package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment variables
	originalEnv := make(map[string]string)
	envVars := []string{
		"HTTP_PORT", "HTTP_SHUTDOWN_TIMEOUT", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT",
		"SQLITE_DB_FILE", "SQLITE_MAX_OPEN_CONNECTIONS", "SQLITE_MAX_IDLE_CONNECTIONS", "SQLITE_CONNECTION_MAX_LIFETIME_SECONDS",
		"JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE", "JWT_EXPIRATION_SECONDS",
		"SESSION_COOKIE_SECRET", "SESSION_COOKIE_NAME", "SESSION_MAX_AGE_SECONDS", "SESSION_COOKIE_HTTP_ONLY", "SESSION_COOKIE_SECURE", "SESSION_COOKIE_SAMESITE",
		"APP_ENV", "APP_LOG_LEVEL", "APP_LOG_FORMAT", "APP_NAME",
		"RATE_LIMIT_REQUESTS_PER_WINDOW", "RATE_LIMIT_WINDOW_SECONDS",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS", "CORS_ALLOWED_HEADERS", "CORS_MAX_AGE_SECONDS",
	}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("loads config with default values", func(t *testing.T) {
		// Clear all config-related environment variables
		for _, key := range envVars {
			os.Unsetenv(key)
		}

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.HTTP.Port != 8880 {
			t.Errorf("expected HTTP.Port to be 8880, got %d", cfg.HTTP.Port)
		}
		if cfg.SQLite.DBFile != "./app.db" {
			t.Errorf("expected SQLite.DBFile to be './app.db', got %s", cfg.SQLite.DBFile)
		}
		if cfg.App.Env != "development" {
			t.Errorf("expected App.Env to be 'development', got %s", cfg.App.Env)
		}
		if cfg.App.LogLevel != "info" {
			t.Errorf("expected App.LogLevel to be 'info', got %s", cfg.App.LogLevel)
		}
	})

	t.Run("loads config from environment variables", func(t *testing.T) {
		os.Setenv("HTTP_PORT", "9090")
		os.Setenv("SQLITE_DB_FILE", "/path/to/db.sqlite")
		os.Setenv("APP_ENV", "production")
		os.Setenv("APP_LOG_LEVEL", "debug")

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.HTTP.Port != 9090 {
			t.Errorf("expected HTTP.Port to be 9090, got %d", cfg.HTTP.Port)
		}
		if cfg.SQLite.DBFile != "/path/to/db.sqlite" {
			t.Errorf("expected SQLite.DBFile to be '/path/to/db.sqlite', got %s", cfg.SQLite.DBFile)
		}
		if cfg.App.Env != "production" {
			t.Errorf("expected App.Env to be 'production', got %s", cfg.App.Env)
		}
		if cfg.App.LogLevel != "debug" {
			t.Errorf("expected App.LogLevel to be 'debug', got %s", cfg.App.LogLevel)
		}
	})

	t.Run("loads duration from environment variables", func(t *testing.T) {
		os.Setenv("HTTP_SHUTDOWN_TIMEOUT", "45s")
		os.Setenv("HTTP_READ_TIMEOUT", "20s")

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.HTTP.ShutdownTimeout != 45*time.Second {
			t.Errorf("expected HTTP.ShutdownTimeout to be 45s, got %v", cfg.HTTP.ShutdownTimeout)
		}
		if cfg.HTTP.ReadTimeout != 20*time.Second {
			t.Errorf("expected HTTP.ReadTimeout to be 20s, got %v", cfg.HTTP.ReadTimeout)
		}
	})

	t.Run("loads boolean from environment variables", func(t *testing.T) {
		os.Setenv("SESSION_COOKIE_HTTP_ONLY", "false")
		os.Setenv("SESSION_COOKIE_SECURE", "false")

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.Session.CookieHTTPOnly {
			t.Errorf("expected Session.CookieHTTPOnly to be false, got %v", cfg.Session.CookieHTTPOnly)
		}
		if cfg.Session.CookieSecure {
			t.Errorf("expected Session.CookieSecure to be false, got %v", cfg.Session.CookieSecure)
		}
	})

	t.Run("uses text log format in development mode", func(t *testing.T) {
		os.Setenv("APP_ENV", "development")
		os.Unsetenv("APP_LOG_FORMAT")

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.App.LogFormat != "text" {
			t.Errorf("expected App.LogFormat to be 'text' in development, got %s", cfg.App.LogFormat)
		}
	})

	t.Run("uses json log format in production mode", func(t *testing.T) {
		os.Setenv("APP_ENV", "production")
		os.Unsetenv("APP_LOG_FORMAT")

		cfg := &Config{}
		loadConfig(cfg)

		if cfg.App.LogFormat != "json" {
			t.Errorf("expected App.LogFormat to be 'json' in production, got %s", cfg.App.LogFormat)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("validates valid config", func(t *testing.T) {
		cfg := &Config{}
		loadConfig(cfg)
		cfg.App.Env = "development"
		cfg.App.LogLevel = "info"
		cfg.App.LogFormat = "text"
		cfg.Session.CookieSameSite = "Lax"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected valid config, got error: %v", err)
		}
	})

	t.Run("rejects invalid HTTP port", func(t *testing.T) {
		cfg := &Config{}
		cfg.HTTP.Port = -1

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for negative port, got nil")
		}

		cfg.HTTP.Port = 70000
		err = cfg.Validate()
		if err == nil {
			t.Error("expected error for port > 65535, got nil")
		}
	})

	t.Run("requires JWT signing secret in production", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "production"
		cfg.JWT.SigningSecret = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing JWT signing secret in production, got nil")
		}
	})

	t.Run("requires session cookie secret in production", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "production"
		cfg.Session.CookieSecret = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing session cookie secret in production, got nil")
		}
	})

	t.Run("rejects invalid APP_ENV", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "invalid"

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid APP_ENV, got nil")
		}
	})

	t.Run("rejects invalid log level", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "development"
		cfg.App.LogLevel = "invalid"

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid log level, got nil")
		}
	})

	t.Run("rejects invalid log format", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "development"
		cfg.App.LogFormat = "invalid"

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid log format, got nil")
		}
	})

	t.Run("rejects invalid Cookie SameSite", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "development"
		cfg.Session.CookieSameSite = "invalid"

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for invalid Cookie SameSite, got nil")
		}
	})

	t.Run("rejects negative SQLite Max Idle Connections", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "development"
		cfg.SQLite.MaxIdleConnections = -1

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for negative SQLite Max Idle Connections, got nil")
		}
	})

	t.Run("rejects zero Rate Limit requests", func(t *testing.T) {
		cfg := &Config{}
		cfg.App.Env = "development"
		cfg.RateLimit.RequestsPerWindow = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for zero Rate Limit requests, got nil")
		}
	})
}

func TestLoadFromEnvFile(t *testing.T) {
	t.Run("loads simple env file", func(t *testing.T) {
		content := "HTTP_PORT=9090\nAPP_ENV=production\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		// Clear environment variables
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("APP_ENV")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("HTTP_PORT") != "9090" {
			t.Errorf("expected HTTP_PORT to be '9090', got '%s'", os.Getenv("HTTP_PORT"))
		}
		if os.Getenv("APP_ENV") != "production" {
			t.Errorf("expected APP_ENV to be 'production', got '%s'", os.Getenv("APP_ENV"))
		}
	})

	t.Run("handles comments", func(t *testing.T) {
		content := "# This is a comment\nHTTP_PORT=8880\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("HTTP_PORT")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("HTTP_PORT") != "8880" {
			t.Errorf("expected HTTP_PORT to be '8880', got '%s'", os.Getenv("HTTP_PORT"))
		}
	})

	t.Run("handles empty lines", func(t *testing.T) {
		content := "\n\nHTTP_PORT=8880\n\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("HTTP_PORT")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("HTTP_PORT") != "8880" {
			t.Errorf("expected HTTP_PORT to be '8880', got '%s'", os.Getenv("HTTP_PORT"))
		}
	})

	t.Run("handles quoted values", func(t *testing.T) {
		content := "APP_NAME=\"My App\"\nHTTP_PORT='9090'\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("APP_NAME")
		os.Unsetenv("HTTP_PORT")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("APP_NAME") != "My App" {
			t.Errorf("expected APP_NAME to be 'My App', got '%s'", os.Getenv("APP_NAME"))
		}
		if os.Getenv("HTTP_PORT") != "9090" {
			t.Errorf("expected HTTP_PORT to be '9090', got '%s'", os.Getenv("HTTP_PORT"))
		}
	})

	t.Run("handles whitespace", func(t *testing.T) {
		content := "HTTP_PORT = 8880\nAPP_ENV= production \n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("APP_ENV")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("HTTP_PORT") != "8880" {
			t.Errorf("expected HTTP_PORT to be '8880', got '%s'", os.Getenv("HTTP_PORT"))
		}
		if os.Getenv("APP_ENV") != "production" {
			t.Errorf("expected APP_ENV to be 'production', got '%s'", os.Getenv("APP_ENV"))
		}
	})

	t.Run("handles unicode and special characters", func(t *testing.T) {
		content := "APP_NAME=🚀 App\nAPP_DESC=Special chars: @#$%^&*()\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_DESC")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("APP_NAME") != "🚀 App" {
			t.Errorf("expected APP_NAME to be '🚀 App', got '%s'", os.Getenv("APP_NAME"))
		}
		if os.Getenv("APP_DESC") != "Special chars: @#$%^&*()" {
			t.Errorf("expected APP_DESC to be 'Special chars: @#$%%^&*()', got '%s'", os.Getenv("APP_DESC"))
		}
	})

	t.Run("handles empty .env file", func(t *testing.T) {
		content := ""
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("expected no error for empty .env file, got: %v", err)
		}
	})

	t.Run("handles missing .env file", func(t *testing.T) {
		err := LoadFromEnvFile("/nonexistent/file.env")
		if err != nil {
			t.Errorf("expected no error for missing .env file, got: %v", err)
		}
	})

	t.Run("rejects malformed .env file", func(t *testing.T) {
		content := "INVALID_LINE_WITHOUT_EQUALS\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		err = LoadFromEnvFile(tmpfile.Name())
		if err == nil {
			t.Error("expected error for malformed .env file, got nil")
		}
	})

	t.Run("does not override existing environment variables", func(t *testing.T) {
		content := "HTTP_PORT=8880\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Setenv("HTTP_PORT", "9090")

		err = LoadFromEnvFile(tmpfile.Name())
		if err != nil {
			t.Errorf("failed to load env file: %v", err)
		}

		if os.Getenv("HTTP_PORT") != "9090" {
			t.Errorf("expected HTTP_PORT to remain '9090', got '%s'", os.Getenv("HTTP_PORT"))
		}
	})
}

func TestGetEnvOrFile(t *testing.T) {
	t.Run("reads from file when _FILE variable is set", func(t *testing.T) {
		content := "my-secret-value"
		tmpfile, err := os.CreateTemp("", "testsecret*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		// Ensure direct env variable is not set
		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Setenv("JWT_SIGNING_SECRET_FILE", tmpfile.Name())

		result := getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")

		if result != "my-secret-value" {
			t.Errorf("expected 'my-secret-value', got '%s'", result)
		}

		os.Unsetenv("JWT_SIGNING_SECRET_FILE")
	})

	t.Run("prefers _FILE variable over direct variable", func(t *testing.T) {
		content := "file-secret"
		tmpfile, err := os.CreateTemp("", "testsecret*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Setenv("JWT_SIGNING_SECRET", "direct-secret")
		os.Setenv("JWT_SIGNING_SECRET_FILE", tmpfile.Name())

		result := getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")

		if result != "file-secret" {
			t.Errorf("expected 'file-secret' from file, got '%s'", result)
		}

		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Unsetenv("JWT_SIGNING_SECRET_FILE")
	})

	t.Run("uses direct variable when _FILE not set", func(t *testing.T) {
		os.Setenv("JWT_SIGNING_SECRET", "direct-value")
		os.Unsetenv("JWT_SIGNING_SECRET_FILE")

		result := getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")

		if result != "direct-value" {
			t.Errorf("expected 'direct-value', got '%s'", result)
		}

		os.Unsetenv("JWT_SIGNING_SECRET")
	})

	t.Run("trims whitespace from file content", func(t *testing.T) {
		content := "  secret-with-spaces  \n"
		tmpfile, err := os.CreateTemp("", "testsecret*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Setenv("JWT_SIGNING_SECRET_FILE", tmpfile.Name())

		result := getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")

		if result != "secret-with-spaces" {
			t.Errorf("expected trimmed 'secret-with-spaces', got '%s'", result)
		}

		os.Unsetenv("JWT_SIGNING_SECRET_FILE")
	})

	t.Run("handles very large file content", func(t *testing.T) {
		// Create a large string (1MB)
		largeContent := strings.Repeat("a", 1024*1024)
		tmpfile, err := os.CreateTemp("", "testsecret*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(largeContent)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Setenv("JWT_SIGNING_SECRET_FILE", tmpfile.Name())

		result := getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")

		if result != largeContent {
			t.Errorf("large content mismatch")
		}

		os.Unsetenv("JWT_SIGNING_SECRET_FILE")
	})

	t.Run("panics on file read error", func(t *testing.T) {
		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Setenv("JWT_SIGNING_SECRET_FILE", "/nonexistent/file.txt")

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when file doesn't exist")
			}
			os.Unsetenv("JWT_SIGNING_SECRET_FILE")
		}()

		_ = getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")
	})
}

func TestLoad(t *testing.T) {
	t.Run("panics on critical missing config in production", func(t *testing.T) {
		os.Setenv("APP_ENV", "production")
		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Unsetenv("JWT_SIGNING_SECRET_FILE")
		os.Unsetenv("SESSION_COOKIE_SECRET")

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on missing JWT signing secret in production")
			}
			os.Unsetenv("APP_ENV")
		}()

		Load("")
	})

	t.Run("loads successfully with valid config", func(t *testing.T) {
		os.Setenv("APP_ENV", "development")
		os.Unsetenv("JWT_SIGNING_SECRET")
		os.Unsetenv("JWT_SIGNING_SECRET_FILE")

		cfg := Load("")

		if cfg == nil {
			t.Error("expected config to be loaded, got nil")
		}
		if cfg.App.Env != "development" {
			t.Errorf("expected App.Env to be 'development', got %s", cfg.App.Env)
		}

		os.Unsetenv("APP_ENV")
	})

	t.Run("loads from .env file", func(t *testing.T) {
		content := "HTTP_PORT=3000\nAPP_ENV=production\nJWT_SIGNING_SECRET=test-secret\nSESSION_COOKIE_SECRET=cookie-secret\n"
		tmpfile, err := os.CreateTemp("", "testenv*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		// Clear environment variables to ensure .env file is used
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("APP_ENV")

		cfg := Load(tmpfile.Name())

		if cfg.HTTP.Port != 3000 {
			t.Errorf("expected HTTP.Port to be 3000, got %d", cfg.HTTP.Port)
		}
		if cfg.App.Env != "production" {
			t.Errorf("expected App.Env to be 'production', got %s", cfg.App.Env)
		}
	})
}

func TestConcurrentConfigAccess(t *testing.T) {
	t.Run("handles concurrent config loading", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cfg := Load("")
				if cfg == nil {
					t.Error("expected config to be loaded, got nil")
				}
			}()
		}

		wg.Wait()
	})
}

func TestGetEnvString(t *testing.T) {
	t.Run("returns environment variable when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test-value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvString("TEST_VAR", "default")
		if result != "test-value" {
			t.Errorf("expected 'test-value', got '%s'", result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		result := getEnvString("TEST_VAR", "default")
		if result != "default" {
			t.Errorf("expected 'default', got '%s'", result)
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		os.Setenv("TEST_VAR", "")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvString("TEST_VAR", "default")
		if result != "default" {
			t.Errorf("expected 'default' for empty env var, got '%s'", result)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("returns integer when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "42")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvInt("TEST_VAR", 0)
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		result := getEnvInt("TEST_VAR", 10)
		if result != 10 {
			t.Errorf("expected 10, got %d", result)
		}
	})

	t.Run("returns default on invalid integer", func(t *testing.T) {
		os.Setenv("TEST_VAR", "not-a-number")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvInt("TEST_VAR", 10)
		if result != 10 {
			t.Errorf("expected default 10 on invalid input, got %d", result)
		}
	})
}

func TestGetEnvBool(t *testing.T) {
	t.Run("returns true for 'true'", func(t *testing.T) {
		os.Setenv("TEST_VAR", "true")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvBool("TEST_VAR", false)
		if !result {
			t.Error("expected true, got false")
		}
	})

	t.Run("returns false for 'false'", func(t *testing.T) {
		os.Setenv("TEST_VAR", "false")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvBool("TEST_VAR", true)
		if result {
			t.Error("expected false, got true")
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		result := getEnvBool("TEST_VAR", true)
		if !result {
			t.Error("expected default true, got false")
		}
	})
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("returns duration when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "30s")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvDuration("TEST_VAR", 0)
		expected := 30 * time.Second
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		result := getEnvDuration("TEST_VAR", 5*time.Second)
		if result != 5*time.Second {
			t.Errorf("expected 5s, got %v", result)
		}
	})

	t.Run("returns default on invalid duration", func(t *testing.T) {
		os.Setenv("TEST_VAR", "not-a-duration")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvDuration("TEST_VAR", 5*time.Second)
		if result != 5*time.Second {
			t.Errorf("expected default 5s on invalid input, got %v", result)
		}
	})
}

func TestEnvFilePermissions(t *testing.T) {
	t.Run("panics on unreadable file", func(t *testing.T) {
		// Create a temporary directory to control permissions
		tmpdir, err := os.MkdirTemp("", "testperms*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		// Create a file and remove read permissions
		secretFile := filepath.Join(tmpdir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("secret"), 0600); err != nil {
			t.Fatal(err)
		}

		// On Unix, we can remove read permissions
		// Note: This may not work on all systems, so we test the behavior
		if err := os.Chmod(secretFile, 0000); err == nil {
			os.Unsetenv("JWT_SIGNING_SECRET")
			os.Setenv("JWT_SIGNING_SECRET_FILE", secretFile)

			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic when file is unreadable")
				}
				os.Unsetenv("JWT_SIGNING_SECRET_FILE")
				os.Chmod(secretFile, 0600) // Restore for cleanup
			}()

			_ = getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")
		}
		// If chmod failed (e.g., on Windows), skip this test
	})
}
