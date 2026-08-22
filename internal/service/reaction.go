package service

import (
	"errors"

	"forum/internal/model"
)

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
