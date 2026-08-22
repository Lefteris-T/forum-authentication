package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forum/internal/model"
	"forum/internal/repository"
	"forum/internal/web/middleware"
)

type fakePostReactionService struct {
	called bool

	userID int64
	postID int64
	value  model.Reaction

	err error
}

func (f *fakePostReactionService) SetPostReaction(
	userID int64,
	postID int64,
	value model.Reaction,
) error {
	f.called = true
	f.userID = userID
	f.postID = postID
	f.value = value

	return f.err
}

func TestPostReactionHandlerValidPOST(t *testing.T) {
	service := &fakePostReactionService{}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/42/react",
		strings.NewReader("value=1"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	ctx := middleware.ContextWithUser(
		req.Context(),
		model.User{
			ID: 7,
		},
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusSeeOther,
		)
	}

	if !service.called {
		t.Fatal("SetPostReaction() was not called")
	}

	if service.userID != 7 {
		t.Fatalf(
			"userID = %d, want 7",
			service.userID,
		)
	}

	if service.postID != 42 {
		t.Fatalf(
			"postID = %d, want 42",
			service.postID,
		)
	}

	if service.value != model.ReactionLike {
		t.Fatalf(
			"value = %d, want %d",
			service.value,
			model.ReactionLike,
		)
	}

	if location := rec.Header().Get("Location"); location != "/posts/42" {
		t.Fatalf(
			"Location = %q, want %q",
			location,
			"/posts/42",
		)
	}
}
func TestPostReactionHandlerRejectsGuest(t *testing.T) {
	service := &fakePostReactionService{}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/42/react",
		strings.NewReader("value=1"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if service.called {
		t.Fatal("SetPostReaction() was called for guest")
	}
}

func TestPostReactionHandlerRejectsBadPostID(t *testing.T) {
	service := &fakePostReactionService{}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/abc/react",
		strings.NewReader("value=1"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	ctx := middleware.ContextWithUser(
		req.Context(),
		model.User{ID: 7},
	)
	req = req.WithContext(ctx)

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

func TestPostReactionHandlerRejectsBadValue(t *testing.T) {
	service := &fakePostReactionService{}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/42/react",
		strings.NewReader("value=5"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	ctx := middleware.ContextWithUser(
		req.Context(),
		model.User{ID: 7},
	)
	req = req.WithContext(ctx)

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

func TestPostReactionHandlerReturnsNotFound(t *testing.T) {
	service := &fakePostReactionService{
		err: repository.ErrPostNotFound,
	}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/999/react",
		strings.NewReader("value=1"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	ctx := middleware.ContextWithUser(
		req.Context(),
		model.User{ID: 7},
	)
	req = req.WithContext(ctx)

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

func TestPostReactionHandlerRejectsGET(t *testing.T) {
	service := &fakePostReactionService{}

	h := NewPostReactionHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/42/react",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if rec.Header().Get("Allow") != "POST" {
		t.Fatalf(
			"Allow = %q, want POST",
			rec.Header().Get("Allow"),
		)
	}
}
