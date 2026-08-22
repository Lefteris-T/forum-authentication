package service

import (
	"errors"
	"testing"

	"forum/internal/model"
	"forum/internal/repository"
)

type fakeReactionSetter struct {
	postCalled    bool
	commentCalled bool

	userID   int64
	targetID int64
	value    model.Reaction

	err error
}

func (f *fakeReactionSetter) SetPostReaction(
	userID int64,
	postID int64,
	value model.Reaction,
) error {
	f.postCalled = true
	f.userID = userID
	f.targetID = postID
	f.value = value

	return f.err
}

func (f *fakeReactionSetter) SetCommentReaction(
	userID int64,
	commentID int64,
	value model.Reaction,
) error {
	f.commentCalled = true
	f.userID = userID
	f.targetID = commentID
	f.value = value

	return f.err
}

func TestReactionServiceSetsPostReaction(t *testing.T) {
	repo := &fakeReactionSetter{}

	service := NewReactionService(repo)

	err := service.SetPostReaction(
		42,
		10,
		model.ReactionLike,
	)
	if err != nil {
		t.Fatalf("SetPostReaction() error = %v, want nil", err)
	}

	if !repo.postCalled {
		t.Fatal("repository SetPostReaction() was not called")
	}

	if repo.userID != 42 {
		t.Fatalf(
			"userID = %d, want 42",
			repo.userID,
		)
	}

	if repo.targetID != 10 {
		t.Fatalf(
			"postID = %d, want 10",
			repo.targetID,
		)
	}

	if repo.value != model.ReactionLike {
		t.Fatalf(
			"value = %d, want %d",
			repo.value,
			model.ReactionLike,
		)
	}
}

func TestReactionServiceSetsCommentReaction(t *testing.T) {
	repo := &fakeReactionSetter{}

	service := NewReactionService(repo)

	err := service.SetCommentReaction(
		42,
		20,
		model.ReactionDislike,
	)
	if err != nil {
		t.Fatalf(
			"SetCommentReaction() error = %v, want nil",
			err,
		)
	}

	if !repo.commentCalled {
		t.Fatal("repository SetCommentReaction() was not called")
	}

	if repo.userID != 42 {
		t.Fatalf(
			"userID = %d, want 42",
			repo.userID,
		)
	}

	if repo.targetID != 20 {
		t.Fatalf(
			"commentID = %d, want 20",
			repo.targetID,
		)
	}

	if repo.value != model.ReactionDislike {
		t.Fatalf(
			"value = %d, want %d",
			repo.value,
			model.ReactionDislike,
		)
	}
}

func TestReactionServiceRejectsGuest(t *testing.T) {
	repo := &fakeReactionSetter{}

	service := NewReactionService(repo)

	err := service.SetPostReaction(
		0,
		10,
		model.ReactionLike,
	)

	if !errors.Is(err, ErrAuthenticationRequired) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			ErrAuthenticationRequired,
		)
	}

	if repo.postCalled {
		t.Fatal("repository was called for guest")
	}
}

func TestReactionServiceRejectsInvalidValue(t *testing.T) {
	repo := &fakeReactionSetter{}

	service := NewReactionService(repo)

	err := service.SetPostReaction(
		42,
		10,
		model.Reaction(5),
	)

	if !errors.Is(err, ErrInvalidReaction) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			ErrInvalidReaction,
		)
	}

	if repo.postCalled {
		t.Fatal("repository was called for invalid reaction")
	}
}

func TestReactionServicePreservesMissingTargetErrors(t *testing.T) {
	t.Run("missing post", func(t *testing.T) {
		repo := &fakeReactionSetter{
			err: repository.ErrPostNotFound,
		}

		service := NewReactionService(repo)

		err := service.SetPostReaction(
			42,
			999,
			model.ReactionLike,
		)

		if !errors.Is(err, repository.ErrPostNotFound) {
			t.Fatalf(
				"error = %v, want %v",
				err,
				repository.ErrPostNotFound,
			)
		}
	})

	t.Run("missing comment", func(t *testing.T) {
		repo := &fakeReactionSetter{
			err: repository.ErrCommentNotFound,
		}

		service := NewReactionService(repo)

		err := service.SetCommentReaction(
			42,
			999,
			model.ReactionLike,
		)

		if !errors.Is(err, repository.ErrCommentNotFound) {
			t.Fatalf(
				"error = %v, want %v",
				err,
				repository.ErrCommentNotFound,
			)
		}
	})
}
