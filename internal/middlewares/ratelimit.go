package middlewares

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// clientInfo tracks request timestamps for a single client
type clientInfo struct {
	requests []time.Time
	mu       sync.RWMutex
}

// RateLimiter implements a sliding window rate limiting algorithm
type RateLimiter struct {
	clients       map[string]*clientInfo
	maxRequests   int
	windowSize    time.Duration
	cleanupTicker *time.Ticker
	mu            sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:     make(map[string]*clientInfo),
		maxRequests: maxRequests,
		windowSize:  windowSize,
	}

	// Start cleanup ticker to remove old client entries
	// Run cleanup every window duration to prevent memory leaks
	rl.cleanupTicker = time.NewTicker(windowSize)
	go rl.cleanup()

	return rl
}

// cleanup removes client entries that have no recent requests
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()
		now := time.Now()

		for clientID, client := range rl.clients {
			client.mu.RLock()
			if len(client.requests) == 0 {
				// Remove client with no requests
				delete(rl.clients, clientID)
				client.mu.RUnlock()
				continue
			}

			// Check if all requests are outside the window
			oldestRequest := client.requests[0]
			client.mu.RUnlock()

			if now.Sub(oldestRequest) > rl.windowSize {
				// All requests are outside the window, safe to remove
				delete(rl.clients, clientID)
			}
		}

		rl.mu.Unlock()
	}
}

// Stop stops the cleanup ticker
func (rl *RateLimiter) Stop() {
	rl.cleanupTicker.Stop()
}

// isAllowed checks if a request from the given client is allowed
func (rl *RateLimiter) isAllowed(clientID string) (bool, int) {
	rl.mu.Lock()
	client, exists := rl.clients[clientID]
	if !exists {
		client = &clientInfo{
			requests: make([]time.Time, 0),
		}
		rl.clients[clientID] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Remove requests outside the window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Check if limit exceeded
	if len(client.requests) >= rl.maxRequests {
		// Calculate retry-after based on oldest request
		oldestRequest := client.requests[0]
		timeUntilExpiry := rl.windowSize - now.Sub(oldestRequest)
		retryAfter := int(timeUntilExpiry.Seconds()) + 1
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}

	// Add current request
	client.requests = append(client.requests, now)
	return true, 0
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		if idx := strings.Index(xff, ","); idx != -1 {
			xff = xff[:idx]
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, return the RemoteAddr as-is
		return r.RemoteAddr
	}

	// Handle IPv6 addresses with brackets (e.g., [::1]:8080)
	ip = strings.Trim(ip, "[]")
	return ip
}

// RateLimitMiddleware returns a middleware that implements rate limiting
func RateLimitMiddleware(maxRequests int, windowSize time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(maxRequests, windowSize)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health endpoints (they're used by monitoring)
			if r.URL.Path == "/healthz" || r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r)

			// Check if request is allowed
			allowed, retryAfter := limiter.isAllowed(clientIP)

			if !allowed {
				// Return 429 Too Many Requests
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.WriteHeader(http.StatusTooManyRequests)

				// Write JSON error response
				w.Write([]byte(`{
					"error": "rate_limit_exceeded",
					"message": "Too many requests. Please try again later."
				}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
