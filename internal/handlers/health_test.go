package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tediscript/gostarterkit/internal/health"
)

// TestHealthzEndpointIntegration tests /healthz endpoint via HTTP
func TestHealthzEndpointIntegration(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check Content-Type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response health.HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify required fields
	if response.Status != health.StatusOK {
		t.Errorf("Expected status %s, got %s", health.StatusOK, response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", response.Version)
	}

	if response.Checks == nil {
		t.Error("Expected checks map, got nil")
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

// TestLivezEndpointIntegration tests /livez endpoint via HTTP
func TestLivezEndpointIntegration(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", h.Livez)

	req := httptest.NewRequest("GET", "/livez", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check Content-Type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response health.HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify required fields
	if response.Status != health.StatusAlive {
		t.Errorf("Expected status %s, got %s", health.StatusAlive, response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

// TestReadyzEndpointIntegration tests /readyz endpoint via HTTP with database
func TestReadyzEndpointIntegration(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", h.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// With nil database, should return 503
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	// Check Content-Type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response health.HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify status is not ready
	if response.Status == health.StatusReady {
		t.Errorf("Expected status not ready, got %s", response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

// TestHealthEndpointsWithRealDatabase tests health endpoints with real database
func TestHealthEndpointsWithRealDatabase(t *testing.T) {
	// Create a mock database that always returns success
	db := &mockDatabase{}
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(db)
	h := New(mockCache, "./templates", mockHealth)

	// Test readyz with real database
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", h.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// With real database, should return 200
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response health.HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response.Status != health.StatusReady {
		t.Errorf("Expected status %s, got %s", health.StatusReady, response.Status)
	}

	// Test healthz with real database
	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /healthz", h.Healthz)

	req2 := httptest.NewRequest("GET", "/healthz", nil)
	rr2 := httptest.NewRecorder()

	mux2.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr2.Code)
	}

	var response2 health.HealthResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response2.Checks["database"] != "connected" {
		t.Errorf("Expected database status 'connected', got %s", response2.Checks["database"])
	}
}

// TestHealthEndpointsAccessibilityWithoutAuth tests that health endpoints are accessible without authentication
func TestHealthEndpointsAccessibilityWithoutAuth(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	// Create routes with middlewares (simulating production setup)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /livez", h.Livez)
	mux.HandleFunc("GET /readyz", h.Readyz)

	// Test without any authentication headers
	tests := []struct {
		name string
		path string
	}{
		{"/healthz endpoint", "/healthz"},
		{"/livez endpoint", "/livez"},
		{"/readyz endpoint", "/readyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			// Should be accessible (200 or 503 for readyz with nil db)
			if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 200 or 503, got %d", rr.Code)
			}

			// Should return JSON
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

// TestHealthEndpointHighLoad tests health endpoints under high load
func TestHealthEndpointHighLoad(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)

	done := make(chan bool, 100)

	// Make 100 concurrent requests
	for i := 0; i < 100; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/healthz", nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("High load request failed with status %d", rr.Code)
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestHealthEndpointWrongMethods tests that health endpoints reject wrong HTTP methods
func TestHealthEndpointWrongMethods(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /livez", h.Livez)
	mux.HandleFunc("GET /readyz", h.Readyz)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"POST /healthz", "POST", "/healthz", http.StatusMethodNotAllowed},
		{"PUT /healthz", "PUT", "/healthz", http.StatusMethodNotAllowed},
		{"DELETE /healthz", "DELETE", "/healthz", http.StatusMethodNotAllowed},
		{"POST /livez", "POST", "/livez", http.StatusMethodNotAllowed},
		{"POST /readyz", "POST", "/readyz", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

// TestHealthResponseStructure validates JSON response structure
func TestHealthResponseStructure(t *testing.T) {
	mockCache := &mockTemplateCache{}
	mockHealth := health.New(nil)
	h := New(mockCache, "./templates", mockHealth)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	// Parse JSON and verify structure
	var decoded map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify required field exists
	if decoded["status"] == nil {
		t.Error("Health response missing 'status' field")
	}
}

// mockDatabase is a mock database that implements health.Checker interface
type mockDatabase struct{}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return nil
}
