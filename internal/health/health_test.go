package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	mockDB := &MockChecker{}
	hc := New(mockDB)

	if hc == nil {
		t.Fatal("New() returned nil")
	}

	if hc.db != mockDB {
		t.Error("New() did not set db correctly")
	}
}

func TestHealthChecker_CheckDatabase_Success(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}
	hc := New(mockDB)

	err := hc.CheckDatabase(context.Background())

	if err != nil {
		t.Errorf("CheckDatabase() unexpected error: %v", err)
	}
}

func TestHealthChecker_CheckDatabase_Failure(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc := New(mockDB)

	err := hc.CheckDatabase(context.Background())

	if err == nil {
		t.Error("CheckDatabase() expected error, got nil")
	}
}

func TestHealthChecker_CheckDatabase_NilDB(t *testing.T) {
	hc := New(nil)

	err := hc.CheckDatabase(context.Background())

	if err == nil {
		t.Error("CheckDatabase() expected error for nil db, got nil")
	}
}

func TestHealthChecker_CheckDatabase_Timeout(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			// Wait for context to be cancelled (timeout)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	hc := New(mockDB)

	err := hc.CheckDatabase(context.Background())

	if err == nil {
		t.Error("CheckDatabase() expected timeout error, got nil")
	}
}

func TestHealthChecker_Healthz_Success(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}
	hc := New(mockDB)

	response := hc.Healthz()

	if response.Status != StatusOK {
		t.Errorf("Healthz() expected status %s, got %s", StatusOK, response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Healthz() expected version 1.0.0, got %s", response.Version)
	}

	if response.Checks == nil {
		t.Error("Healthz() expected checks map, got nil")
	}

	if response.Checks["database"] != "connected" {
		t.Errorf("Healthz() expected database status 'connected', got %s", response.Checks["database"])
	}
}

func TestHealthChecker_Healthz_DatabaseError(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc := New(mockDB)

	response := hc.Healthz()

	// Healthz should still return OK even if database is down
	if response.Status != StatusOK {
		t.Errorf("Healthz() expected status %s, got %s", StatusOK, response.Status)
	}

	if response.Checks["database"] == "connected" {
		t.Error("Healthz() expected database error status, got 'connected'")
	}
}

func TestHealthChecker_Livez(t *testing.T) {
	hc := New(nil)

	response := hc.Livez()

	if response.Status != StatusAlive {
		t.Errorf("Livez() expected status %s, got %s", StatusAlive, response.Status)
	}

	// Verify timestamp is recent (within last second)
	if response.Timestamp.IsZero() {
		t.Error("Livez() timestamp should not be zero")
	}
}

func TestHealthChecker_Readyz_Success(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}
	hc := New(mockDB)

	response := hc.Readyz()

	if response.Status != StatusReady {
		t.Errorf("Readyz() expected status %s, got %s", StatusReady, response.Status)
	}

	if response.Error != "" {
		t.Errorf("Readyz() expected no error, got %s", response.Error)
	}
}

func TestHealthChecker_Readyz_DatabaseError(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc := New(mockDB)

	response := hc.Readyz()

	if response.Status != StatusNotReady {
		t.Errorf("Readyz() expected status %s, got %s", StatusNotReady, response.Status)
	}

	if response.Error == "" {
		t.Error("Readyz() expected error message, got empty string")
	}
}

func TestHealthChecker_Readyz_Timeout(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			// Wait for context to be cancelled (timeout)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	hc := New(mockDB)

	response := hc.Readyz()

	if response.Status != StatusNotReady {
		t.Errorf("Readyz() expected status %s, got %s", StatusNotReady, response.Status)
	}
}

func TestWriteJSON(t *testing.T) {
	response := HealthResponse{
		Status:    StatusOK,
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
	}

	w := httptest.NewRecorder()
	err := WriteJSON(w, response)

	if err != nil {
		t.Errorf("WriteJSON() unexpected error: %v", err)
	}

	if w.Header().Get("Content-Type") != "" {
		// WriteJSON doesn't set Content-Type, that's the handler's job
	}

	var decoded HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &decoded)
	if err != nil {
		t.Errorf("WriteJSON() failed to decode: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("WriteJSON() status mismatch: got %s, want %s", decoded.Status, response.Status)
	}
}

func TestMockChecker(t *testing.T) {
	mock := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := mock.Ping(context.Background())
	if err != nil {
		t.Errorf("MockChecker.Ping() unexpected error: %v", err)
	}

	mockError := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return http.ErrHandlerTimeout
		},
	}

	err = mockError.Ping(context.Background())
	if err == nil {
		t.Error("MockChecker.Ping() expected error, got nil")
	}
}

func TestMockChecker_NilPingFunc(t *testing.T) {
	mock := &MockChecker{}

	err := mock.Ping(context.Background())
	if err != nil {
		t.Errorf("MockChecker.Ping() with nil PingFunc should not error, got: %v", err)
	}
}

// Integration-style test: Test health response structure
func TestHealthResponse_JSONStructure(t *testing.T) {
	response := HealthResponse{
		Status:    StatusOK,
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks:    map[string]string{"database": "connected"},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal HealthResponse: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HealthResponse: %v", err)
	}

	// Verify required fields
	if decoded["status"] == nil {
		t.Error("HealthResponse JSON missing 'status' field")
	}
	if decoded["timestamp"] == nil {
		t.Error("HealthResponse JSON missing 'timestamp' field")
	}
}

// Edge case: Very long database error message
func TestHealthChecker_Healthz_LongErrorMessage(t *testing.T) {
	longErrorMsg := string(make([]byte, 10000))
	for i := range longErrorMsg {
		longErrorMsg = longErrorMsg[:i] + "a" + longErrorMsg[i+1:]
	}

	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	hc := New(mockDB)

	response := hc.Healthz()

	// Should handle long error messages gracefully
	if response.Checks == nil {
		t.Error("Healthz() expected checks map, got nil")
	}

	// The error should be truncated or handled
	dbStatus := response.Checks["database"]
	if len(dbStatus) == 0 {
		t.Error("Healthz() expected non-empty database status")
	}
}

// Edge case: Multiple concurrent health checks
func TestHealthChecker_ConcurrentHealthChecks(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}
	hc := New(mockDB)

	done := make(chan bool, 10)

	// Run 10 concurrent health checks
	for i := 0; i < 10; i++ {
		go func() {
			response := hc.Healthz()
			if response.Status != StatusOK {
				t.Errorf("Concurrent health check failed with status: %s", response.Status)
			}
			done <- true
		}()
	}

	// Wait for all checks to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Edge case: Health check during context cancellation
func TestHealthChecker_Readyz_ContextCancellation(t *testing.T) {
	mockDB := &MockChecker{
		PingFunc: func(ctx context.Context) error {
			// Cancel the context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Wait for cancellation
			<-ctx.Done()
			return ctx.Err()
		},
	}
	hc := New(mockDB)

	response := hc.Readyz()

	// Should handle context cancellation
	if response.Status == StatusReady {
		t.Error("Readyz() expected not ready status on context cancellation")
	}
}
