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

type fakeCommentReactionService struct {
	called bool

	userID    int64
	commentID int64
	value     model.Reaction

	err error
}

func (f *fakeCommentReactionService) SetCommentReaction(
	userID int64,
	commentID int64,
	value model.Reaction,
) error {
	f.called = true
	f.userID = userID
	f.commentID = commentID
	f.value = value

	return f.err
}

type fakeCommentPostFinder struct {
	postID int64
	err    error
}

func (f *fakeCommentPostFinder) PostIDForComment(
	commentID int64,
) (int64, error) {
	return f.postID, f.err
}

func TestCommentReactionHandlerValidPOST(t *testing.T) {
	service := &fakeCommentReactionService{}

	comments := &fakeCommentPostFinder{
		postID: 42,
	}

	h := NewCommentReactionHandler(
		service,
		comments,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/comments/15/react",
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
		t.Fatal("SetCommentReaction() was not called")
	}

	if service.userID != 7 {
		t.Fatalf(
			"userID = %d, want 7",
			service.userID,
		)
	}

	if service.commentID != 15 {
		t.Fatalf(
			"commentID = %d, want 15",
			service.commentID,
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
func TestCommentReactionHandlerRejectsGuest(t *testing.T) {
	service := &fakeCommentReactionService{}
	comments := &fakeCommentPostFinder{postID: 42}

	h := NewCommentReactionHandler(service, comments)

	req := httptest.NewRequest(
		http.MethodPost,
		"/comments/15/react",
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
		t.Fatal("SetCommentReaction() was called for guest")
	}
}

func TestCommentReactionHandlerRejectsBadCommentID(t *testing.T) {
	service := &fakeCommentReactionService{}
	comments := &fakeCommentPostFinder{postID: 42}

	h := NewCommentReactionHandler(service, comments)

	req := httptest.NewRequest(
		http.MethodPost,
		"/comments/abc/react",
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

func TestCommentReactionHandlerRejectsBadValue(t *testing.T) {
	service := &fakeCommentReactionService{}
	comments := &fakeCommentPostFinder{postID: 42}

	h := NewCommentReactionHandler(service, comments)

	req := httptest.NewRequest(
		http.MethodPost,
		"/comments/15/react",
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

func TestCommentReactionHandlerReturnsNotFound(t *testing.T) {
	service := &fakeCommentReactionService{
		err: repository.ErrCommentNotFound,
	}

	comments := &fakeCommentPostFinder{
		postID: 42,
	}

	h := NewCommentReactionHandler(service, comments)

	req := httptest.NewRequest(
		http.MethodPost,
		"/comments/999/react",
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

func TestCommentReactionHandlerRejectsGET(t *testing.T) {
	service := &fakeCommentReactionService{}
	comments := &fakeCommentPostFinder{}

	h := NewCommentReactionHandler(service, comments)

	req := httptest.NewRequest(
		http.MethodGet,
		"/comments/15/react",
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
