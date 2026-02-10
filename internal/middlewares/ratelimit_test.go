package middlewares

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestRateLimiterSlidingWindow tests the sliding window algorithm
func TestRateLimiterSlidingWindow(t *testing.T) {
	maxRequests := 5
	windowSize := 1 * time.Second
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	clientID := "192.168.1.1"

	// Make maxRequests requests - all should be allowed
	for i := 0; i < maxRequests; i++ {
		allowed, _ := limiter.isAllowed(clientID)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	allowed, retryAfter := limiter.isAllowed(clientID)
	if allowed {
		t.Errorf("Request %d should be denied (over limit)", maxRequests+1)
	}
	if retryAfter <= 0 {
		t.Errorf("Retry-After should be positive, got %d", retryAfter)
	}

	// Wait for window to expire
	time.Sleep(windowSize + 100*time.Millisecond)

	// Next request should be allowed again
	allowed, retryAfter = limiter.isAllowed(clientID)
	if !allowed {
		t.Errorf("Request after window expiry should be allowed")
	}
	if retryAfter != 0 {
		t.Errorf("Retry-After should be 0 for allowed request, got %d", retryAfter)
	}
}

// TestRateLimiterBoundaryConditions tests boundary conditions
func TestRateLimiterBoundaryConditions(t *testing.T) {
	maxRequests := 3
	windowSize := 2 * time.Second
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	clientID := "192.168.1.2"

	// Test exactly at limit - should be allowed
	for i := 0; i < maxRequests; i++ {
		allowed, _ := limiter.isAllowed(clientID)
		if !allowed {
			t.Errorf("Request %d at exactly limit should be allowed", i+1)
		}
	}

	// Test limit + 1 - should be denied
	allowed, _ := limiter.isAllowed(clientID)
	if allowed {
		t.Errorf("Request at limit+1 should be denied")
	}

	// Test with staggered requests
	time.Sleep(windowSize / 2)
	allowed, _ = limiter.isAllowed(clientID)
	if allowed {
		t.Errorf("Request before window expires should still be denied")
	}

	time.Sleep(windowSize/2 + 100*time.Millisecond)
	allowed, _ = limiter.isAllowed(clientID)
	if !allowed {
		t.Errorf("Request after window expires should be allowed")
	}
}

// TestRateLimiterMultipleClients tests multiple clients
func TestRateLimiterMultipleClients(t *testing.T) {
	maxRequests := 2
	windowSize := 1 * time.Second
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	client1 := "192.168.1.3"
	client2 := "192.168.1.4"

	// Client 1 makes maxRequests
	for i := 0; i < maxRequests; i++ {
		allowed, _ := limiter.isAllowed(client1)
		if !allowed {
			t.Errorf("Client1 request %d should be allowed", i+1)
		}
	}

	// Client 1 should be denied
	allowed, _ := limiter.isAllowed(client1)
	if allowed {
		t.Errorf("Client1 request over limit should be denied")
	}

	// Client 2 should still be allowed (independent limits)
	for i := 0; i < maxRequests; i++ {
		allowed, _ := limiter.isAllowed(client2)
		if !allowed {
			t.Errorf("Client2 request %d should be allowed", i+1)
		}
	}

	// Client 2 should be denied
	allowed, _ = limiter.isAllowed(client2)
	if allowed {
		t.Errorf("Client2 request over limit should be denied")
	}
}

// TestRateLimiterConcurrentRequests tests concurrent requests from same client
func TestRateLimiterConcurrentRequests(t *testing.T) {
	maxRequests := 100
	windowSize := 1 * time.Second
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	clientID := "192.168.1.5"

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Send 150 concurrent requests (over limit)
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.isAllowed(clientID)
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowedCount > maxRequests {
		t.Errorf("Allowed %d requests, but max is %d", allowedCount, maxRequests)
	}
}

// TestRateLimiterCleanup tests the cleanup mechanism
func TestRateLimiterCleanup(t *testing.T) {
	maxRequests := 10
	windowSize := 100 * time.Millisecond
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	// Create multiple clients
	clients := []string{"192.168.1.6", "192.168.1.7", "192.168.1.8", "192.168.1.9", "192.168.1.10"}

	for _, client := range clients {
		limiter.isAllowed(client)
	}

	// Verify clients exist
	limiter.mu.RLock()
	initialCount := len(limiter.clients)
	limiter.mu.RUnlock()

	if initialCount != len(clients) {
		t.Errorf("Expected %d clients, got %d", len(clients), initialCount)
	}

	// Wait for cleanup (window + one ticker cycle)
	time.Sleep(windowSize + 200*time.Millisecond)

	// Verify clients were cleaned up
	limiter.mu.RLock()
	cleanedCount := len(limiter.clients)
	limiter.mu.RUnlock()

	if cleanedCount >= initialCount {
		t.Errorf("Expected clients to be cleaned up, but still have %d", cleanedCount)
	}
}

// TestGetClientIP tests IP address extraction
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.168.1.1:8080",
			headers:    nil,
			expected:   "192.168.1.1",
		},
		{
			name:       "IPv4 without port",
			remoteAddr: "192.168.1.1",
			headers:    nil,
			expected:   "192.168.1.1",
		},
		{
			name:       "IPv6 with brackets and port",
			remoteAddr: "[::1]:8080",
			headers:    nil,
			expected:   "::1",
		},
		{
			name:       "IPv6 without brackets",
			remoteAddr: "::1",
			headers:    nil,
			expected:   "::1",
		},
		{
			name:       "IPv6 full address with brackets",
			remoteAddr: "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080",
			headers:    nil,
			expected:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		},
		{
			name:       "X-Forwarded-For header",
			remoteAddr: "192.168.1.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 203.0.113.2"},
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "192.168.1.1:8080",
			headers:    map[string]string{"X-Real-IP": "203.0.113.3"},
			expected:   "203.0.113.3",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "192.168.1.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.4", "X-Real-IP": "203.0.113.5"},
			expected:   "203.0.113.4",
		},
		{
			name:       "Invalid RemoteAddr returns as-is",
			remoteAddr: "invalid-addr",
			headers:    nil,
			expected:   "invalid-addr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

// TestRateLimitMiddlewareIntegration tests the middleware with HTTP requests
func TestRateLimitMiddlewareIntegration(t *testing.T) {
	maxRequests := 3
	windowSize := 1 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Make maxRequests requests - all should succeed
	for i := 0; i < maxRequests; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// Next request should be rate limited
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request over limit failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}

	// Check Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("Expected Retry-After header to be set")
	}

	// Check response body is JSON
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if result["error"] != "rate_limit_exceeded" {
		t.Errorf("Expected error 'rate_limit_exceeded', got %v", result["error"])
	}

	// Wait for window to expire
	time.Sleep(windowSize + 100*time.Millisecond)

	// Request should succeed again
	resp, err = http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request after window expiry failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after window expiry, got %d", resp.StatusCode)
	}
}

// TestRateLimitMiddlewareHealthEndpoints tests that health endpoints bypass rate limiting
func TestRateLimitMiddlewareHealthEndpoints(t *testing.T) {
	maxRequests := 1
	windowSize := 10 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	healthEndpoints := []string{"/healthz", "/livez", "/readyz"}

	for _, endpoint := range healthEndpoints {
		// Make multiple requests to health endpoint - all should succeed
		for i := 0; i < 5; i++ {
			resp, err := http.Get(server.URL + endpoint)
			if err != nil {
				t.Fatalf("Health endpoint %s request %d failed: %v", endpoint, i+1, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Health endpoint %s request %d expected status 200, got %d", endpoint, i+1, resp.StatusCode)
			}
		}
	}
}

// TestRateLimitMiddlewareIPv6 tests IPv6 address handling
func TestRateLimitMiddlewareIPv6(t *testing.T) {
	maxRequests := 2
	windowSize := 1 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Create request with IPv6 address
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.RemoteAddr = "[::1]:12345"

	// Make maxRequests from IPv6
	for i := 0; i < maxRequests; i++ {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("IPv6 request %d failed: %v", i+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("IPv6 request %d expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// Next request from same IPv6 should be rate limited
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("IPv6 request over limit failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("IPv6 request over limit expected status 429, got %d", resp.StatusCode)
	}
}

// TestRateLimitMiddlewareInvalidIP tests handling of invalid IP addresses
func TestRateLimitMiddlewareInvalidIP(t *testing.T) {
	maxRequests := 1
	windowSize := 1 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Create request with invalid RemoteAddr
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.RemoteAddr = "invalid-ip-address"

	// Request should still work (limiter uses the invalid string as client ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request with invalid IP failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Request with invalid IP expected status 200, got %d", resp.StatusCode)
	}
}

// TestRateLimitMiddlewareUnicodeInXForwardedFor tests unicode/special characters in headers
func TestRateLimitMiddlewareUnicodeInXForwardedFor(t *testing.T) {
	maxRequests := 1
	windowSize := 1 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Create request with unicode in X-Forwarded-For
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 203.0.113.1, test🎉")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request with unicode in header failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Request with unicode in header expected status 200, got %d", resp.StatusCode)
	}
}

// TestRateLimitMiddlewareResetTiming tests rate limit reset timing precision
func TestRateLimitMiddlewareResetTiming(t *testing.T) {
	maxRequests := 5
	windowSize := 500 * time.Millisecond
	limiter := NewRateLimiter(maxRequests, windowSize)
	defer limiter.Stop()

	clientID := "192.168.1.20"

	// Fill up the limit
	for i := 0; i < maxRequests; i++ {
		allowed, _ := limiter.isAllowed(clientID)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be denied
	allowed, _ := limiter.isAllowed(clientID)
	if allowed {
		t.Error("Should be denied after reaching limit")
	}

	// Wait just before window expires
	time.Sleep(windowSize - 50*time.Millisecond)
	allowed, _ = limiter.isAllowed(clientID)
	if allowed {
		t.Error("Should still be denied just before window expires")
	}

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)
	allowed, _ = limiter.isAllowed(clientID)
	if !allowed {
		t.Error("Should be allowed after window expires")
	}
}

// TestRateLimitMiddlewareRetryAfterHeader tests Retry-After header accuracy
func TestRateLimitMiddlewareRetryAfterHeader(t *testing.T) {
	maxRequests := 2
	windowSize := 2 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Fill up the limit
	for i := 0; i < maxRequests; i++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		resp.Body.Close()
	}

	// Get rate limited response
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Rate limited request failed: %v", err)
	}
	defer resp.Body.Close()

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Expected Retry-After header to be set")
	}

	// Retry-After should be positive and reasonable (close to window size)
	// It might be slightly less due to timing, but should be > 0
	if retryAfter == "0" {
		t.Error("Retry-After should be greater than 0")
	}

	// Parse and verify it's a valid number
	var retryAfterInt int
	if _, err := fmt.Sscanf(retryAfter, "%d", &retryAfterInt); err != nil {
		t.Fatalf("Retry-After is not a valid number: %s", retryAfter)
	}

	if retryAfterInt <= 0 || retryAfterInt > int(windowSize.Seconds())+1 {
		t.Errorf("Retry-After %d is out of expected range", retryAfterInt)
	}
}

// TestRateLimitMiddlewareErrorResponse tests error response format
func TestRateLimitMiddlewareErrorResponse(t *testing.T) {
	maxRequests := 1
	windowSize := 1 * time.Second

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(maxRequests, windowSize)
	protectedHandler := middleware(handler)

	server := httptest.NewServer(protectedHandler)
	defer server.Close()

	// Make request to fill limit
	resp, _ := http.Get(server.URL)
	resp.Body.Close()

	// Get rate limited response
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Rate limited request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if result["error"] != "rate_limit_exceeded" {
		t.Errorf("Expected error 'rate_limit_exceeded', got %v", result["error"])
	}

	message, ok := result["message"].(string)
	if !ok {
		t.Fatal("Expected message field to be a string")
	}

	if message == "" {
		t.Error("Expected non-empty message")
	}
}
