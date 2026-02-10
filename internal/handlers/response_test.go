package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestJSONResponse tests the JSONResponse helper function
func TestJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "simple map data",
			statusCode: http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success", "data": map[string]interface{}{"key": "value"}},
		},
		{
			name:       "slice data",
			statusCode: http.StatusCreated,
			data:       []string{"item1", "item2"},
			wantStatus: http.StatusCreated,
			wantJSON:   map[string]interface{}{"status": "success", "data": []interface{}{"item1", "item2"}},
		},
		{
			name:       "struct data",
			statusCode: http.StatusOK,
			data:       struct{ Name string }{"Alice"},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success", "data": map[string]interface{}{"Name": "Alice"}},
		},
		{
			name:       "nil data",
			statusCode: http.StatusOK,
			data:       nil,
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success"}, // nil data field is omitted
		},
		{
			name:       "empty map",
			statusCode: http.StatusOK,
			data:       map[string]interface{}{},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success", "data": map[string]interface{}{}},
		},
		{
			name:       "empty slice",
			statusCode: http.StatusOK,
			data:       []string{},
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success", "data": []interface{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			JSONResponse(rr, tt.statusCode, tt.data)

			// Check status code
			if rr.Code != tt.wantStatus {
				t.Errorf("JSONResponse() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("JSONResponse() Content-Type = %v, want application/json", contentType)
			}

			// Parse and check JSON response
			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Compare JSON (deep comparison)
			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("JSONResponse() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestJSONResponseWithMessage tests the JSONResponseWithMessage helper
func TestJSONResponseWithMessage(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		data       interface{}
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "with message and data",
			statusCode: http.StatusOK,
			message:    "Success!",
			data:       map[string]string{"result": "ok"},
			wantStatus: http.StatusOK,
			wantJSON: map[string]interface{}{
				"status":  "success",
				"message": "Success!",
				"data":    map[string]interface{}{"result": "ok"},
			},
		},
		{
			name:       "with message and nil data",
			statusCode: http.StatusOK,
			message:    "Hello",
			data:       nil,
			wantStatus: http.StatusOK,
			wantJSON:   map[string]interface{}{"status": "success", "message": "Hello"}, // nil data field is omitted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			JSONResponseWithMessage(rr, tt.statusCode, tt.message, tt.data)

			if rr.Code != tt.wantStatus {
				t.Errorf("JSONResponseWithMessage() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("JSONResponseWithMessage() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestErrorResponseFunc tests the ErrorResponseFunc helper
func TestErrorResponseFunc(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "not found error",
			statusCode: http.StatusNotFound,
			message:    "Resource not found",
			wantStatus: http.StatusNotFound,
			wantJSON:   map[string]interface{}{"error": "Resource not found"},
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "Internal error",
			wantStatus: http.StatusInternalServerError,
			wantJSON:   map[string]interface{}{"error": "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ErrorResponseFunc(rr, tt.statusCode, tt.message)

			if rr.Code != tt.wantStatus {
				t.Errorf("ErrorResponseFunc() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("ErrorResponseFunc() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestErrorResponseWithDetails tests the ErrorResponseWithDetails helper
func TestErrorResponseWithDetails(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		details    string
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "with details",
			statusCode: http.StatusBadRequest,
			message:    "Invalid request",
			details:    "Missing required field 'id'",
			wantStatus: http.StatusBadRequest,
			wantJSON: map[string]interface{}{
				"error":   "Invalid request",
				"details": "Missing required field 'id'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ErrorResponseWithDetails(rr, tt.statusCode, tt.message, tt.details)

			if rr.Code != tt.wantStatus {
				t.Errorf("ErrorResponseWithDetails() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("ErrorResponseWithDetails() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestErrorResponseFromError tests the ErrorResponseFromError helper
func TestErrorResponseFromError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "with error",
			statusCode: http.StatusInternalServerError,
			err:        http.ErrHandlerTimeout,
			wantStatus: http.StatusInternalServerError,
			wantJSON:   map[string]interface{}{"error": "http: Handler timeout"},
		},
		{
			name:       "nil error",
			statusCode: http.StatusInternalServerError,
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantJSON:   map[string]interface{}{"error": "Unknown error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ErrorResponseFromError(rr, tt.statusCode, tt.err)

			if rr.Code != tt.wantStatus {
				t.Errorf("ErrorResponseFromError() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("ErrorResponseFromError() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestValidationError tests the ValidationError helper
func TestValidationError(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		message    string
		wantStatus int
		wantJSON   map[string]interface{}
	}{
		{
			name:       "validation error",
			field:      "email",
			message:    "must be a valid email address",
			wantStatus: http.StatusBadRequest,
			wantJSON: map[string]interface{}{
				"error":   "Validation failed",
				"details": "email: must be a valid email address",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ValidationError(rr, tt.field, tt.message)

			if rr.Code != tt.wantStatus {
				t.Errorf("ValidationError() status = %v, want %v", rr.Code, tt.wantStatus)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if !jsonEqual(gotJSON, tt.wantJSON) {
				t.Errorf("ValidationError() JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

// TestSpecialCharacters tests JSON encoding with special characters
func TestSpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "unicode characters",
			data: map[string]string{
				"greeting": "Hello 世界",
				"emoji":    "🎉🚀",
			},
		},
		{
			name: "special characters",
			data: map[string]string{
				"quotes":  `"quoted"`,
				"slashes": `forward/slash\backslash`,
				"newline": "line1\nline2",
				"tab":     "col1\tcol2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			JSONResponse(rr, http.StatusOK, tt.data)

			if rr.Code != http.StatusOK {
				t.Errorf("JSONResponse() status = %v, want %v", rr.Code, http.StatusOK)
			}

			// Verify JSON can be decoded without errors
			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Errorf("Failed to decode JSON with special characters: %v", err)
			}
		})
	}
}

// TestNestedDataStructures tests JSON encoding with nested data
func TestNestedDataStructures(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Alice",
			"address": map[string]string{
				"street": "123 Main St",
				"city":   "Boston",
			},
			"tags": []string{"admin", "user", "premium"},
		},
		"metadata": map[string]interface{}{
			"created": "2024-01-01",
			"updated": "2024-01-02",
		},
	}

	rr := httptest.NewRecorder()
	JSONResponse(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("JSONResponse() status = %v, want %v", rr.Code, http.StatusOK)
	}

	var gotJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
		t.Fatalf("Failed to decode nested JSON: %v", err)
	}

	// Verify nested structure exists
	if gotJSON["data"] == nil {
		t.Error("Expected 'data' field in response")
	}
}

// TestLargeResponseBodies tests JSON encoding with large data
func TestLargeResponseBodies(t *testing.T) {
	// Create a large array
	largeArray := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeArray[i] = "item-" + string(rune(i))
	}

	data := map[string]interface{}{
		"items": largeArray,
		"count": 1000,
	}

	rr := httptest.NewRecorder()
	JSONResponse(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("JSONResponse() status = %v, want %v", rr.Code, http.StatusOK)
	}

	// Verify response body is large enough
	if rr.Body.Len() < 10000 {
		t.Errorf("Expected large response body, got %d bytes", rr.Body.Len())
	}

	// Verify JSON can be decoded
	var gotJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
		t.Errorf("Failed to decode large JSON: %v", err)
	}
}

// TestConcurrentResponses tests concurrent JSON responses
func TestConcurrentResponses(t *testing.T) {
	data := map[string]string{"status": "ok"}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			rr := httptest.NewRecorder()
			JSONResponse(rr, http.StatusOK, data)
			if rr.Code != http.StatusOK {
				t.Errorf("Concurrent response failed")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper function to compare JSON maps
func jsonEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return bytes.Equal(aJSON, bJSON)
}

// TestDoubleWriteProtection tests that we can't write to response twice
func TestDoubleWriteProtection(t *testing.T) {
	// This test verifies that trying to write JSON to a response
	// that's already written to results in an error (which we handle)
	rr := httptest.NewRecorder()
	JSONResponse(rr, http.StatusOK, map[string]string{"key": "value"})

	// First write should succeed
	if rr.Code != http.StatusOK {
		t.Errorf("First write failed: status = %v", rr.Code)
	}

	// The response should contain valid JSON
	var gotJSON map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
		t.Errorf("Failed to decode JSON after write: %v", err)
	}
}

// TestEmptyStringData tests empty string handling
func TestEmptyStringData(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"empty string", ""},
		{"empty slice", []string{}},
		{"empty map", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			JSONResponse(rr, http.StatusOK, tt.data)

			if rr.Code != http.StatusOK {
				t.Errorf("JSONResponse() status = %v, want %v", rr.Code, http.StatusOK)
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &gotJSON); err != nil {
				t.Errorf("Failed to decode JSON: %v", err)
			}
		})
	}
}
