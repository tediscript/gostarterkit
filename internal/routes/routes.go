package routes

import (
	"encoding/json"
	"net/http"

	"github.com/tediscript/gostarterkit/internal/handlers"
	"github.com/tediscript/gostarterkit/internal/middlewares"
)

// Routes registers all application routes and returns a handler with middleware
func Routes(mux *http.ServeMux, h *handlers.Handlers) http.Handler {
	// Register routes on the mux
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /livez", handleLivez)
	mux.HandleFunc("GET /readyz", handleReadyz)
	mux.HandleFunc("GET /", h.Home)

	// Apply middlewares to all routes
	return middlewares.CorrelationIDMiddleware(
		middlewares.LoggingMiddleware(mux),
	)
}

// Health check handlers
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleLivez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
