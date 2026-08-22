package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forum/internal/model"
	"forum/internal/repository"
	"forum/internal/validation"
	"forum/internal/web/middleware"
)

type fakeCommentService struct {
	called   bool
	authorID int64
	postID   int64
	input    validation.CommentInput

	commentID int64
	err       error
}

func (f *fakeCommentService) Create(
	authorID int64,
	postID int64,
	input validation.CommentInput,
) (int64, error) {
	f.called = true
	f.authorID = authorID
	f.postID = postID
	f.input = input

	return f.commentID, f.err
}

func TestCommentSubmissionHandlerValidPOST(t *testing.T) {
	service := &fakeCommentService{
		commentID: 100,
	}

	h := NewCommentSubmissionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/42/comments",
		strings.NewReader("body=Hello+comment"),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	ctx := middleware.ContextWithUser(
		req.Context(),
		model.User{
			ID:       7,
			Username: "lefteris",
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
		t.Fatal("CommentService.Create() was not called")
	}

	if service.authorID != 7 {
		t.Fatalf(
			"authorID = %d, want 7",
			service.authorID,
		)
	}

	if service.postID != 42 {
		t.Fatalf(
			"postID = %d, want 42",
			service.postID,
		)
	}

	if service.input.Body != "Hello comment" {
		t.Fatalf(
			"body = %q, want %q",
			service.input.Body,
			"Hello comment",
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
func TestCommentSubmissionHandlerRejectsGuest(t *testing.T) {
	service := &fakeCommentService{}

	h := NewCommentSubmissionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/42/comments",
		strings.NewReader("body=Hello"),
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
		t.Fatal("CommentService.Create() was called for guest")
	}
}

func TestCommentSubmissionHandlerRejectsBadPostID(t *testing.T) {
	service := &fakeCommentService{}

	h := NewCommentSubmissionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/not-a-number/comments",
		strings.NewReader("body=Hello"),
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusBadRequest,
		)
	}

	if service.called {
		t.Fatal("CommentService.Create() was called for bad post ID")
	}
}

func TestCommentSubmissionHandlerReturnsNotFoundForMissingPost(t *testing.T) {
	service := &fakeCommentService{
		err: repository.ErrPostNotFound,
	}

	h := NewCommentSubmissionHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/posts/999/comments",
		strings.NewReader("body=Hello"),
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

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusNotFound,
		)
	}
}

func TestCommentSubmissionHandlerRejectsGET(t *testing.T) {
	service := &fakeCommentService{}

	h := NewCommentSubmissionHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/42/comments",
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
