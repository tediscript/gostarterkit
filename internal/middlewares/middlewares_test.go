package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		inputHeader     string
		expectHeader    bool
		expectGenerated bool
	}{
		{
			name:            "no request ID header generates new ID",
			inputHeader:     "",
			expectHeader:    true,
			expectGenerated: true,
		},
		{
			name:            "existing request ID is preserved",
			inputHeader:     "existing-request-id",
			expectHeader:    true,
			expectGenerated: false,
		},
		{
			name:            "empty request ID header generates new ID",
			inputHeader:     "",
			expectHeader:    true,
			expectGenerated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that checks context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID := logger.GetRequestID(r.Context())

				if !tt.expectHeader && requestID != "" {
					t.Errorf("Expected no request ID, got: %s", requestID)
				}
				if tt.expectHeader && requestID == "" {
					t.Error("Expected request ID to be set")
				}
				if tt.expectGenerated && requestID == tt.inputHeader {
					t.Error("Expected generated request ID, got input header")
				}
				if !tt.expectGenerated && requestID != tt.inputHeader {
					t.Errorf("Expected preserved request ID %s, got %s", tt.inputHeader, requestID)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Apply middleware
			middleware := RequestIDMiddleware(handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.inputHeader != "" {
				req.Header.Set("X-Request-ID", tt.inputHeader)
			}

			// Execute request
			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check response header
			responseRequestID := rr.Header().Get("X-Request-ID")
			if !tt.expectHeader && responseRequestID != "" {
				t.Errorf("Expected no request ID in response, got: %s", responseRequestID)
			}
			if tt.expectHeader && responseRequestID == "" {
				t.Error("Expected request ID in response header")
			}
			if tt.expectGenerated && responseRequestID == tt.inputHeader {
				t.Error("Expected generated request ID in response, got input header")
			}
			if !tt.expectGenerated && responseRequestID != tt.inputHeader {
				t.Errorf("Expected preserved request ID %s in response, got %s", tt.inputHeader, responseRequestID)
			}
		})
	}
}

func TestRequestIDMiddlewareWithLogging(t *testing.T) {
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
		// Log something in handler
		logger.InfoCtx(r.Context(), "Handler processing request")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply both middlewares
	middleware := RequestIDMiddleware(LoggingMiddleware(handler))

	// Create request with request ID
	requestID := "test-request-id-123"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", requestID)

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check log output contains request ID
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, requestID) {
		t.Errorf("Expected request ID %s in log output, got: %s", requestID, logOutput)
	}

	// Verify request ID is in response header
	responseID := rr.Header().Get("X-Request-ID")
	if responseID != requestID {
		t.Errorf("Expected request ID %s in response header, got %s", requestID, responseID)
	}
}

func TestRequestIDMiddlewareMalformedUUID(t *testing.T) {
	// Malformed UUID should still be preserved (middleware doesn't validate format)
	malformedUUID := "not-a-valid-uuid"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := logger.GetRequestID(r.Context())
		if requestID != malformedUUID {
			t.Errorf("Expected malformed UUID to be preserved, got: %s", requestID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", malformedUUID)

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	responseID := rr.Header().Get("X-Request-ID")
	if responseID != malformedUUID {
		t.Errorf("Expected malformed UUID in response, got: %s", responseID)
	}
}

func TestRequestIDMiddlewareEmptyOrNull(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"null-like", "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID := logger.GetRequestID(r.Context())
				// Should generate a new UUID when empty/null
				if requestID == "" {
					t.Error("Expected generated request ID, got empty")
				}
				if tt.value != "" && requestID == tt.value {
					t.Errorf("Expected generated request ID, got input: %s", requestID)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequestIDMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.value != "" {
				req.Header.Set("X-Request-ID", tt.value)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			responseID := rr.Header().Get("X-Request-ID")
			if responseID == "" {
				t.Error("Expected request ID in response, got empty")
			}
			if tt.value != "" && responseID == tt.value {
				t.Errorf("Expected generated request ID in response, got input: %s", responseID)
			}
		})
	}
}

func TestRequestIDMiddlewareVeryLongID(t *testing.T) {
	// Create a very long request ID
	longID := strings.Repeat("a", 10000)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := logger.GetRequestID(r.Context())
		if requestID != longID {
			t.Errorf("Expected long request ID preserved, got length %d", len(requestID))
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestIDMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", longID)

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	responseID := rr.Header().Get("X-Request-ID")
	if responseID != longID {
		t.Errorf("Expected long request ID in response, got length %d", len(responseID))
	}
}

func TestRequestIDMiddlewarePreserved(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"UUID format", "550e8400-e29b-41d4-a716-446655440000"},
		{"ID with dashes", "test-id-with-dashes"},
		{"ID with underscores", "test_id_with_underscores"},
		{"ID with dots", "test.id.with.dots"},
		{"ID with numbers", "req-123-456-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID := logger.GetRequestID(r.Context())
				if requestID != tt.id {
					t.Errorf("Expected request ID %s, got %s", tt.id, requestID)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequestIDMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Request-ID", tt.id)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			responseID := rr.Header().Get("X-Request-ID")
			if responseID != tt.id {
				t.Errorf("Expected request ID %s in response, got %s", tt.id, responseID)
			}
		})
	}
}

func TestRequestIDMiddlewareConcurrentRequests(t *testing.T) {
	// Test multiple concurrent requests with different IDs
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := logger.GetRequestID(r.Context())
		w.Write([]byte(requestID))
	})

	middleware := RequestIDMiddleware(handler)

	done := make(chan bool, 3)

	// Launch 3 concurrent requests
	for i := 0; i < 3; i++ {
		go func(index int) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Request-ID", fmt.Sprintf("req-%d", index))

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			responseBody := rr.Body.String()
			expectedID := fmt.Sprintf("req-%d", index)
			if responseBody != expectedID {
				t.Errorf("Expected request ID %s, got %s", expectedID, responseBody)
			}

			responseID := rr.Header().Get("X-Request-ID")
			if responseID != expectedID {
				t.Errorf("Expected request ID %s in header, got %s", expectedID, responseID)
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestRequestIDMiddlewareSpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"ID with unicode", "test-id-αβγ"},
		{"ID with emoji", "test-id-🎉"},
		{"ID with slashes", "test/id/with/slashes"},
		{"ID with spaces", "test id with spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID := logger.GetRequestID(r.Context())
				if requestID != tt.id {
					t.Errorf("Expected request ID %s, got %s", tt.id, requestID)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequestIDMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Request-ID", tt.id)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			responseID := rr.Header().Get("X-Request-ID")
			if responseID != tt.id {
				t.Errorf("Expected request ID %s in response, got %s", tt.id, responseID)
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

		// Check request ID is in context
		requestID := logger.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Expected request ID in handler context")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply all middlewares in chain
	middleware := CorrelationIDMiddleware(RequestIDMiddleware(LoggingMiddleware(handler)))

	// Create request
	correlationID := "test-correlation-id"
	requestID := "test-request-id"
	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-Correlation-ID", correlationID)
	req.Header.Set("X-Request-ID", requestID)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", rr.Code)
	}

	// Check correlation ID in response
	responseCorrID := rr.Header().Get("X-Correlation-ID")
	if responseCorrID != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, responseCorrID)
	}

	// Check request ID in response
	responseReqID := rr.Header().Get("X-Request-ID")
	if responseReqID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, responseReqID)
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, correlationID) {
		t.Errorf("Expected correlation ID %s in log output", correlationID)
	}
	if !strings.Contains(logOutput, requestID) {
		t.Errorf("Expected request ID %s in log output", requestID)
	}
	if !strings.Contains(logOutput, "POST") {
		t.Error("Expected 'POST' in log output")
	}
	if !strings.Contains(logOutput, "/api/test") {
		t.Error("Expected '/api/test' in log output")
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	// Create test handler that checks user ID
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r)
		if !ok {
			t.Error("Expected user ID in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(userID))
	})

	// Apply JWT middleware
	middleware := JWTAuthMiddleware(handler)

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got: %d", rr.Code)
		}

		// Response should indicate authorization is required
		body := rr.Body.String()
		if body == "" {
			t.Error("Expected response body to contain error message")
		}
	})

	t.Run("empty authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "")

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got: %d", rr.Code)
		}
	})

	t.Run("bearer prefix format", func(t *testing.T) {
		token := "some.jwt.token"

		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		// Will fail validation but demonstrates Bearer prefix handling
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got: %d", rr.Code)
		}
	})

	t.Run("token without bearer prefix", func(t *testing.T) {
		token := "some.jwt.token"

		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", token)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		// Will fail validation but demonstrates raw token handling
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got: %d", rr.Code)
		}
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("user ID in context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r)
			if !ok {
				t.Error("Expected user ID to be present")
			}
			if userID != "testuser123" {
				t.Errorf("Expected user ID 'testuser123', got: %s", userID)
			}
			w.WriteHeader(http.StatusOK)
		})

		// Manually set user ID in context
		ctx := context.WithValue(context.Background(), UserIDContextKey, "testuser123")
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", rr.Code)
		}
	})

	t.Run("user ID not in context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r)
			if ok {
				t.Error("Expected user ID to not be present")
			}
			if userID != "" {
				t.Errorf("Expected empty user ID, got: %s", userID)
			}
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", rr.Code)
		}
	})
}
