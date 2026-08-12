package view

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRendererRendersTemplateWithStatus(t *testing.T) {
	dir := t.TempDir()

	templatePath := filepath.Join(dir, "test.html")

	err := os.WriteFile(
		templatePath,
		[]byte(`
			<!doctype html>
			<html>
			<body>
				<h1>{{.Title}}</h1>
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	renderer, err := NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	recorder := httptest.NewRecorder()

	err = renderer.Render(
		recorder,
		http.StatusCreated,
		"test.html",
		map[string]string{
			"Title": "Forum",
		},
	)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusCreated,
		)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, "<h1>Forum</h1>") {
		t.Fatalf("body does not contain rendered title: %s", body)
	}
}
func TestRendererEscapesUserHTML(t *testing.T) {
	dir := t.TempDir()

	templatePath := filepath.Join(dir, "test.html")

	err := os.WriteFile(
		templatePath,
		[]byte(`
			<!doctype html>
			<html>
			<body>
				<p>{{.Username}}</p>
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	renderer, err := NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	recorder := httptest.NewRecorder()

	err = renderer.Render(
		recorder,
		http.StatusOK,
		"test.html",
		map[string]string{
			"Username": `<script>alert("xss")</script>`,
		},
	)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	body := recorder.Body.String()

	if strings.Contains(body, `<script>`) {
		t.Fatalf("user HTML was not escaped: %s", body)
	}

	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("escaped HTML not found: %s", body)
	}
}
func TestRendererReturnsErrorForMissingTemplate(t *testing.T) {
	dir := t.TempDir()

	templatePath := filepath.Join(dir, "test.html")

	err := os.WriteFile(
		templatePath,
		[]byte(`
			<!doctype html>
			<html>
			<body>
				<p>Forum</p>
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	renderer, err := NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	recorder := httptest.NewRecorder()

	err = renderer.Render(
		recorder,
		http.StatusOK,
		"missing.html",
		nil,
	)

	if err == nil {
		t.Fatal("Render() error = nil, want error")
	}

	if recorder.Body.Len() != 0 {
		t.Fatalf(
			"response body was written for missing template: %q",
			recorder.Body.String(),
		)
	}
}
