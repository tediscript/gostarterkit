package auth

import (
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/tediscript/gostarterkit/internal/config"
)

const (
	// SessionUserIDKey is the key used to store the user ID in the session
	SessionUserIDKey = "user_id"
	// SessionAuthenticatedKey is the key used to track authentication status
	SessionAuthenticatedKey = "authenticated"
	// SessionCreatedAtKey is the key used to store session creation timestamp
	SessionCreatedAtKey = "created_at"
)

// Store is the global session store
var Store *sessions.CookieStore

// Initialize creates and configures the session store
func Initialize(cfg *config.Config) {
	// Use the session cookie secret from config
	// If not provided, generate a random one (for development only)
	secret := []byte(cfg.Session.CookieSecret)
	if len(secret) == 0 {
		if cfg.App.Env == "production" {
			panic("SESSION_COOKIE_SECRET is required in production")
		}
		// For development, use a temporary secret
		secret = []byte("development-secret-change-in-production")
	}

	// Create cookie store
	Store = sessions.NewCookieStore(secret)

	// Configure session options
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   cfg.Session.MaxAgeSeconds,
		HttpOnly: cfg.Session.CookieHTTPOnly,
		Secure:   cfg.Session.CookieSecure,
		SameSite: parseSameSite(cfg.Session.CookieSameSite),
	}
}

// parseSameSite converts string to http.SameSite
func parseSameSite(sameSite string) http.SameSite {
	switch sameSite {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	case "Lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}

// GetSession retrieves the current session
func GetSession(r *http.Request) (*sessions.Session, error) {
	return Store.Get(r, "session")
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(r *http.Request) (bool, string) {
	session, err := GetSession(r)
	if err != nil {
		return false, ""
	}

	authenticated := session.Values[SessionAuthenticatedKey]
	if authenticated != true {
		return false, ""
	}

	userID := session.Values[SessionUserIDKey]
	if userID == nil {
		return false, ""
	}

	return true, userID.(string)
}

// SetUserSession creates a session for the authenticated user
func SetUserSession(w http.ResponseWriter, r *http.Request, userID string) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	session.Values[SessionAuthenticatedKey] = true
	session.Values[SessionUserIDKey] = userID
	session.Values[SessionCreatedAtKey] = time.Now().Unix()

	return session.Save(r, w)
}

// ClearSession removes the user's session
func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	// Clear all session values
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1 // Expire immediately

	return session.Save(r, w)
}

// RequireAuth is middleware that ensures the user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated, _ := IsAuthenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserID retrieves the user ID from the session
func GetUserID(r *http.Request) (string, bool) {
	_, userID := IsAuthenticated(r)
	if userID == "" {
		return "", false
	}
	return userID, true
}
