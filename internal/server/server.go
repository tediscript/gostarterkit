package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tediscript/gostarterkit/internal/config"
)

// Server wraps the HTTP server with graceful shutdown capabilities
type Server struct {
	httpServer *http.Server
	config     *config.Config
	logger     *slog.Logger
}

// New creates a new Server instance with the given configuration and handler
func New(cfg *config.Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
			Handler:      handler,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
			IdleTimeout:  cfg.HTTP.IdleTimeout,
		},
		config: cfg,
		logger: logger,
	}
}

// Start begins listening for HTTP connections and blocks until the server is stopped
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		slog.Int("port", s.config.HTTP.Port),
		slog.Duration("read_timeout", s.config.HTTP.ReadTimeout),
		slog.Duration("write_timeout", s.config.HTTP.WriteTimeout),
		slog.Duration("idle_timeout", s.config.HTTP.IdleTimeout),
	)

	// Listen on the configured port
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.HTTP.Port, err)
	}

	s.logger.Info("Server started successfully",
		slog.String("addr", listener.Addr().String()),
	)

	// Start serving HTTP requests
	if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the server with a timeout
func (s *Server) Stop() error {
	s.logger.Info("Shutting down server gracefully",
		slog.Duration("timeout", s.config.HTTP.ShutdownTimeout),
	)

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		// If context deadline exceeded, force close
		if err == context.DeadlineExceeded {
			s.logger.Warn("Shutdown timeout exceeded, forcing server close",
				slog.Duration("timeout", s.config.HTTP.ShutdownTimeout),
			)
			if closeErr := s.httpServer.Close(); closeErr != nil {
				return fmt.Errorf("error forcing server close: %w", closeErr)
			}
			return fmt.Errorf("shutdown timed out after %v", s.config.HTTP.ShutdownTimeout)
		}
		return fmt.Errorf("error during shutdown: %w", err)
	}

	s.logger.Info("Server stopped successfully")
	return nil
}

// WaitForShutdown blocks until SIGTERM or SIGINT is received, then shuts down gracefully
func (s *Server) WaitForShutdown() {
	// Create channel to listen for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Wait for signal
	sig := <-sigChan
	s.logger.Info("Received shutdown signal",
		slog.String("signal", sig.String()),
	)

	// Perform graceful shutdown
	if err := s.Stop(); err != nil {
		s.logger.Error("Error during server shutdown",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
