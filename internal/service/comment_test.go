package service

import (
	"errors"
	"testing"

	"forum/internal/repository"
	"forum/internal/validation"
)

type fakeCommentCreator struct {
	called   bool
	postID   int64
	authorID int64
	body     string

	commentID int64
	err       error
}

func (f *fakeCommentCreator) Create(
	postID int64,
	authorID int64,
	body string,
) (int64, error) {
	f.called = true
	f.postID = postID
	f.authorID = authorID
	f.body = body

	return f.commentID, f.err
}

func TestCommentServiceCreatesValidatedComment(t *testing.T) {
	repo := &fakeCommentCreator{
		commentID: 100,
	}

	service := NewCommentService(repo)

	commentID, err := service.Create(
		42,
		10,
		validation.CommentInput{
			Body: "  Hello comment  ",
		},
	)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	if commentID != 100 {
		t.Fatalf(
			"commentID = %d, want 100",
			commentID,
		)
	}

	if !repo.called {
		t.Fatal("repository Create() was not called")
	}

	if repo.postID != 10 {
		t.Fatalf(
			"postID = %d, want 10",
			repo.postID,
		)
	}

	if repo.authorID != 42 {
		t.Fatalf(
			"authorID = %d, want 42",
			repo.authorID,
		)
	}

	if repo.body != "Hello comment" {
		t.Fatalf(
			"body = %q, want %q",
			repo.body,
			"Hello comment",
		)
	}
}

func TestCommentServiceRejectsGuest(t *testing.T) {
	repo := &fakeCommentCreator{}

	service := NewCommentService(repo)

	_, err := service.Create(
		0,
		10,
		validation.CommentInput{
			Body: "Hello",
		},
	)

	if !errors.Is(err, ErrAuthenticationRequired) {
		t.Fatalf(
			"Create() error = %v, want %v",
			err,
			ErrAuthenticationRequired,
		)
	}

	if repo.called {
		t.Fatal("repository Create() was called for guest")
	}
}

func TestCommentServiceInvalidInputStopsEarly(t *testing.T) {
	repo := &fakeCommentCreator{}

	service := NewCommentService(repo)

	_, err := service.Create(
		42,
		10,
		validation.CommentInput{
			Body: "   ",
		},
	)

	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}

	if repo.called {
		t.Fatal("repository Create() was called for invalid input")
	}
}

func TestCommentServicePreservesUnknownPostError(t *testing.T) {
	repo := &fakeCommentCreator{
		err: repository.ErrPostNotFound,
	}

	service := NewCommentService(repo)

	_, err := service.Create(
		42,
		999,
		validation.CommentInput{
			Body: "Hello",
		},
	)

	if !errors.Is(err, repository.ErrPostNotFound) {
		t.Fatalf(
			"Create() error = %v, want %v",
			err,
			repository.ErrPostNotFound,
		)
	}
}
