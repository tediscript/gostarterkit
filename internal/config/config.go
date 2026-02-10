package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration with contextually named fields
type Config struct {
	// HTTP Server Configuration
	HTTP struct {
		Port            int           `env:"HTTP_PORT" default:"8080"`
		ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" default:"30s"`
		ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" default:"15s"`
		WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" default:"15s"`
		IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" default:"60s"`
	}

	// Database Configuration
	SQLite struct {
		DBFile                       string `env:"SQLITE_DB_FILE" default:"./app.db"`
		MaxOpenConnections           int    `env:"SQLITE_MAX_OPEN_CONNECTIONS" default:"25"`
		MaxIdleConnections           int    `env:"SQLITE_MAX_IDLE_CONNECTIONS" default:"25"`
		ConnectionMaxLifetimeSeconds int    `env:"SQLITE_CONNECTION_MAX_LIFETIME_SECONDS" default:"300"`
	}

	// JWT Authentication Configuration
	JWT struct {
		SigningSecret     string `env:"JWT_SIGNING_SECRET"`
		SigningSecretFile string `env:"JWT_SIGNING_SECRET_FILE"`
		ExpirationSeconds int    `env:"JWT_EXPIRATION_SECONDS" default:"3600"`
	}

	// Session Authentication Configuration
	Session struct {
		CookieSecret   string `env:"SESSION_COOKIE_SECRET"`
		CookieName     string `env:"SESSION_COOKIE_NAME" default:"session"`
		MaxAgeSeconds  int    `env:"SESSION_MAX_AGE_SECONDS" default:"3600"`
		CookieHTTPOnly bool   `env:"SESSION_COOKIE_HTTP_ONLY" default:"true"`
		CookieSecure   bool   `env:"SESSION_COOKIE_SECURE" default:"true"`
		CookieSameSite string `env:"SESSION_COOKIE_SAMESITE" default:"Lax"`
	}

	// Application Configuration
	App struct {
		Env       string `env:"APP_ENV" default:"development"`
		LogLevel  string `env:"APP_LOG_LEVEL" default:"info"`
		LogFormat string `env:"APP_LOG_FORMAT" default:"json"`
		Name      string `env:"APP_NAME" default:"Go Starter Kit"`
	}

	// Rate Limiting Configuration
	RateLimit struct {
		RequestsPerWindow int `env:"RATE_LIMIT_REQUESTS_PER_WINDOW" default:"100"`
		WindowSeconds     int `env:"RATE_LIMIT_WINDOW_SECONDS" default:"60"`
	}

	// CORS Configuration
	CORS struct {
		AllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" default:"*"`
		AllowedMethods string `env:"CORS_ALLOWED_METHODS" default:"GET,POST,PUT,DELETE,OPTIONS"`
		AllowedHeaders string `env:"CORS_ALLOWED_HEADERS" default:"Content-Type,Authorization"`
		MaxAgeSeconds  int    `env:"CORS_MAX_AGE_SECONDS" default:"86400"`
	}
}

// Load creates a new Config instance by loading from environment variables
// and optionally from a .env file
func Load(envFile string) *Config {
	cfg := &Config{}

	// Load from .env file if specified
	if envFile != "" {
		if err := LoadFromEnvFile(envFile); err != nil {
			panic(fmt.Sprintf("failed to load .env file: %v", err))
		}
	}

	// Load configuration from environment variables
	loadConfig(cfg)

	// Validate critical configuration
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("config validation failed: %v", err))
	}

	return cfg
}

// loadConfig populates the Config struct from environment variables
func loadConfig(cfg *Config) {
	// HTTP Configuration
	cfg.HTTP.Port = getEnvInt("HTTP_PORT", 8080)
	cfg.HTTP.ShutdownTimeout = getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second)
	cfg.HTTP.ReadTimeout = getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second)
	cfg.HTTP.WriteTimeout = getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second)
	cfg.HTTP.IdleTimeout = getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)

	// SQLite Configuration
	cfg.SQLite.DBFile = getEnvString("SQLITE_DB_FILE", "./app.db")
	cfg.SQLite.MaxOpenConnections = getEnvInt("SQLITE_MAX_OPEN_CONNECTIONS", 25)
	cfg.SQLite.MaxIdleConnections = getEnvInt("SQLITE_MAX_IDLE_CONNECTIONS", 25)
	cfg.SQLite.ConnectionMaxLifetimeSeconds = getEnvInt("SQLITE_CONNECTION_MAX_LIFETIME_SECONDS", 300)

	// JWT Configuration
	cfg.JWT.SigningSecret = getEnvOrFile("JWT_SIGNING_SECRET", "JWT_SIGNING_SECRET_FILE")
	cfg.JWT.ExpirationSeconds = getEnvInt("JWT_EXPIRATION_SECONDS", 3600)

	// Session Configuration
	cfg.Session.CookieSecret = getEnvString("SESSION_COOKIE_SECRET", "")
	cfg.Session.CookieName = getEnvString("SESSION_COOKIE_NAME", "session")
	cfg.Session.MaxAgeSeconds = getEnvInt("SESSION_MAX_AGE_SECONDS", 3600)
	cfg.Session.CookieHTTPOnly = getEnvBool("SESSION_COOKIE_HTTP_ONLY", true)
	cfg.Session.CookieSecure = getEnvBool("SESSION_COOKIE_SECURE", true)
	cfg.Session.CookieSameSite = getEnvString("SESSION_COOKIE_SAMESITE", "Lax")

	// Application Configuration
	cfg.App.Env = getEnvString("APP_ENV", "development")
	cfg.App.LogLevel = getEnvString("APP_LOG_LEVEL", "info")
	cfg.App.LogFormat = getEnvString("APP_LOG_FORMAT", cfg.getAppDefaultLogFormat())
	cfg.App.Name = getEnvString("APP_NAME", "Go Starter Kit")

	// Rate Limiting Configuration
	cfg.RateLimit.RequestsPerWindow = getEnvInt("RATE_LIMIT_REQUESTS_PER_WINDOW", 100)
	cfg.RateLimit.WindowSeconds = getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)

	// CORS Configuration
	cfg.CORS.AllowedOrigins = getEnvString("CORS_ALLOWED_ORIGINS", "*")
	cfg.CORS.AllowedMethods = getEnvString("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	cfg.CORS.AllowedHeaders = getEnvString("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")
	cfg.CORS.MaxAgeSeconds = getEnvInt("CORS_MAX_AGE_SECONDS", 86400)
}

// getAppDefaultLogFormat returns the default log format based on APP_ENV
func (c *Config) getAppDefaultLogFormat() string {
	if c.App.Env == "development" {
		return "text"
	}
	return "json"
}

// Validate checks the configuration for critical errors
func (c *Config) Validate() error {
	// Validate HTTP Port
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535, got: %d", c.HTTP.Port)
	}

	// Validate JWT signing secret (required in production)
	if c.App.Env == "production" && c.JWT.SigningSecret == "" {
		return fmt.Errorf("JWT_SIGNING_SECRET or JWT_SIGNING_SECRET_FILE is required in production")
	}

	// Validate Session cookie secret (required in production)
	if c.App.Env == "production" && c.Session.CookieSecret == "" {
		return fmt.Errorf("SESSION_COOKIE_SECRET is required in production")
	}

	// Validate App Environment
	if c.App.Env != "development" && c.App.Env != "production" && c.App.Env != "test" {
		return fmt.Errorf("APP_ENV must be 'development', 'production', or 'test', got: %s", c.App.Env)
	}

	// Validate Log Level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.App.LogLevel] {
		return fmt.Errorf("APP_LOG_LEVEL must be 'debug', 'info', 'warn', or 'error', got: %s", c.App.LogLevel)
	}

	// Validate Log Format
	if c.App.LogFormat != "json" && c.App.LogFormat != "text" {
		return fmt.Errorf("APP_LOG_FORMAT must be 'json' or 'text', got: %s", c.App.LogFormat)
	}

	// Validate Cookie SameSite
	validSameSiteValues := map[string]bool{"Strict": true, "Lax": true, "None": true}
	if !validSameSiteValues[c.Session.CookieSameSite] {
		return fmt.Errorf("SESSION_COOKIE_SAMESITE must be 'Strict', 'Lax', or 'None', got: %s", c.Session.CookieSameSite)
	}

	// Validate SQLite Max Open Connections
	if c.SQLite.MaxOpenConnections <= 0 {
		return fmt.Errorf("SQLITE_MAX_OPEN_CONNECTIONS must be positive, got: %d", c.SQLite.MaxOpenConnections)
	}

	// Validate SQLite Max Idle Connections
	if c.SQLite.MaxIdleConnections < 0 {
		return fmt.Errorf("SQLITE_MAX_IDLE_CONNECTIONS must be non-negative, got: %d", c.SQLite.MaxIdleConnections)
	}

	// Validate Rate Limiting
	if c.RateLimit.RequestsPerWindow <= 0 {
		return fmt.Errorf("RATE_LIMIT_REQUESTS_PER_WINDOW must be positive, got: %d", c.RateLimit.RequestsPerWindow)
	}
	if c.RateLimit.WindowSeconds <= 0 {
		return fmt.Errorf("RATE_LIMIT_WINDOW_SECONDS must be positive, got: %d", c.RateLimit.WindowSeconds)
	}

	return nil
}

// LoadFromEnvFile reads a .env file and sets environment variables
func LoadFromEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// .env file is optional, don't error if it doesn't exist
			return nil
		}
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid syntax at line %d: %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present (handle both single and double quotes)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %v", err)
	}

	return nil
}

// Helper functions for getting environment variables with defaults

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if durationVal, err := time.ParseDuration(value); err == nil {
			return durationVal
		}
	}
	return defaultValue
}

// getEnvOrFile returns the value from environment variable or reads from file
// This implements Docker Swarm-style _FILE suffix support
func getEnvOrFile(key, fileKey string) string {
	// First check if the file-based variable is set
	if filePath := os.Getenv(fileKey); filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			panic(fmt.Sprintf("failed to read file for %s: %v", fileKey, err))
		}
		return strings.TrimSpace(string(content))
	}

	// Otherwise, return the direct environment variable
	return os.Getenv(key)
}
