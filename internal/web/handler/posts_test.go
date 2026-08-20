package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forum/internal/model"
	"forum/internal/repository"
	"forum/internal/web/middleware"
	"forum/internal/web/view"
)

type fakePostReader struct {
	posts  []repository.PostListItem
	detail repository.PostDetail
	err    error
}

func (f *fakePostReader) List() ([]repository.PostListItem, error) {
	return f.posts, f.err
}

func (f *fakePostReader) Detail(
	id int64,
) (repository.PostDetail, error) {
	return f.detail, f.err
}

func TestHomeHandlerEmptyList(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "home.html"),
		[]byte(`
			<!doctype html>
			<html>
			<body>
				{{if .Posts}}
					{{range .Posts}}
						<h2>{{.Title}}</h2>
					{{end}}
				{{else}}
					<p>No posts yet</p>
				{{end}}
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{}

	h := NewHomeHandler(
		posts,
		renderer,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"No posts yet",
	) {
		t.Fatal("empty state was not rendered")
	}
}
func TestHomeHandlerRendersPostsAndEscapesUserContent(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "home.html"),
		[]byte(`
			<!doctype html>
			<html>
			<body>
				{{range .Posts}}
					<article>
						<h2>{{.Title}}</h2>
						<p>{{.Body}}</p>
						<span>{{.Author.Username}}</span>
					</article>
				{{end}}
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{
		posts: []repository.PostListItem{
			{
				ID:    1,
				Title: `<script>alert("xss")</script>`,
				Body:  `<img src=x onerror=alert(1)>`,
				Author: model.User{
					ID:       42,
					Username: "lefteris",
				},
			},
		},
	}

	h := NewHomeHandler(
		posts,
		renderer,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "lefteris") {
		t.Fatal("post author was not rendered")
	}

	if strings.Contains(body, `<script>`) {
		t.Fatal("user script tag was not escaped")
	}

	if strings.Contains(body, `<img src=x`) {
		t.Fatal("user HTML was not escaped")
	}

	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("escaped title was not rendered")
	}
}
func TestHomeHandlerNavigationReflectsAuthentication(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "home.html"),
		[]byte(`
			<!doctype html>
			<html>
			<body>
				<nav>
					{{if .CurrentUser}}
						<span>{{.CurrentUser.Username}}</span>
						<a href="/posts/new">New Post</a>
						<form method="post" action="/logout">
							<button type="submit">Logout</button>
						</form>
					{{else}}
						<a href="/login">Login</a>
						<a href="/register">Register</a>
					{{end}}
				</nav>
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{}

	h := NewHomeHandler(
		posts,
		renderer,
	)

	t.Run("guest", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		body := rec.Body.String()

		if !strings.Contains(body, "Login") {
			t.Fatal("guest navigation does not contain Login")
		}

		if !strings.Contains(body, "Register") {
			t.Fatal("guest navigation does not contain Register")
		}

		if strings.Contains(body, "New Post") {
			t.Fatal("guest navigation contains New Post")
		}
	})

	t.Run("authenticated user", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		)

		ctx := middleware.ContextWithUser(
			req.Context(),
			model.User{
				ID:       42,
				Username: "lefteris",
			},
		)

		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		body := rec.Body.String()

		if !strings.Contains(body, "lefteris") {
			t.Fatal("authenticated username was not rendered")
		}

		if !strings.Contains(body, "New Post") {
			t.Fatal("authenticated navigation does not contain New Post")
		}

		if !strings.Contains(body, "Logout") {
			t.Fatal("authenticated navigation does not contain Logout")
		}

		if strings.Contains(body, ">Login<") {
			t.Fatal("authenticated navigation contains Login")
		}
	})
}
func TestPostDetailHandlerReturnsPost(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "post.html"),
		[]byte(`
			<!doctype html>
			<html>
			<body>
				<h1>{{.Post.Title}}</h1>
				<p>{{.Post.Body}}</p>
				<span>{{.Post.Author.Username}}</span>
			</body>
			</html>
		`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{
		detail: repository.PostDetail{
			ID:    42,
			Title: "Post title",
			Body:  "Post body",
			Author: model.User{
				ID:       1,
				Username: "lefteris",
			},
		},
	}

	h := NewPostDetailHandler(
		posts,
		renderer,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/42",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Post title") {
		t.Fatal("post title was not rendered")
	}

	if !strings.Contains(body, "lefteris") {
		t.Fatal("post author was not rendered")
	}
}
func TestPostDetailHandlerRejectsMalformedID(t *testing.T) {
	posts := &fakePostReader{}

	h := NewPostDetailHandler(
		posts,
		nil,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/not-a-number",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}
}

func TestPostDetailHandlerReturnsNotFound(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "post.html"),
		[]byte(`<html><body>{{.Post.Title}}</body></html>`),
		0o644,
	)
	if err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{
		err: repository.ErrPostNotFound,
	}

	h := NewPostDetailHandler(
		posts,
		renderer,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/999",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusNotFound,
		)
	}
}
func TestPublicPagesContainNoJavaScript(t *testing.T) {
	dir := t.TempDir()

	templates := map[string]string{
		"home.html": `
			<!doctype html>
			<html>
			<body>
				<h1>Forum</h1>
			</body>
			</html>
		`,
		"post.html": `
			<!doctype html>
			<html>
			<body>
				<h1>{{.Post.Title}}</h1>
				<p>{{.Post.Body}}</p>
			</body>
			</html>
		`,
	}

	for name, content := range templates {
		err := os.WriteFile(
			filepath.Join(dir, name),
			[]byte(content),
			0o644,
		)
		if err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	renderer, err := view.NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer(): %v", err)
	}

	posts := &fakePostReader{
		detail: repository.PostDetail{
			ID:    1,
			Title: "Post",
			Body:  "Body",
		},
	}

	tests := []struct {
		name    string
		handler http.Handler
		path    string
	}{
		{
			name:    "home",
			handler: NewHomeHandler(posts, renderer),
			path:    "/",
		},
		{
			name:    "post detail",
			handler: NewPostDetailHandler(posts, renderer),
			path:    "/posts/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			rec := httptest.NewRecorder()

			tt.handler.ServeHTTP(rec, req)

			body := strings.ToLower(
				rec.Body.String(),
			)

			forbidden := []string{
				"<script",
				"javascript:",
				"onclick=",
				"onerror=",
				"onload=",
			}

			for _, value := range forbidden {
				if strings.Contains(body, value) {
					t.Fatalf(
						"response contains forbidden JavaScript: %q",
						value,
					)
				}
			}
		})
	}
}
