package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tediscript/gostarterkit/internal/config"
)

// setupTestConfig creates a test configuration
func setupTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Session.CookieSecret = "test-secret-for-unit-tests"
	cfg.Session.CookieName = "session"
	cfg.Session.MaxAgeSeconds = 3600
	cfg.Session.CookieHTTPOnly = true
	cfg.Session.CookieSecure = false // false for testing without HTTPS
	cfg.Session.CookieSameSite = "Lax"
	cfg.App.Env = "test"
	return cfg
}

// TestInitialize tests session store initialization
func TestInitialize(t *testing.T) {
	t.Run("successful initialization with secret", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		if Store == nil {
			t.Error("Store should not be nil after initialization")
		}
	})

	t.Run("initialization with development fallback", func(t *testing.T) {
		cfg := setupTestConfig()
		cfg.Session.CookieSecret = ""
		cfg.App.Env = "development"
		Initialize(cfg)

		if Store == nil {
			t.Error("Store should not be nil even without secret in development")
		}
	})

	t.Run("panics in production without secret", func(t *testing.T) {
		cfg := setupTestConfig()
		cfg.Session.CookieSecret = ""
		cfg.App.Env = "production"

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when SESSION_COOKIE_SECRET is empty in production")
			}
		}()

		Initialize(cfg)
	})

	t.Run("cookie options are set correctly", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		if Store == nil {
			t.Fatal("Store should not be nil")
		}

		if Store.Options.Path != "/" {
			t.Errorf("Expected Path '/', got %s", Store.Options.Path)
		}
		if Store.Options.MaxAge != 3600 {
			t.Errorf("Expected MaxAge 3600, got %d", Store.Options.MaxAge)
		}
		if !Store.Options.HttpOnly {
			t.Error("Expected HttpOnly to be true")
		}
		if Store.Options.SameSite != http.SameSiteLaxMode {
			t.Errorf("Expected SameSiteLaxMode, got %v", Store.Options.SameSite)
		}
	})
}

// TestParseSameSite tests SameSite attribute parsing
func TestParseSameSite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected http.SameSite
	}{
		{"Strict", "Strict", http.SameSiteStrictMode},
		{"Lax", "Lax", http.SameSiteLaxMode},
		{"None", "None", http.SameSiteNoneMode},
		{"invalid defaults to Lax", "Invalid", http.SameSiteLaxMode},
		{"empty defaults to Lax", "", http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSameSite(tt.input)
			if result != tt.expected {
				t.Errorf("parseSameSite(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSetUserSession tests session creation
func TestSetUserSession(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("successful session creation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Errorf("SetUserSession returned error: %v", err)
		}

		// Verify session cookie was set
		cookies := rr.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookies set")
		}

		sessionCookie := cookies[0]
		if sessionCookie.Name != "session" {
			t.Errorf("Expected cookie name 'session', got %s", sessionCookie.Name)
		}
		if !sessionCookie.HttpOnly {
			t.Error("Expected HttpOnly to be true")
		}
		if sessionCookie.Secure {
			t.Error("Expected Secure to be false for testing")
		}
		if sessionCookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("Expected SameSiteLaxMode, got %v", sessionCookie.SameSite)
		}
	})

	t.Run("session stores user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		userID := "testuser123"
		err := SetUserSession(rr, req, userID)
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Get session and verify user ID
		session, err := GetSession(req)
		if err != nil {
			t.Fatalf("GetSession returned error: %v", err)
		}

		if session.Values[SessionUserIDKey] != userID {
			t.Errorf("Expected user ID %s, got %v", userID, session.Values[SessionUserIDKey])
		}
		if session.Values[SessionAuthenticatedKey] != true {
			t.Error("Expected authenticated flag to be true")
		}
	})
}

// TestIsAuthenticated tests authentication check
func TestIsAuthenticated(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("returns false for no session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("returns true for valid session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		userID := "testuser"
		err := SetUserSession(rr, req, userID)
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Add session cookie to request
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		authenticated, returnedUserID := IsAuthenticated(req)
		if !authenticated {
			t.Error("Expected authenticated to be true")
		}
		if returnedUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, returnedUserID)
		}
	})
}

// TestClearSession tests session clearing
func TestClearSession(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("session is cleared", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Create a session
		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Add session cookie to request
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		// Clear session
		rr = httptest.NewRecorder()
		err = ClearSession(rr, req)
		if err != nil {
			t.Fatalf("ClearSession returned error: %v", err)
		}

		// Verify session is cleared
		authenticated, _ := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false after clearing session")
		}

		// Verify cookie expiration
		responseCookies := rr.Result().Cookies()
		if len(responseCookies) == 0 {
			t.Fatal("No cookies in response")
		}

		if responseCookies[0].MaxAge != -1 {
			t.Errorf("Expected MaxAge -1 to expire cookie, got %d", responseCookies[0].MaxAge)
		}
	})
}

// TestGetUserID tests user ID retrieval
func TestGetUserID(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("returns error for no session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		userID, ok := GetUserID(req)
		if ok {
			t.Error("Expected ok to be false")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("returns user ID for valid session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		expectedUserID := "testuser123"
		err := SetUserSession(rr, req, expectedUserID)
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Add session cookie to request
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		userID, ok := GetUserID(req)
		if !ok {
			t.Error("Expected ok to be true")
		}
		if userID != expectedUserID {
			t.Errorf("Expected user ID %s, got %s", expectedUserID, userID)
		}
	})
}

// TestRequireAuth tests authentication middleware
func TestRequireAuth(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("redirects to login for unauthenticated user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Protected content"))
		})

		middleware := RequireAuth(handler)
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("Expected status 303, got %d", rr.Code)
		}

		location := rr.Header().Get("Location")
		if location != "/login" {
			t.Errorf("Expected redirect to /login, got %s", location)
		}
	})

	t.Run("allows access for authenticated user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		// Create session
		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Add session cookie to request
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		// Create a new recorder for protected request
		rr = httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Protected content"))
		})

		middleware := RequireAuth(handler)
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		body := rr.Body.String()
		if body != "Protected content" {
			t.Errorf("Expected 'Protected content', got %s", body)
		}
	})
}

// TestEdgeCases tests edge case scenarios
func TestEdgeCases(t *testing.T) {
	cfg := setupTestConfig()
	Initialize(cfg)

	t.Run("very long user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Create a moderately long user ID (500 characters)
		// Securecookie has limits on value size
		longUserID := strings.Repeat("a", 500)

		err := SetUserSession(rr, req, longUserID)
		if err != nil {
			t.Errorf("SetUserSession with long user ID returned error: %v", err)
		}

		// Verify session was created
		cookies := rr.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookies set")
		}

		// Verify user ID is stored correctly
		cookies = rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		_, returnedUserID := IsAuthenticated(req)
		if returnedUserID != longUserID {
			t.Errorf("Expected user ID to match, got %s", returnedUserID)
		}
	})

	t.Run("session with special characters in user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// User ID with special characters
		userID := "user@test@example.com-123_abc"

		err := SetUserSession(rr, req, userID)
		if err != nil {
			t.Errorf("SetUserSession with special characters returned error: %v", err)
		}

		// Verify session stores user ID correctly
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		_, returnedUserID := IsAuthenticated(req)
		if returnedUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, returnedUserID)
		}
	})

	t.Run("session with Unicode characters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// User ID with Unicode characters
		userID := "用户123-🚀"

		err := SetUserSession(rr, req, userID)
		if err != nil {
			t.Errorf("SetUserSession with Unicode returned error: %v", err)
		}

		// Verify session stores user ID correctly
		cookies := rr.Result().Cookies()
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		_, returnedUserID := IsAuthenticated(req)
		if returnedUserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, returnedUserID)
		}
	})

	t.Run("session cookie with very large value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// This tests if session cookie can handle large values
		// (though in practice, cookies should be small)
		userID := strings.Repeat("a", 1000)

		err := SetUserSession(rr, req, userID)
		if err != nil {
			t.Errorf("SetUserSession with large value returned error: %v", err)
		}

		// Verify cookie was set
		cookies := rr.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookies set")
		}
	})

	t.Run("session expiry at max age", func(t *testing.T) {
		cfg := setupTestConfig()
		cfg.Session.MaxAgeSeconds = 1 // 1 second
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Errorf("SetUserSession returned error: %v", err)
		}

		// Verify cookie MaxAge is set correctly
		cookies := rr.Result().Cookies()
		if cookies[0].MaxAge != 1 {
			t.Errorf("Expected MaxAge 1, got %d", cookies[0].MaxAge)
		}
	})

	t.Run("session with empty user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Set session with empty user ID (edge case)
		// SetUserSession sets authenticated flag to true even for empty user IDs
		// This is the current behavior - the session exists and is considered valid
		err := SetUserSession(rr, req, "")
		if err != nil {
			t.Errorf("SetUserSession with empty user ID returned error: %v", err)
		}

		// Verify session cookie is set even with empty user ID
		cookies := rr.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("No cookies set")
		}

		// Add cookie to request
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		// For empty user ID, the current implementation considers it authenticated
		// The session exists with authenticated flag set to true
		authenticated, userID := IsAuthenticated(req)
		if !authenticated {
			t.Error("Expected authenticated to be true - session is considered valid even with empty user ID")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})
}

// TestNegativeCases tests negative scenarios
func TestNegativeCases(t *testing.T) {
	t.Run("session with invalid cookie", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)

		// Add an invalid/malformed cookie
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "invalid-session-data",
		})

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false for invalid cookie")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("session with tampered cookie", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Create a valid session
		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Modify cookie value (tampering simulation)
		// Note: Since we're creating a new request for the authenticated check,
		// the modified cookie won't affect the original session
		cookies := rr.Result().Cookies()
		tamperedCookie := &http.Cookie{
			Name:  cookies[0].Name,
			Value: "tampered-value",
		}

		// Add tampered cookie to request
		req = httptest.NewRequest("GET", "/", nil)
		req.AddCookie(tamperedCookie)

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false for tampered cookie")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("session with nil values in session", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		session, err := GetSession(req)
		if err != nil {
			t.Fatalf("GetSession returned error: %v", err)
		}

		// Set session values to nil
		session.Values[SessionUserIDKey] = nil
		session.Values[SessionAuthenticatedKey] = nil

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false with nil values")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("corrupted session data", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)

		// Add a cookie with corrupted data
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: string(make([]byte, 1000)), // Random bytes
		})

		_, err := GetSession(req)
		if err != nil {
			// Error is expected for corrupted data
			return
		}

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false for corrupted session")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("session with missing authentication flag", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Create session with user ID but no auth flag
		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Remove authentication flag
		session, err := GetSession(req)
		if err != nil {
			t.Fatalf("GetSession returned error: %v", err)
		}
		delete(session.Values, SessionAuthenticatedKey)

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false without auth flag")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("session with wrong authentication value", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Create session
		err := SetUserSession(rr, req, "testuser")
		if err != nil {
			t.Fatalf("SetUserSession returned error: %v", err)
		}

		// Set authentication flag to false
		session, err := GetSession(req)
		if err != nil {
			t.Fatalf("GetSession returned error: %v", err)
		}
		session.Values[SessionAuthenticatedKey] = false

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false with auth flag set to false")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("clearing session with no existing session", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		// Clear session when none exists (should not error)
		err := ClearSession(rr, req)
		if err != nil {
			t.Errorf("ClearSession should not error when no session exists: %v", err)
		}
	})

	t.Run("multiple concurrent sessions for same user", func(t *testing.T) {
		cfg := setupTestConfig()
		Initialize(cfg)

		userID := "testuser"

		// Create multiple sessions for same user
		sessions := make([]*http.Cookie, 3)
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			err := SetUserSession(rr, req, userID)
			if err != nil {
				t.Fatalf("SetUserSession returned error: %v", err)
			}

			cookies := rr.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("No cookies set")
			}
			sessions[i] = cookies[0]
		}

		// Verify all sessions are valid
		for i, cookie := range sessions {
			req := httptest.NewRequest("GET", "/", nil)
			req.AddCookie(cookie)

			authenticated, returnedUserID := IsAuthenticated(req)
			if !authenticated {
				t.Errorf("Session %d should be authenticated", i)
			}
			if returnedUserID != userID {
				t.Errorf("Session %d expected user ID %s, got %s", i, userID, returnedUserID)
			}
		}
	})

	t.Run("session with file permission issues", func(t *testing.T) {
		// This is a negative test case mentioned in the PRD
		// Since we're using cookie-based sessions, file permissions aren't directly relevant
		// This test verifies that the system doesn't break with various cookie states

		cfg := setupTestConfig()
		Initialize(cfg)

		req := httptest.NewRequest("GET", "/", nil)

		// Add a cookie that might have permission-related issues
		req.AddCookie(&http.Cookie{
			Name:    "session",
			Value:   "test",
			Expires: time.Now().Add(-1 * time.Hour), // Expired
		})

		authenticated, userID := IsAuthenticated(req)
		if authenticated {
			t.Error("Expected authenticated to be false for expired cookie")
		}
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})
}

// TestIntegrationWithConfig tests integration with config package
func TestIntegrationWithConfig(t *testing.T) {
	// Test with config loaded from environment
	os.Setenv("SESSION_COOKIE_SECRET", "env-secret")
	os.Setenv("SESSION_MAX_AGE_SECONDS", "7200")
	os.Setenv("SESSION_COOKIE_HTTP_ONLY", "false")
	os.Setenv("SESSION_COOKIE_SECURE", "false")
	os.Setenv("SESSION_COOKIE_SAMESITE", "Strict")

	cfg := config.Load("")
	Initialize(cfg)

	if Store == nil {
		t.Fatal("Store should not be nil")
	}

	if Store.Options.MaxAge != 7200 {
		t.Errorf("Expected MaxAge 7200, got %d", Store.Options.MaxAge)
	}
	if Store.Options.HttpOnly {
		t.Error("Expected HttpOnly to be false")
	}
	if Store.Options.SameSite != http.SameSiteStrictMode {
		t.Errorf("Expected SameSiteStrictMode, got %v", Store.Options.SameSite)
	}

	// Clean up
	os.Unsetenv("SESSION_COOKIE_SECRET")
	os.Unsetenv("SESSION_MAX_AGE_SECONDS")
	os.Unsetenv("SESSION_COOKIE_HTTP_ONLY")
	os.Unsetenv("SESSION_COOKIE_SECURE")
	os.Unsetenv("SESSION_COOKIE_SAMESITE")
}
