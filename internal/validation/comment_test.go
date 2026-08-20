package validation

import (
	"strings"
	"testing"
)

func TestValidateCommentAcceptsValidInput(t *testing.T) {
	input := CommentInput{
		Body: "  This is a comment.  ",
	}

	got, err := ValidateComment(input)
	if err != nil {
		t.Fatalf("ValidateComment() error = %v, want nil", err)
	}

	if got.Body != "This is a comment." {
		t.Fatalf(
			"Body = %q, want %q",
			got.Body,
			"This is a comment.",
		)
	}
}

func TestValidateCommentRejectsEmptyOrWhitespace(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"\n\t",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			_, err := ValidateComment(
				CommentInput{
					Body: body,
				},
			)

			if err == nil {
				t.Fatal("ValidateComment() error = nil, want error")
			}
		})
	}
}

func TestValidateCommentRejectsOversizedBody(t *testing.T) {
	input := CommentInput{
		Body: strings.Repeat(
			"a",
			MaxCommentBodyLength+1,
		),
	}

	_, err := ValidateComment(input)

	if err == nil {
		t.Fatal("ValidateComment() error = nil, want error")
	}
}
