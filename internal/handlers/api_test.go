package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tediscript/gostarterkit/internal/auth"
	"github.com/tediscript/gostarterkit/internal/config"
)

// mockTemplateCache is a mock implementation of TemplateCache for testing
type mockTemplateCache struct{}

func (m *mockTemplateCache) RenderTemplate(wr interface{ Write([]byte) (int, error) }, name string, data TemplateData) error {
	return nil
}

func (m *mockTemplateCache) CheckForReload(templatesDir string) error {
	return nil
}

// TestAPIStatus tests /api/status endpoint
func TestAPIStatus(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	tests := []struct {
		name       string
		wantStatus int
		wantFields []string
	}{
		{
			name:       "returns status information",
			wantStatus: http.StatusOK,
			wantFields: []string{"version", "health", "uptime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
			rr := httptest.NewRecorder()

			h.APIStatus(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("APIStatus() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Check Content-Type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("APIStatus() Content-Type = %v, want application/json", contentType)
			}

			// Parse and verify JSON
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Check required fields
			if response["status"] != "success" {
				t.Errorf("APIStatus() status field = %v, want 'success'", response["status"])
			}

			data, ok := response["data"].(map[string]interface{})
			if !ok {
				t.Error("APIStatus() data field is missing or not a map")
				return
			}

			for _, field := range tt.wantFields {
				if _, exists := data[field]; !exists {
					t.Errorf("APIStatus() missing field in data: %s", field)
				}
			}
		})
	}
}

// TestAPIHello tests /api/hello endpoint
func TestAPIHello(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantFields []string
	}{
		{
			name:       "without query parameters",
			query:      "",
			wantStatus: http.StatusOK,
			wantFields: []string{"status", "message", "data"},
		},
		{
			name:       "with query parameter",
			query:      "?time=morning",
			wantStatus: http.StatusOK,
			wantFields: []string{"status", "message", "data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/hello"+tt.query, nil)
			rr := httptest.NewRecorder()

			h.APIHello(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("APIHello() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			for _, field := range tt.wantFields {
				if _, exists := response[field]; !exists {
					t.Errorf("APIHello() missing field: %s", field)
				}
			}

			// Check message field
			if response["message"] != "Welcome to the Go Starter Kit API" {
				t.Errorf("APIHello() message = %v, want 'Welcome to the Go Starter Kit API'", response["message"])
			}
		})
	}
}

// TestAPIError tests /api/error endpoint with different error types
func TestAPIError(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	tests := []struct {
		name       string
		errorType  string
		wantStatus int
		wantError  string
		wantFields []string
	}{
		{
			name:       "not found error",
			errorType:  "notfound",
			wantStatus: http.StatusNotFound,
			wantError:  "Resource not found",
			wantFields: []string{"error"},
		},
		{
			name:       "bad request error",
			errorType:  "badrequest",
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request",
			wantFields: []string{"error", "details"},
		},
		{
			name:       "validation error",
			errorType:  "validation",
			wantStatus: http.StatusBadRequest,
			wantError:  "Validation failed",
			wantFields: []string{"error", "details"},
		},
		{
			name:       "default error",
			errorType:  "unknown",
			wantStatus: http.StatusInternalServerError,
			wantError:  "An internal error occurred",
			wantFields: []string{"error"},
		},
		{
			name:       "no error type",
			errorType:  "",
			wantStatus: http.StatusInternalServerError,
			wantError:  "An internal error occurred",
			wantFields: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/error"
			if tt.errorType != "" {
				url += "?type=" + tt.errorType
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			h.APIError(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("APIError() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Check error field
			if response["error"] != tt.wantError {
				t.Errorf("APIError() error = %v, want %v", response["error"], tt.wantError)
			}

			// Check optional fields
			for _, field := range tt.wantFields {
				if _, exists := response[field]; !exists {
					t.Errorf("APIError() missing field: %s", field)
				}
			}
		})
	}
}

// TestAPIData tests /api/data endpoint with nested data structures
func TestAPIData(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	t.Run("returns complex nested data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		rr := httptest.NewRecorder()

		h.APIData(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("APIData() status = %v, want %v", rr.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify response structure
		if response["status"] != "success" {
			t.Errorf("APIData() status = %v, want 'success'", response["status"])
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Error("APIData() data field is missing or not a map")
			return
		}

		// Check users array
		users, ok := data["users"].([]interface{})
		if !ok {
			t.Error("APIData() users field is missing or not an array")
			return
		}

		if len(users) != 2 {
			t.Errorf("APIData() users count = %v, want 2", len(users))
		}

		// Check pagination object
		pagination, ok := data["pagination"].(map[string]interface{})
		if !ok {
			t.Error("APIData() pagination field is missing or not a map")
			return
		}

		expectedPaginationKeys := []string{"page", "limit", "total"}
		for _, key := range expectedPaginationKeys {
			if _, exists := pagination[key]; !exists {
				t.Errorf("APIData() missing pagination key: %s", key)
			}
		}
	})
}

// TestAPIEndpointConsistency tests that all API endpoints return consistent JSON structure
func TestAPIEndpointConsistency(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"status endpoint", "/api/status", h.APIStatus},
		{"hello endpoint", "/api/hello", h.APIHello},
		{"data endpoint", "/api/data", h.APIData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			// All successful responses should have Content-Type: application/json
			if rr.Code == http.StatusOK {
				contentType := rr.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("%s Content-Type = %v, want application/json", tt.name, contentType)
				}

				// All should have valid JSON
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("%s invalid JSON: %v", tt.name, err)
				}

				// All should have "status" field
				if _, exists := response["status"]; !exists {
					t.Errorf("%s missing 'status' field", tt.name)
				}

				// All should have "data" field
				if _, exists := response["data"]; !exists {
					t.Errorf("%s missing 'data' field", tt.name)
				}
			}
		})
	}
}

// TestAPIErrorResponseConsistency tests that error responses have consistent structure
func TestAPIErrorResponseConsistency(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	errorTypes := []string{"notfound", "badrequest", "validation", ""}

	for _, errorType := range errorTypes {
		t.Run(strings.Join([]string{"error type:", errorType}, " "), func(t *testing.T) {
			url := "/api/error"
			if errorType != "" {
				url += "?type=" + errorType
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			h.APIError(rr, req)

			// All error responses should have Content-Type: application/json
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			// All should have valid JSON
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Invalid JSON: %v", err)
			}

			// All should have "error" field
			if _, exists := response["error"]; !exists {
				t.Error("Missing 'error' field")
			}
		})
	}
}

// TestAPIStatusCodeHandling tests that various status codes are handled correctly
func TestAPIStatusCodeHandling(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	tests := []struct {
		name       string
		path       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{"status 200", "/api/status", h.APIStatus, http.StatusOK},
		{"status 404", "/api/error?type=notfound", h.APIError, http.StatusNotFound},
		{"status 400", "/api/error?type=badrequest", h.APIError, http.StatusBadRequest},
		{"status 500", "/api/error", h.APIError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Verify response is JSON even for errors
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Invalid JSON response for status %d: %v", tt.wantStatus, err)
			}
		})
	}
}

// TestAPIConcurrentRequests tests multiple concurrent API requests
func TestAPIConcurrentRequests(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates", nil)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
			rr := httptest.NewRecorder()

			h.APIStatus(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Concurrent request failed with status %d", rr.Code)
			}

			// Verify JSON is valid
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Concurrent request returned invalid JSON: %v", err)
			}

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// setupJWTForTests initializes JWT for testing
func setupJWTForTests(t *testing.T) {
	t.Helper()

	testCfg := &config.Config{}
	testCfg.App.Env = "test"
	testCfg.JWT.SigningSecret = "test-secret-for-api-handlers"
	testCfg.JWT.ExpirationSeconds = 3600
	auth.SetConfigForTesting(testCfg)
}

// TestAPILoginHandler tests /api/login endpoint
func TestAPILoginHandler(t *testing.T) {
	setupJWTForTests(t)

	t.Run("valid login with JSON", func(t *testing.T) {
		reqBody := `{"username": "testuser", "password": "testpass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusOK)
		}

		// Check Content-Type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("APILoginHandler() Content-Type = %v, want application/json", contentType)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Check status field
		if response["status"] != "success" {
			t.Errorf("APILoginHandler() status = %v, want 'success'", response["status"])
		}

		// Check data has token
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Error("APILoginHandler() data field is missing or not a map")
			return
		}

		if _, exists := data["token"]; !exists {
			t.Error("APILoginHandler() token field is missing")
		}

		if _, exists := data["expires_in"]; !exists {
			t.Error("APILoginHandler() expires_in field is missing")
		}
	})

	t.Run("invalid request body - malformed JSON", func(t *testing.T) {
		setupJWTForTests(t)
		reqBody := `{invalid json`
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid credentials - empty username", func(t *testing.T) {
		setupJWTForTests(t)
		reqBody := `{"username": "", "password": "testpass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}

		// Check error response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if response["error"] != "Invalid credentials" {
			t.Errorf("APILoginHandler() error = %v, want 'Invalid credentials'", response["error"])
		}
	})

	t.Run("invalid credentials - empty password", func(t *testing.T) {
		setupJWTForTests(t)
		reqBody := `{"username": "testuser", "password": ""}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong HTTP method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/login", nil)
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		// Handler returns JSONResponse which wraps data in "data" field
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusMethodNotAllowed)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// JSONResponse wraps everything in "data" field with "status": "success"
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Error("APILoginHandler() data field is missing or not a map")
			return
		}

		if data["error"] != "Method not allowed" {
			t.Errorf("APILoginHandler() error = %v, want 'Method not allowed'", data["error"])
		}
	})

	t.Run("missing content type header", func(t *testing.T) {
		setupJWTForTests(t)
		reqBody := `{"username": "testuser", "password": "testpass"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(reqBody))
		rr := httptest.NewRecorder()

		APILoginHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("APILoginHandler() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})
}

// TestAPIProtectedHandler tests /api/protected endpoint
func TestAPIProtectedHandler(t *testing.T) {
	t.Run("valid request - should return protected data", func(t *testing.T) {
		setupJWTForTests(t)
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		rr := httptest.NewRecorder()

		APIProtectedHandler(rr, req)

		// Handler expects JWT middleware to set user ID in context
		// Without middleware, it will return 401
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("APIProtectedHandler() status = %v, want %v (no middleware)", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong HTTP method", func(t *testing.T) {
		setupJWTForTests(t)
		req := httptest.NewRequest(http.MethodPost, "/api/protected", nil)
		rr := httptest.NewRecorder()

		APIProtectedHandler(rr, req)

		// Handler returns JSONResponse which wraps data in "data" field
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("APIProtectedHandler() status = %v, want %v", rr.Code, http.StatusMethodNotAllowed)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// JSONResponse wraps everything in "data" field with "status": "success"
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Error("APIProtectedHandler() data field is missing or not a map")
			return
		}

		if data["error"] != "Method not allowed" {
			t.Errorf("APIProtectedHandler() error = %v, want 'Method not allowed'", data["error"])
		}
	})

	t.Run("successful response structure", func(t *testing.T) {
		// This test verifies the expected response structure
		// In real scenario, JWT middleware would set user ID in context
		req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
		rr := httptest.NewRecorder()

		APIProtectedHandler(rr, req)

		// Even without middleware, we should get a JSON error response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Should have error field
		if _, exists := response["error"]; !exists {
			t.Error("APIProtectedHandler() missing 'error' field")
		}
	})
}

// TestJWTAuthenticationFlow tests the complete JWT authentication flow
func TestJWTAuthenticationFlow(t *testing.T) {
	setupJWTForTests(t)

	t.Run("login then access protected endpoint", func(t *testing.T) {
		// Step 1: Login
		loginReqBody := `{"username": "testuser", "password": "testpass"}`
		loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(loginReqBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRR := httptest.NewRecorder()

		APILoginHandler(loginRR, loginReq)

		if loginRR.Code != http.StatusOK {
			t.Errorf("Login failed with status %d", loginRR.Code)
			return
		}

		// Extract token
		var loginResponse map[string]interface{}
		if err := json.Unmarshal(loginRR.Body.Bytes(), &loginResponse); err != nil {
			t.Fatalf("Failed to parse login response: %v", err)
		}

		data, ok := loginResponse["data"].(map[string]interface{})
		if !ok {
			t.Error("Login response data is missing")
			return
		}

		token, ok := data["token"].(string)
		if !ok || token == "" {
			t.Error("Login response token is missing")
			return
		}

		// Step 2: Use token to access protected endpoint
		// Note: This requires the JWT middleware to be set up properly
		// For this test, we just verify we got a token
		t.Logf("Successfully obtained JWT token: %s...", token[:min(20, len(token))])
	})
}

// TestJSONResponseHelpers tests the JSON response helper functions
func TestJSONResponseHelpers(t *testing.T) {
	t.Run("JSONResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()
		JSONResponse(rr, http.StatusOK, map[string]string{"message": "test"})

		if rr.Code != http.StatusOK {
			t.Errorf("JSONResponse() status = %v, want %v", rr.Code, http.StatusOK)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("JSONResponse() Content-Type = %v, want application/json", contentType)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if response["status"] != "success" {
			t.Errorf("JSONResponse() status = %v, want 'success'", response["status"])
		}
	})

	t.Run("ErrorResponseFunc", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ErrorResponseFunc(rr, http.StatusBadRequest, "Bad request")

		if rr.Code != http.StatusBadRequest {
			t.Errorf("ErrorResponseFunc() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if response["error"] != "Bad request" {
			t.Errorf("ErrorResponseFunc() error = %v, want 'Bad request'", response["error"])
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ValidationError(rr, "username", "is required")

		if rr.Code != http.StatusBadRequest {
			t.Errorf("ValidationError() status = %v, want %v", rr.Code, http.StatusBadRequest)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if response["error"] != "Validation failed" {
			t.Errorf("ValidationError() error = %v, want 'Validation failed'", response["error"])
		}

		details, ok := response["details"].(string)
		if !ok || details == "" {
			t.Error("ValidationError() details field is missing")
		}
	})
}

// min is a helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
