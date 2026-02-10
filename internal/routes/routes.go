package routes

import (
	"html/template"
	"net/http"
	"time"

	"github.com/tediscript/gostarterkit/internal/auth"
	"github.com/tediscript/gostarterkit/internal/config"
	"github.com/tediscript/gostarterkit/internal/handlers"
	"github.com/tediscript/gostarterkit/internal/middlewares"
)

// Routes registers all application routes and returns a handler with middleware
func Routes(mux *http.ServeMux, h *handlers.Handlers, cfg *config.Config, tpl *template.Template) http.Handler {
	// Register routes on the mux
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /livez", h.Livez)
	mux.HandleFunc("GET /readyz", h.Readyz)
	mux.HandleFunc("GET /", h.Home)

	// Auth routes
	mux.HandleFunc("GET /login", handlers.LoginPage(tpl))
	mux.HandleFunc("POST /login", handlers.LoginHandler)
	mux.HandleFunc("GET /logout", handlers.LogoutHandler)
	mux.Handle("GET /protected", auth.RequireAuth(handlers.ProtectedPage(tpl)))

	// API routes
	mux.HandleFunc("GET /api/status", h.APIStatus)
	mux.HandleFunc("GET /api/hello", h.APIHello)
	mux.HandleFunc("GET /api/error", h.APIError)
	mux.HandleFunc("GET /api/data", h.APIData)

	// API authentication routes (JWT)
	mux.HandleFunc("POST /api/login", handlers.APILoginHandler)
	mux.Handle("GET /api/protected", middlewares.JWTAuthMiddleware(http.HandlerFunc(handlers.APIProtectedHandler)))

	// Create rate limit middleware with configuration
	rateLimitMiddleware := middlewares.RateLimitMiddleware(
		cfg.RateLimit.RequestsPerWindow,
		time.Duration(cfg.RateLimit.WindowSeconds)*time.Second,
	)

	// Apply middlewares to all routes (order matters)
	return middlewares.CorrelationIDMiddleware(
		middlewares.RequestIDMiddleware(
			middlewares.LoggingMiddleware(
				rateLimitMiddleware(mux),
			),
		),
	)
}
