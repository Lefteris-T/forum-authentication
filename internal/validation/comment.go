package validation

import (
	"errors"
	"strings"
)

// MaxCommentBodyLength limits storage and rendered page size.
const MaxCommentBodyLength = 2000

var (
	ErrCommentBodyRequired = errors.New("comment body is required")
	ErrCommentBodyTooLong  = errors.New("comment body is too long")
)

// CommentInput contains an untrusted comment body.
type CommentInput struct {
	Body string
}

// ValidateComment trims surrounding whitespace and rejects empty or long text.
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
