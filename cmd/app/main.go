package main

import (
	"net/http"
	"os"

	"github.com/tediscript/gostarterkit/internal/config"
	"github.com/tediscript/gostarterkit/internal/handlers"
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

	// Initialize template cache
	templatesDir := "./templates"
	templateCache := templates.NewCache(cfg.App.Env == "development")
	if err := templateCache.LoadTemplates(templatesDir); err != nil {
		log.Error("Failed to load templates",
			"error", err.Error(),
		)
		os.Exit(1)
	}

	// Initialize handlers
	handlersInstance := handlers.New(templateCache, templatesDir)

	// Initialize routes with middlewares
	mux := http.NewServeMux()
	handler := routes.Routes(mux, handlersInstance)

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
