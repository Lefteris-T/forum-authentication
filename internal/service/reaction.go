package service

import (
	"errors"

	"forum/internal/model"
)

// ErrInvalidReaction protects repositories from values outside like/dislike.
var ErrInvalidReaction = errors.New("invalid reaction")

type ReactionSetter interface {
	SetPostReaction(
		userID int64,
		postID int64,
		value model.Reaction,
	) error

	SetCommentReaction(
		userID int64,
		commentID int64,
		value model.Reaction,
	) error
}

// ReactionService applies authentication and value checks to reactions.
type ReactionService struct {
	reactions ReactionSetter
}

func NewReactionService(
	reactions ReactionSetter,
) *ReactionService {
	return &ReactionService{
		reactions: reactions,
	}
}

// SetPostReaction validates the actor and reaction before persistence.
func (s *ReactionService) SetPostReaction(
	userID int64,
	postID int64,
	value model.Reaction,
) error {
	if userID <= 0 {
		return ErrAuthenticationRequired
	}

	if !validReaction(value) {
		return ErrInvalidReaction
	}

	return s.reactions.SetPostReaction(
		userID,
		postID,
		value,
	)
}

// SetCommentReaction validates the actor and reaction before persistence.
func (s *ReactionService) SetCommentReaction(
	userID int64,
	commentID int64,
	value model.Reaction,
) error {
	if userID <= 0 {
		return ErrAuthenticationRequired
	}

	if !validReaction(value) {
		return ErrInvalidReaction
	}

	return s.reactions.SetCommentReaction(
		userID,
		commentID,
		value,
	)
}

func validReaction(value model.Reaction) bool {
	return value == model.ReactionLike ||
		value == model.ReactionDislike
}
