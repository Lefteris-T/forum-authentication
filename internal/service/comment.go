package service

import "forum/internal/validation"

type CommentCreator interface {
	Create(
		postID int64,
		authorID int64,
		body string,
	) (int64, error)
}

type CommentService struct {
	comments CommentCreator
}

func NewCommentService(
	comments CommentCreator,
) *CommentService {
	return &CommentService{
		comments: comments,
	}
}

func (s *CommentService) Create(
	authorID int64,
	postID int64,
	input validation.CommentInput,
) (int64, error) {
	if authorID <= 0 {
		return 0, ErrAuthenticationRequired
	}

	validated, err := validation.ValidateComment(input)
	if err != nil {
		return 0, err
	}

	return s.comments.Create(
		postID,
		authorID,
		validated.Body,
	)
}
