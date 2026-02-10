package middlewares

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tediscript/gostarterkit/internal/logger"
)

// CorrelationIDMiddleware adds or generates a correlation ID for each request
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing correlation ID in header
		correlationID := r.Header.Get("X-Correlation-ID")

		// Generate new correlation ID if not present
		if correlationID == "" {
			correlationID = logger.GenerateCorrelationID()
		}

		// Add correlation ID to context
		ctx := logger.NewContextWithCorrelationID(r.Context(), correlationID)

		// Add correlation ID to response header
		w.Header().Set("X-Correlation-ID", correlationID)

		// Serve request with new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDMiddleware adds or generates a request ID for each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing request ID in header
		requestID := r.Header.Get("X-Request-ID")

		// Generate new request ID if not present, empty, whitespace-only, or "null"
		trimmedID := strings.TrimSpace(requestID)
		if trimmedID == "" || strings.EqualFold(trimmedID, "null") {
			requestID = logger.GenerateRequestID()
		}

		// Add request ID to context
		ctx := logger.NewContextWithRequestID(r.Context(), requestID)

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Serve request with new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware logs request details with correlation ID
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get correlation ID from context
		correlationID := logger.GetCorrelationID(r.Context())

		// Log request details
		logger.InfoCtx(r.Context(), "HTTP request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status", wrapped.status),
			slog.Duration("duration", duration),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)

		// Also log if correlation ID was set
		if correlationID != "" {
			logger.DebugCtx(r.Context(), "Request processed with correlation ID",
				slog.String("correlation_id", correlationID),
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the status code on first write if not already set
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
