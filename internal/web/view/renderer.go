// Package view parses and safely executes server-rendered HTML templates.
package view

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

// Renderer holds templates parsed once during application startup.
type Renderer struct {
	templates *template.Template
}

// NewRenderer fails startup when required templates cannot be parsed.
func NewRenderer(dir string) (*Renderer, error) {
	pattern := filepath.Join(dir, "*.html")

	templates, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Renderer{
		templates: templates,
	}, nil
}

// Render writes a status before executing the named escaped HTML template.
func (r *Renderer) Render(
	w http.ResponseWriter,
	status int,
	name string,
	data any,
) error {
	tmpl := r.templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("template %q not found", name)
	}

	var buffer bytes.Buffer

	if err := tmpl.Execute(&buffer, data); err != nil {
		return fmt.Errorf("execute template %q: %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if _, err := buffer.WriteTo(w); err != nil {
		return fmt.Errorf("write template response: %w", err)
	}

	return nil
}
