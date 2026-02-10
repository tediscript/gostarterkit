package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tediscript/gostarterkit/internal/logger"
)

func TestCorrelationIDMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		inputHeader     string
		expectHeader    bool
		expectGenerated bool
	}{
		{
			name:            "no correlation ID header generates new ID",
			inputHeader:     "",
			expectHeader:    true,
			expectGenerated: true,
		},
		{
			name:            "existing correlation ID is preserved",
			inputHeader:     "existing-correlation-id",
			expectHeader:    true,
			expectGenerated: false,
		},
		{
			name:            "empty correlation ID header generates new ID",
			inputHeader:     "",
			expectHeader:    true,
			expectGenerated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that checks context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				correlationID := logger.GetCorrelationID(r.Context())

				if !tt.expectHeader && correlationID != "" {
					t.Errorf("Expected no correlation ID, got: %s", correlationID)
				}
				if tt.expectHeader && correlationID == "" {
					t.Error("Expected correlation ID to be set")
				}
				if tt.expectGenerated && correlationID == tt.inputHeader {
					t.Error("Expected generated correlation ID, got input header")
				}
				if !tt.expectGenerated && correlationID != tt.inputHeader {
					t.Errorf("Expected preserved correlation ID %s, got %s", tt.inputHeader, correlationID)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Apply middleware
			middleware := CorrelationIDMiddleware(handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.inputHeader != "" {
				req.Header.Set("X-Correlation-ID", tt.inputHeader)
			}

			// Execute request
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check response header
			responseCorrelationID := rr.Header().Get("X-Correlation-ID")
			if !tt.expectHeader && responseCorrelationID != "" {
				t.Errorf("Expected no correlation ID in response, got: %s", responseCorrelationID)
			}
			if tt.expectHeader && responseCorrelationID == "" {
				t.Error("Expected correlation ID in response header")
			}
			if tt.expectGenerated && responseCorrelationID == tt.inputHeader {
				t.Error("Expected generated correlation ID in response, got input header")
			}
			if !tt.expectGenerated && responseCorrelationID != tt.inputHeader {
				t.Errorf("Expected preserved correlation ID %s in response, got %s", tt.inputHeader, responseCorrelationID)
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer

	// Initialize logger
	cfg := logger.Config{
		Level:       slog.LevelInfo,
		Format:      "text",
		Output:      &logBuf,
		ErrorOutput: &logBuf,
	}
	log := logger.New(cfg)
	logger.SetDefault(log)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	middleware := LoggingMiddleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent/1.0")

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	// Check log output
	logOutput := logBuf.String()

	// Should contain request details
	if !strings.Contains(logOutput, "GET") {
		t.Error("Expected 'GET' in log output")
	}
	if !strings.Contains(logOutput, "/test") {
		t.Error("Expected '/test' in log output")
	}
	if !strings.Contains(logOutput, "foo=bar") {
		t.Error("Expected query string in log output")
	}
	if !strings.Contains(logOutput, "127.0.0.1:12345") {
		t.Error("Expected remote address in log output")
	}
	if !strings.Contains(logOutput, "test-agent/1.0") {
		t.Error("Expected user agent in log output")
	}
	if !strings.Contains(logOutput, "status=200") {
		t.Error("Expected status code in log output")
	}
}

func TestLoggingMiddlewareWithCorrelationID(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer

	// Initialize logger
	cfg := logger.Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &logBuf,
		ErrorOutput: &logBuf,
	}
	log := logger.New(cfg)
	logger.SetDefault(log)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply both middlewares
	middleware := CorrelationIDMiddleware(LoggingMiddleware(handler))

	// Create request with correlation ID
	correlationID := "test-correlation-id-123"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", correlationID)

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check log output contains correlation ID
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, correlationID) {
		t.Errorf("Expected correlation ID %s in log output, got: %s", correlationID, logOutput)
	}
}

func TestLoggingMiddlewareDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		expected int
	}{
		{
			name:     "200 OK",
			handler:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			expected: http.StatusOK,
		},
		{
			name:     "404 Not Found",
			handler:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			expected: http.StatusNotFound,
		},
		{
			name:     "500 Internal Server Error",
			handler:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var logBuf bytes.Buffer

			// Initialize logger
			cfg := logger.Config{
				Level:       slog.LevelInfo,
				Format:      "text",
				Output:      &logBuf,
				ErrorOutput: &logBuf,
			}
			log := logger.New(cfg)
			logger.SetDefault(log)

			// Apply middleware
			middleware := LoggingMiddleware(tt.handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			// Execute request
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check log output contains status code
			logOutput := logBuf.String()
			if !strings.Contains(logOutput, "status=") {
				t.Error("Expected status code in log output")
			}
		})
	}
}

func TestCorrelationIDMiddlewareVeryLongID(t *testing.T) {
	// Create a very long correlation ID
	longID := strings.Repeat("a", 10000)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := logger.GetCorrelationID(r.Context())
		if correlationID != longID {
			t.Errorf("Expected long correlation ID preserved, got length %d", len(correlationID))
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := CorrelationIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", longID)

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	responseID := rr.Header().Get("X-Correlation-ID")
	if responseID != longID {
		t.Errorf("Expected long correlation ID in response, got length %d", len(responseID))
	}
}

func TestCorrelationIDMiddlewareWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"UUID format", "550e8400-e29b-41d4-a716-446655440000"},
		{"ID with dashes", "test-id-with-dashes"},
		{"ID with underscores", "test_id_with_underscores"},
		{"ID with dots", "test.id.with.dots"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				correlationID := logger.GetCorrelationID(r.Context())
				if correlationID != tt.id {
					t.Errorf("Expected correlation ID %s, got %s", tt.id, correlationID)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := CorrelationIDMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Correlation-ID", tt.id)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			responseID := rr.Header().Get("X-Correlation-ID")
			if responseID != tt.id {
				t.Errorf("Expected correlation ID %s in response, got %s", tt.id, responseID)
			}
		})
	}
}

func TestLoggingMiddlewareJSONFormat(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer

	// Initialize logger with JSON format
	cfg := logger.Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &logBuf,
		ErrorOutput: &logBuf,
	}
	log := logger.New(cfg)
	logger.SetDefault(log)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	middleware := LoggingMiddleware(handler)

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check log output is valid JSON
	logOutput := logBuf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(logOutput), &logEntry); err != nil {
		t.Errorf("Expected valid JSON output, got error: %v\nOutput: %s", err, logOutput)
	}

	// Check required fields - slog uses "msg" for message, "message" for custom attrs
	if _, ok := logEntry["msg"]; !ok {
		t.Error("Expected 'msg' field in JSON output")
	}
	if _, ok := logEntry["level"]; !ok {
		t.Error("Expected 'level' field in JSON output")
	}
	if _, ok := logEntry["time"]; !ok {
		t.Error("Expected 'time' field in JSON output")
	}
	// Check that HTTP request fields are present
	if _, ok := logEntry["method"]; !ok {
		t.Error("Expected 'method' field in JSON output")
	}
	if _, ok := logEntry["path"]; !ok {
		t.Error("Expected 'path' field in JSON output")
	}
	if _, ok := logEntry["status"]; !ok {
		t.Error("Expected 'status' field in JSON output")
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		writeBody  bool
	}{
		{"200 with body", http.StatusOK, true},
		{"200 without body", http.StatusOK, false},
		{"404 without body", http.StatusNotFound, false},
		{"500 with body", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			rw := &responseWriter{ResponseWriter: rr}

			// Write status first (as in real HTTP)
			if tt.statusCode != 0 {
				rw.WriteHeader(tt.statusCode)
			}

			if tt.writeBody {
				rw.Write([]byte("test body"))
			}

			if tt.statusCode != 0 && rr.Code != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, rr.Code)
			}

			if tt.statusCode == 0 && rr.Code != http.StatusOK {
				t.Errorf("Expected default status code 200, got %d", rr.Code)
			}
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer

	// Initialize logger
	cfg := logger.Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &logBuf,
		ErrorOutput: &logBuf,
	}
	log := logger.New(cfg)
	logger.SetDefault(log)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correlation ID is in context
		correlationID := logger.GetCorrelationID(r.Context())
		if correlationID == "" {
			t.Error("Expected correlation ID in handler context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middlewares in chain
	middleware := CorrelationIDMiddleware(LoggingMiddleware(handler))

	// Create request
	correlationID := "test-chain-id"
	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-Correlation-ID", correlationID)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	// Check correlation ID in response
	responseID := rr.Header().Get("X-Correlation-ID")
	if responseID != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, responseID)
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, correlationID) {
		t.Errorf("Expected correlation ID %s in log output", correlationID)
	}
	if !strings.Contains(logOutput, "POST") {
		t.Error("Expected 'POST' in log output")
	}
	if !strings.Contains(logOutput, "/api/test") {
		t.Error("Expected '/api/test' in log output")
	}
}
