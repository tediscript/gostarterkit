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
