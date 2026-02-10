package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/tediscript/gostarterkit/internal/config"
)

// TestServerStartup tests that the server starts successfully on the configured port
func TestServerStartup(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9999
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responding
	resp, err := http.Get("http://localhost:9999/")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Stop server
	if err := srv.Stop(); err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	// Give server time to shutdown
	time.Sleep(200 * time.Millisecond)
}

// TestStopMethod tests the Stop method
func TestStopMethod(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9998
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://localhost:9998/")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()

	// Stop server
	if err := srv.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Give time for shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify server is no longer responding
	resp2, err := http.Get("http://localhost:9998/")
	if err == nil {
		resp2.Body.Close()
		t.Error("Server should not be responding after shutdown")
	}
}

// TestInFlightRequests tests that in-flight requests complete during shutdown
func TestInFlightRequests(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9997
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	requestReceived := make(chan bool, 1)
	requestProcessed := make(chan bool, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- true
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Done"))
		requestProcessed <- true
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Start slow request in goroutine
	go func() {
		resp, err := http.Get("http://localhost:9997/slow")
		if err != nil {
			t.Logf("Request failed: %v", err)
			return
		}
		defer resp.Body.Close()
	}()

	// Wait for request to be received
	<-requestReceived

	// Initiate shutdown immediately after request received
	go func() {
		if err := srv.Stop(); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	// Wait for request to complete (should complete before shutdown timeout)
	select {
	case <-requestProcessed:
		// Request completed successfully during shutdown
	case <-time.After(3 * time.Second):
		t.Error("Request did not complete during shutdown")
	}
}

// TestShutdownTimeout tests shutdown behavior when timeout is exceeded
func TestShutdownTimeout(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9996
	cfg.HTTP.ShutdownTimeout = 100 * time.Millisecond // Very short timeout
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second) // Longer than shutdown timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Done"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Start slow request in goroutine
	go func() {
		resp, err := http.Get("http://localhost:9996/slow")
		if err != nil {
			t.Logf("Request failed (expected): %v", err)
			return
		}
		defer resp.Body.Close()
	}()

	// Give request time to start
	time.Sleep(50 * time.Millisecond)

	// Initiate shutdown - should timeout
	start := time.Now()
	err := srv.Stop()
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected shutdown timeout error")
	}

	// Shutdown should take approximately to timeout duration
	if duration < 50*time.Millisecond || duration > 500*time.Millisecond {
		t.Errorf("Shutdown took %v, expected around 100ms", duration)
	}
}

// TestPortAlreadyInUse tests that server fails to start when port is already in use
func TestPortAlreadyInUse(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9995
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Start first server
	srv1 := New(cfg, mux, logger)
	go func() {
		if err := srv1.Start(); err != nil {
			t.Logf("Server1 stopped: %v", err)
		}
	}()

	// Give first server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second server on same port
	srv2 := New(cfg, mux, logger)
	err := srv2.Start()

	if err == nil {
		t.Error("Expected error when starting server on already-in-use port")
	}

	// Cleanup first server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv1.httpServer.Shutdown(ctx)
}

// TestRapidStartStop tests rapid start/stop cycles
func TestRapidStartStop(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9994
	cfg.HTTP.ShutdownTimeout = 1 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Perform multiple start/stop cycles
	for i := 0; i < 3; i++ {
		srv := New(cfg, mux, logger)

		// Start server
		go func() {
			if err := srv.Start(); err != nil {
				t.Logf("Server stopped: %v", err)
			}
		}()

		// Give server time to start
		time.Sleep(50 * time.Millisecond)

		// Verify server is running
		resp, err := http.Get("http://localhost:9994/")
		if err != nil {
			t.Fatalf("Failed to connect to server on cycle %d: %v", i, err)
		}
		resp.Body.Close()

		// Stop server
		if err := srv.Stop(); err != nil {
			t.Errorf("Failed to stop server on cycle %d: %v", i, err)
		}

		// Give server time to stop
		time.Sleep(100 * time.Millisecond)
	}
}

// TestZeroConnectionsShutdown tests shutdown when no connections are active
func TestZeroConnectionsShutdown(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9993
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown immediately without any requests
	start := time.Now()
	err := srv.Stop()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error during shutdown: %v", err)
	}

	// Shutdown should be fast with no connections
	if duration > 1*time.Second {
		t.Errorf("Shutdown took too long with no connections: %v", duration)
	}
}

// TestMultipleRequests tests handling of multiple concurrent requests
func TestMultipleRequests(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9992
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send multiple concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			resp, err := http.Get("http://localhost:9992/")
			if err != nil {
				t.Errorf("Request failed: %v", err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cleanup
	if err := srv.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

// TestLargeRequestBody tests handling of very large request bodies
func TestLargeRequestBody(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9991
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	requestReceived := make(chan bool, 1)
	requestProcessed := make(chan bool, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- true

		// Read the entire body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify we received the expected data
		expectedSize := 10 * 1024 * 1024 // 10MB
		if len(body) != expectedSize {
			t.Errorf("Expected body size %d, got %d", expectedSize, len(body))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		requestProcessed <- true
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create a large request body (10MB)
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Send large request
	go func() {
		req, err := http.NewRequest("POST", "http://localhost:9991/upload", bytes.NewReader(largeData))
		if err != nil {
			t.Errorf("Failed to create request: %v", err)
			return
		}

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("Request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}()

	// Wait for request to be received
	select {
	case <-requestReceived:
		// Request received successfully
	case <-time.After(5 * time.Second):
		t.Error("Request was not received within timeout")
	}

	// Wait for request to be processed
	select {
	case <-requestProcessed:
		// Request processed successfully
	case <-time.After(5 * time.Second):
		t.Error("Request was not processed within timeout")
	}

	// Cleanup
	if err := srv.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

// TestConcurrentShutdownSignals tests that server handles shutdown correctly even if signals arrive rapidly
func TestConcurrentShutdownSignals(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Port = 9990
	cfg.HTTP.ShutdownTimeout = 5 * time.Second
	cfg.HTTP.ReadTimeout = 5 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 5 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, mux, logger)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server stopped with error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://localhost:9990/")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()

	// Call Stop() multiple times rapidly to simulate concurrent shutdown signals
	// This should not cause a panic or deadlock
	stopDone := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(index int) {
			if err := srv.Stop(); err != nil {
				t.Logf("Stop call %d returned error: %v", index, err)
			}
			stopDone <- true
		}(i)
	}

	// Wait for all Stop calls to complete
	for i := 0; i < 3; i++ {
		select {
		case <-stopDone:
			// Stop completed
		case <-time.After(5 * time.Second):
			t.Errorf("Stop call %d did not complete within timeout", i)
		}
	}

	// Give server time to fully shutdown
	time.Sleep(500 * time.Millisecond)

	// Verify server is no longer responding
	resp2, err := http.Get("http://localhost:9990/")
	if err == nil {
		resp2.Body.Close()
		t.Error("Server should not be responding after shutdown")
	}
}
