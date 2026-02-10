package handlers

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/tediscript/gostarterkit/internal/auth"
)

// LoginPage renders the login form
func LoginPage(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If already authenticated, redirect to home
		if authenticated, _ := auth.IsAuthenticated(r); authenticated {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get any error message from query params
		errorMsg := r.URL.Query().Get("error")

		data := struct {
			Error string
		}{
			Error: errorMsg,
		}

		// Execute template
		if err := tpl.ExecuteTemplate(w, "login.html", data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// LoginHandler processes login requests
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate credentials
	// In a real application, you would check against a database
	// For this example, we'll use a simple hardcoded check
	if !validateCredentials(username, password) {
		// Redirect back to login with error
		http.Redirect(w, r, "/login?error=invalid+credentials", http.StatusSeeOther)
		return
	}

	// Create session
	if err := auth.SetUserSession(w, r, username); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect to the page the user was trying to access, or to home
	redirectURL := r.FormValue("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Validate redirect URL to prevent open redirect attacks
	if parsedURL, err := url.Parse(redirectURL); err != nil || parsedURL.IsAbs() {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// LogoutHandler clears the user's session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := auth.ClearSession(w, r); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ProtectedPage renders a page that requires authentication
func ProtectedPage(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from session
		userID, ok := auth.GetUserID(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		data := struct {
			UserID string
		}{
			UserID: userID,
		}

		// Execute template
		if err := tpl.ExecuteTemplate(w, "protected.html", data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

// validateCredentials checks if the username and password are valid
// In a real application, this would check against a database
// For this example, we accept any non-empty username and password
func validateCredentials(username, password string) bool {
	return username != "" && password != ""
}
