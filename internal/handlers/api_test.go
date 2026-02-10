package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockTemplateCache is a mock implementation of TemplateCache for testing
type mockTemplateCache struct{}

func (m *mockTemplateCache) RenderTemplate(wr interface{ Write([]byte) (int, error) }, name string, data TemplateData) error {
	return nil
}

func (m *mockTemplateCache) CheckForReload(templatesDir string) error {
	return nil
}

// TestAPIStatus tests the /api/status endpoint
func TestAPIStatus(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates")

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

// TestAPIHello tests the /api/hello endpoint
func TestAPIHello(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates")

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

// TestAPIError tests the /api/error endpoint with different error types
func TestAPIError(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates")

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

// TestAPIData tests the /api/data endpoint with nested data structures
func TestAPIData(t *testing.T) {
	h := New(&mockTemplateCache{}, "templates")

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
	h := New(&mockTemplateCache{}, "templates")

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
	h := New(&mockTemplateCache{}, "templates")

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
	h := New(&mockTemplateCache{}, "templates")

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
	h := New(&mockTemplateCache{}, "templates")

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
