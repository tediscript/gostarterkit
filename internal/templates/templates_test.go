package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tediscript/gostarterkit/internal/handlers"
)

// TestNewCache tests creating a new template cache
func TestNewCache(t *testing.T) {
	tests := []struct {
		name  string
		isDev bool
	}{
		{
			name:  "Production mode cache",
			isDev: false,
		},
		{
			name:  "Development mode cache",
			isDev: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(tt.isDev)
			if cache == nil {
				t.Fatal("NewCache returned nil")
			}
			if cache.isDev != tt.isDev {
				t.Errorf("Expected isDev=%v, got %v", tt.isDev, cache.isDev)
			}
			if cache.templates == nil {
				t.Error("templates map should be initialized")
			}
		})
	}
}

// TestParseTemplate tests parsing templates from strings
func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmplName string
		content  string
		wantErr  bool
	}{
		{
			name:     "Valid simple template",
			tmplName: "test",
			content:  "Hello {{.Name}}",
			wantErr:  false,
		},
		{
			name:     "Template with HTML",
			tmplName: "test",
			content:  "<h1>{{.Title}}</h1><p>{{.Content}}</p>",
			wantErr:  false,
		},
		{
			name:     "Template with loops",
			tmplName: "test",
			content:  "{{range .Items}}{{.}} {{end}}",
			wantErr:  false,
		},
		{
			name:     "Template with conditionals",
			tmplName: "test",
			content:  "{{if .Show}}Visible{{end}}",
			wantErr:  false,
		},
		{
			name:     "Invalid template syntax",
			tmplName: "test",
			content:  "{{unclosed brace",
			wantErr:  true,
		},
		{
			name:     "Empty template",
			tmplName: "test",
			content:  "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := ParseTemplate(tt.tmplName, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tmpl == nil {
				t.Error("ParseTemplate() returned nil template")
			}
		})
	}
}

// TestDataBinding tests data binding to templates
func TestDataBinding(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "Multiple fields",
			template: "{{.FirstName}} {{.LastName}} is {{.Age}} years old",
			data: struct {
				FirstName string
				LastName  string
				Age       int
			}{FirstName: "John", LastName: "Doe", Age: 30},
			want:    "John Doe is 30 years old",
			wantErr: false,
		},
		{
			name:     "Map data",
			template: "Hello {{.name}}",
			data:     map[string]string{"name": "Alice"},
			want:     "Hello Alice",
			wantErr:  false,
		},
		{
			name:     "Nil data",
			template: "Hello",
			data:     nil,
			want:     "Hello",
			wantErr:  false,
		},
		{
			name:     "Empty string data",
			template: "Value: [{{.Value}}]",
			data:     struct{ Value string }{Value: ""},
			want:     "Value: []",
			wantErr:  false,
		},
		{
			name:     "Zero values",
			template: "Int:{{.I}} Float:{{.F}} String:{{.S}} Bool:{{.B}}",
			data: struct {
				I int
				F float64
				S string
				B bool
			}{I: 0, F: 0.0, S: "", B: false},
			want:    "Int:0 Float:0 String: Bool:false",
			wantErr: false,
		},
		{
			name:     "Missing field",
			template: "Hello {{.Missing}}",
			data:     struct{ Name string }{Name: "Test"},
			want:     "Hello ",
			wantErr:  true, // html/template returns error for missing struct fields
		},
		{
			name:     "Unicode characters",
			template: "Hello {{.Name}} 你好 こんにちは",
			data:     struct{ Name string }{Name: "世界"},
			want:     "Hello 世界 你好 こんにちは",
			wantErr:  false,
		},
		{
			name:     "Special characters",
			template: "Value: {{.Value}}",
			data:     struct{ Value string }{Value: "<>&\"'"},
			want:     "Value: &lt;&gt;&amp;&#34;&#39;",
			wantErr:  false,
		},
		{
			name:     "Emoji",
			template: "Mood: {{.Emoji}}",
			data:     struct{ Emoji string }{Emoji: "😀🎉"},
			want:     "Mood: 😀🎉",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := ParseTemplate("test", tt.template)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := buf.String()
			if got != tt.want {
				t.Errorf("Execute() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderTemplate tests rendering templates through cache
func TestRenderTemplate(t *testing.T) {
	// Create a temporary directory for templates
	tmpDir := t.TempDir()

	// Create a test template file
	tmplPath := filepath.Join(tmpDir, "test.html")
	tmplContent := `<h1>{{.Title}}</h1><p>{{.Message}}</p>`
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	// Create cache and load templates
	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	tests := []struct {
		name     string
		tmplName string
		data     handlers.TemplateData
		wantErr  bool
	}{
		{
			name:     "Valid template render",
			tmplName: "test.html",
			data: handlers.TemplateData{
				Title:   "Test Title",
				Message: "Test Message",
			},
			wantErr: false,
		},
		{
			name:     "Template with empty data",
			tmplName: "test.html",
			data:     handlers.TemplateData{},
			wantErr:  false,
		},
		{
			name:     "Template with nil data field",
			tmplName: "test.html",
			data: handlers.TemplateData{
				Title:   "Test",
				Message: "Message",
				Data:    nil,
			},
			wantErr: false,
		},
		{
			name:     "Non-existent template",
			tmplName: "missing.html",
			data:     handlers.TemplateData{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := cache.RenderTemplate(&buf, tt.tmplName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				if !strings.Contains(output, tt.data.Title) {
					t.Errorf("Output does not contain title: %s", output)
				}
				if !strings.Contains(output, tt.data.Message) {
					t.Errorf("Output does not contain message: %s", output)
				}
			}
		})
	}
}

// TestLoadTemplates tests loading templates from directory
func TestLoadTemplates(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		wantCount int
	}{
		{
			name: "Load single template",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				tmplPath := filepath.Join(tmpDir, "test.html")
				os.WriteFile(tmplPath, []byte("Test {{.Title}}"), 0644)
				return tmpDir
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "Load multiple templates",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "one.html"), []byte("One {{.Title}}"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "two.html"), []byte("Two {{.Title}}"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "three.html"), []byte("Three {{.Title}}"), 0644)
				return tmpDir
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "Ignore non-html files",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "test.html"), []byte("HTML"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Text"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte("{}"), 0644)
				return tmpDir
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "Empty directory",
			setupFunc: func() string {
				return t.TempDir()
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "Non-existent directory",
			setupFunc: func() string {
				return "/non/existent/directory"
			},
			wantErr:   true,
			wantCount: 0,
		},
		{
			name: "Invalid template syntax",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				tmplPath := filepath.Join(tmpDir, "bad.html")
				os.WriteFile(tmplPath, []byte("{{unclosed"), 0644)
				return tmpDir
			},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tt.setupFunc()
			cache := NewCache(false)
			err := cache.LoadTemplates(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				count := len(cache.templates)
				if count != tt.wantCount {
					t.Errorf("Loaded %d templates, want %d", count, tt.wantCount)
				}
			}
		})
	}
}

// TestGetTemplate tests retrieving templates by name
func TestGetTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "test.html")
	os.WriteFile(tmplPath, []byte("Test {{.Title}}"), 0644)

	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	tests := []struct {
		name     string
		tmplName string
		wantErr  bool
	}{
		{
			name:     "Get existing template",
			tmplName: "test.html",
			wantErr:  false,
		},
		{
			name:     "Get non-existent template",
			tmplName: "missing.html",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := cache.GetTemplate(tt.tmplName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tmpl == nil {
				t.Error("GetTemplate() returned nil template")
			}
		})
	}
}

// TestCachingBehavior tests template caching in production mode
func TestCachingBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "test.html")
	os.WriteFile(tmplPath, []byte("Original {{.Title}}"), 0644)

	// Create production cache (no hot-reload)
	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Render original template
	var buf1 bytes.Buffer
	data := handlers.TemplateData{Title: "Test"}
	if err := cache.RenderTemplate(&buf1, "test.html", data); err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	originalOutput := buf1.String()

	// Modify template file
	time.Sleep(time.Millisecond * 10) // Ensure different modification time
	os.WriteFile(tmplPath, []byte("Modified {{.Title}}"), 0644)

	// Render again - should use cached template
	var buf2 bytes.Buffer
	if err := cache.RenderTemplate(&buf2, "test.html", data); err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	cachedOutput := buf2.String()

	// Output should be the same (from cache)
	if cachedOutput != originalOutput {
		t.Errorf("Production cache should not reload templates: got %q, want %q", cachedOutput, originalOutput)
	}
}

// TestHotReloadBehavior tests template hot-reload in development mode
func TestHotReloadBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "test.html")
	os.WriteFile(tmplPath, []byte("Original {{.Title}}"), 0644)

	// Create development cache (with hot-reload)
	cache := NewCache(true)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Render original template
	var buf1 bytes.Buffer
	data := handlers.TemplateData{Title: "Test"}
	if err := cache.RenderTemplate(&buf1, "test.html", data); err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	originalOutput := buf1.String()

	// Modify template file
	time.Sleep(time.Millisecond * 10) // Ensure different modification time
	os.WriteFile(tmplPath, []byte("Modified {{.Title}}"), 0644)

	// Check for reload
	if err := cache.CheckForReload(tmpDir); err != nil {
		t.Fatalf("Failed to check for reload: %v", err)
	}

	// Render again - should use reloaded template
	var buf2 bytes.Buffer
	if err := cache.RenderTemplate(&buf2, "test.html", data); err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	reloadedOutput := buf2.String()

	// Output should be different (reloaded)
	if reloadedOutput == originalOutput {
		t.Error("Development cache should reload templates")
	}

	if !strings.Contains(reloadedOutput, "Modified") {
		t.Errorf("Reloaded template does not contain modified content: %s", reloadedOutput)
	}
}

// TestLargeTemplate tests handling of very large templates
func TestLargeTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large template (1MB)
	var largeContent strings.Builder
	largeContent.WriteString("<html><body>")
	for i := 0; i < 10000; i++ {
		largeContent.WriteString("<p>Line {{.Title}} " + strings.Repeat("x", 100) + "</p>")
	}
	largeContent.WriteString("</body></html>")

	tmplPath := filepath.Join(tmpDir, "large.html")
	if err := os.WriteFile(tmplPath, []byte(largeContent.String()), 0644); err != nil {
		t.Fatalf("Failed to create large template: %v", err)
	}

	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load large template: %v", err)
	}

	var buf bytes.Buffer
	data := handlers.TemplateData{Title: "Test"}
	if err := cache.RenderTemplate(&buf, "large.html", data); err != nil {
		t.Fatalf("Failed to render large template: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Large template rendered empty output")
	}
}

// TestConcurrentAccess tests concurrent access to template cache
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "test.html")
	os.WriteFile(tmplPath, []byte("{{.Title}} - {{.Message}}"), 0644)

	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Concurrent renders
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			var buf bytes.Buffer
			data := handlers.TemplateData{
				Title:   "Concurrent",
				Message: "Test",
			}
			err := cache.RenderTemplate(&buf, "test.html", data)
			if err != nil {
				t.Errorf("Concurrent render failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestParseTemplateFiles tests parsing templates from multiple files
func TestParseTemplateFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base template
	basePath := filepath.Join(tmpDir, "base.html")
	baseContent := `<html><body>{{block "content" .}}{{end}}</body></html>`
	os.WriteFile(basePath, []byte(baseContent), 0644)

	// Create content template
	contentPath := filepath.Join(tmpDir, "content.html")
	contentContent := `{{define "content"}}<h1>{{.Title}}</h1>{{end}}`
	os.WriteFile(contentPath, []byte(contentContent), 0644)

	tmpl, err := ParseTemplateFiles("combined", basePath, contentPath)
	if err != nil {
		t.Fatalf("Failed to parse template files: %v", err)
	}

	if tmpl == nil {
		t.Fatal("ParseTemplateFiles returned nil")
	}

	// Render the combined template
	var buf bytes.Buffer
	data := handlers.TemplateData{Title: "Test"}
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute combined template: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<html>") {
		t.Error("Combined template missing base HTML")
	}
	if !strings.Contains(output, "Test") {
		t.Error("Combined template missing content")
	}
}

// TestNestedTemplateIncludes tests templates with nested includes
func TestNestedTemplateIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested templates
	os.WriteFile(filepath.Join(tmpDir, "parent.html"), []byte("Parent: {{template \"child\" .}}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "child.html"), []byte("{{define \"child\"}}Child: {{.Title}}{{end}}"), 0644)

	cache := NewCache(false)
	if err := cache.LoadTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to load nested templates: %v", err)
	}

	// Note: Nested template inheritance requires template.ParseFiles to be called together
	// This test verifies that templates can be loaded individually
	if len(cache.templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(cache.templates))
	}
}
