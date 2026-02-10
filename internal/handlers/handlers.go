package handlers

import (
	"net/http"
)

// TemplateCache interface for template rendering
type TemplateCache interface {
	RenderTemplate(wr interface{ Write([]byte) (int, error) }, name string, data TemplateData) error
	CheckForReload(templatesDir string) error
}

// TemplateData represents the data passed to templates
type TemplateData struct {
	Title   string
	Message string
	Data    interface{} // Additional data for custom templates
}

// Handlers holds handler dependencies
type Handlers struct {
	Templates    TemplateCache
	TemplatesDir string
}

// New creates a new Handlers instance
func New(templates TemplateCache, templatesDir string) *Handlers {
	return &Handlers{
		Templates:    templates,
		TemplatesDir: templatesDir,
	}
}

// Home handles the home page
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	// Check for template reload in development mode
	if err := h.Templates.CheckForReload(h.TemplatesDir); err != nil {
		http.Error(w, "Failed to reload templates", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := TemplateData{
		Title:   "Go Starter Kit",
		Message: "Welcome to the production-ready Go starter kit",
		Data:    nil,
	}

	// Render home page template
	// Note: For now, we're using base.html which includes the content block
	// In the future, we can implement template inheritance with home.html
	if err := h.Templates.RenderTemplate(w, "base.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// APIStatus handles GET /api/status - returns API status information
func (h *Handlers) APIStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"version": "1.0.0",
		"health":  "ok",
		"uptime":  "running",
	}
	JSONResponse(w, http.StatusOK, status)
}

// APIHello handles GET /api/hello - demonstrates JSONResponseWithMessage
func (h *Handlers) APIHello(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"greeting": "Hello, World!",
		"time":     r.URL.Query().Get("time"),
	}
	JSONResponseWithMessage(w, http.StatusOK, "Welcome to the Go Starter Kit API", data)
}

// APIError handles GET /api/error - demonstrates error responses
func (h *Handlers) APIError(w http.ResponseWriter, r *http.Request) {
	errorType := r.URL.Query().Get("type")
	switch errorType {
	case "notfound":
		ErrorResponseFunc(w, http.StatusNotFound, "Resource not found")
	case "badrequest":
		ErrorResponseWithDetails(w, http.StatusBadRequest, "Invalid request", "The 'id' parameter is required")
	case "validation":
		ValidationError(w, "email", "must be a valid email address")
	default:
		ErrorResponseFunc(w, http.StatusInternalServerError, "An internal error occurred")
	}
}

// APIData handles GET /api/data - demonstrates complex nested data structures
func (h *Handlers) APIData(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"users": []map[string]interface{}{
			{
				"id":    1,
				"name":  "Alice",
				"email": "alice@example.com",
				"roles": []string{"admin", "user"},
			},
			{
				"id":    2,
				"name":  "Bob",
				"email": "bob@example.com",
				"roles": []string{"user"},
			},
		},
		"pagination": map[string]int{
			"page":  1,
			"limit": 10,
			"total": 2,
		},
	}
	JSONResponse(w, http.StatusOK, data)
}
