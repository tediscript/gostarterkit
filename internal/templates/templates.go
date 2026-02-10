package templates

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tediscript/gostarterkit/internal/handlers"
)

// TemplateCache manages parsed HTML templates with caching support
type TemplateCache struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	filenames map[string]string    // Track template file paths for hot-reload
	modTimes  map[string]time.Time // Track modification times for hot-reload
	isDev     bool                 // Development mode flag for hot-reload
}

// NewCache creates a new template cache
func NewCache(isDev bool) *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*template.Template),
		filenames: make(map[string]string),
		modTimes:  make(map[string]time.Time),
		isDev:     isDev,
	}
}

// LoadTemplates loads all templates from the templates directory
func (tc *TemplateCache) LoadTemplates(templatesDir string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Clear existing templates
	tc.templates = make(map[string]*template.Template)
	tc.filenames = make(map[string]string)
	tc.modTimes = make(map[string]time.Time)

	// Walk through templates directory
	err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .html files
		if filepath.Ext(path) != ".html" {
			return nil
		}

		// Get file modification time for hot-reload
		fileInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat template file %s: %w", path, err)
		}

		// Load template content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		// Parse template
		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// If this is base.html, we need to handle it specially for template inheritance
		// For now, we'll store all templates by their base filename
		name := filepath.Base(path)
		tc.templates[name] = tmpl
		tc.filenames[name] = path
		tc.modTimes[name] = fileInfo.ModTime()

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	return nil
}

// CheckForReload checks if templates need to be reloaded in development mode
func (tc *TemplateCache) CheckForReload(templatesDir string) error {
	if !tc.isDev {
		return nil
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	needsReload := false

	// Check if any template files have been modified
	for name, path := range tc.filenames {
		fileInfo, err := os.Stat(path)
		if err != nil {
			// If file doesn't exist or can't be accessed, trigger reload
			needsReload = true
			break
		}

		// Check if modification time has changed
		if !fileInfo.ModTime().Equal(tc.modTimes[name]) {
			needsReload = true
			break
		}
	}

	if needsReload {
		// Release lock before calling LoadTemplates
		tc.mu.Unlock()
		err := tc.LoadTemplates(templatesDir)
		tc.mu.Lock()
		if err != nil {
			return fmt.Errorf("failed to reload templates: %w", err)
		}
	}

	return nil
}

// RenderTemplate renders a template with the given data
func (tc *TemplateCache) RenderTemplate(wr interface{ Write([]byte) (int, error) }, name string, data handlers.TemplateData) error {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tmpl, ok := tc.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	// Execute template with data
	err := tmpl.Execute(wr, data)
	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return nil
}

// GetTemplate returns a template by name
func (tc *TemplateCache) GetTemplate(name string) (*template.Template, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tmpl, ok := tc.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %s not found", name)
	}

	return tmpl, nil
}

// ParseTemplate parses a single template from a string
func ParseTemplate(name, content string) (*template.Template, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	return tmpl, nil
}

// ParseTemplateFiles parses templates from files for use in template inheritance
func ParseTemplateFiles(name string, files ...string) (*template.Template, error) {
	if len(files) == 0 {
		return nil, errors.New("no template files provided")
	}

	// Read the first file as the base template
	content, err := os.ReadFile(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", files[0], err)
	}

	// Create template from base file
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base template: %w", err)
	}

	// Parse additional files for template inheritance
	for _, file := range files[1:] {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read template file %s: %w", file, err)
		}

		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template file %s: %w", file, err)
		}
	}

	return tmpl, nil
}
