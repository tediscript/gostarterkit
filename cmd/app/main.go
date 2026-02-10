package main

import (
	"net/http"
	"os"

	"github.com/tediscript/gostarterkit/internal/auth"
	"github.com/tediscript/gostarterkit/internal/config"
	"github.com/tediscript/gostarterkit/internal/database"
	"github.com/tediscript/gostarterkit/internal/handlers"
	"github.com/tediscript/gostarterkit/internal/health"
	"github.com/tediscript/gostarterkit/internal/logger"
	"github.com/tediscript/gostarterkit/internal/routes"
	"github.com/tediscript/gostarterkit/internal/server"
	"github.com/tediscript/gostarterkit/internal/templates"
)

func main() {
	// Load configuration
	cfg := config.Load(".env")

	// Initialize logger
	log := logger.Init(cfg.App.LogLevel, cfg.App.LogFormat)

	log.Info("Go Starter Kit is starting",
		"version", "1.0.0",
		"environment", cfg.App.Env,
		"log_level", cfg.App.LogLevel,
		"log_format", cfg.App.LogFormat,
	)

	// Initialize database
	log.Info("Initializing database",
		"db_file", cfg.SQLite.DBFile,
		"max_open_connections", cfg.SQLite.MaxOpenConnections,
		"max_idle_connections", cfg.SQLite.MaxIdleConnections,
	)
	db, err := database.New(cfg)
	if err != nil {
		log.Error("Failed to initialize database",
			"error", err.Error(),
		)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	migrationsDir := "./migrations"
	log.Info("Running database migrations",
		"migrations_dir", migrationsDir,
	)
	if err := database.RunMigrations(db, migrationsDir); err != nil {
		log.Error("Failed to run migrations",
			"error", err.Error(),
		)
		os.Exit(1)
	}
	log.Info("Database migrations completed successfully")

	// Initialize template cache
	templatesDir := "./templates"
	templateCache := templates.NewCache(cfg.App.Env == "development")
	if err := templateCache.LoadTemplates(templatesDir); err != nil {
		log.Error("Failed to load templates",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// Initialize session store
	log.Info("Initializing session store",
		"cookie_name", cfg.Session.CookieName,
		"max_age_seconds", cfg.Session.MaxAgeSeconds,
		"cookie_http_only", cfg.Session.CookieHTTPOnly,
		"cookie_secure", cfg.Session.CookieSecure,
		"cookie_samesite", cfg.Session.CookieSameSite,
	)
	auth.Initialize(cfg)

	// Initialize health checker
	healthChecker := health.New(db)

	// Initialize handlers
	handlersInstance := handlers.New(templateCache, templatesDir, healthChecker)

	// Get the template for auth routes
	tpl, err := templateCache.GetTemplate("base.html")
	if err != nil {
		log.Error("Failed to get base template",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// Initialize routes with middlewares
	mux := http.NewServeMux()
	handler := routes.Routes(mux, handlersInstance, cfg, tpl)

	// Initialize server
	srv := server.New(cfg, handler, log.Logger)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Server error",
				"error", err.Error(),
			)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	srv.WaitForShutdown()
	log.Info("Application shutdown complete")
}
