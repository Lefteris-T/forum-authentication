package validation

import (
	"errors"
	"strings"
)

const MaxCommentBodyLength = 2000

var (
	ErrCommentBodyRequired = errors.New("comment body is required")
	ErrCommentBodyTooLong  = errors.New("comment body is too long")
)

type CommentInput struct {
	Body string
}

func ValidateComment(input CommentInput) (CommentInput, error) {
	input.Body = strings.TrimSpace(input.Body)

	if input.Body == "" {
		return CommentInput{}, ErrCommentBodyRequired
	}

	if len(input.Body) > MaxCommentBodyLength {
		return CommentInput{}, ErrCommentBodyTooLong
	}

	return input, nil
}
