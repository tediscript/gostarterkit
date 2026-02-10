package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the health status of the application
type Status string

const (
	StatusOK       Status = "ok"
	StatusNotOK    Status = "not_ok"
	StatusAlive    Status = "alive"
	StatusReady    Status = "ready"
	StatusNotReady Status = "not_ready"
)

// HealthResponse represents the response from health check endpoints
type HealthResponse struct {
	Status    Status            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Timestamp time.Time         `json:"timestamp,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// Checker defines the interface for health check dependencies
type Checker interface {
	Ping(ctx context.Context) error
}

// HealthChecker performs health checks on various dependencies
type HealthChecker struct {
	db Checker
}

// New creates a new HealthChecker instance
func New(db Checker) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

// CheckDatabase verifies database connectivity
func (h *HealthChecker) CheckDatabase(ctx context.Context) error {
	if h.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Use a short timeout for health checks
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return h.db.Ping(ctx)
}

// Healthz returns basic health information
func (h *HealthChecker) Healthz() HealthResponse {
	checks := make(map[string]string)
	dbStatus := "connected"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.CheckDatabase(ctx); err != nil {
		dbStatus = fmt.Sprintf("failed: %v", err)
	}

	checks["database"] = dbStatus

	return HealthResponse{
		Status:    StatusOK,
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}
}

// Livez checks if the application is running (liveness probe)
func (h *HealthChecker) Livez() HealthResponse {
	return HealthResponse{
		Status:    StatusAlive,
		Timestamp: time.Now().UTC(),
	}
}

// Readyz checks if the application is ready to serve traffic (readiness probe)
func (h *HealthChecker) Readyz() HealthResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.CheckDatabase(ctx); err != nil {
		return HealthResponse{
			Status:    StatusNotReady,
			Timestamp: time.Now().UTC(),
			Error:     fmt.Sprintf("database not ready: %v", err),
		}
	}

	return HealthResponse{
		Status:    StatusReady,
		Timestamp: time.Now().UTC(),
	}
}

// WriteJSON writes a HealthResponse as JSON to the provided writer
func WriteJSON(w interface{ Write([]byte) (int, error) }, response HealthResponse) error {
	return json.NewEncoder(w).Encode(response)
}

// MockChecker is a mock implementation of Checker for testing
type MockChecker struct {
	PingFunc func(ctx context.Context) error
}

func (m *MockChecker) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}
