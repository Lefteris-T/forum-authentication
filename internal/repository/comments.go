package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// CommentRepository stores comments and resolves their parent posts.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository binds comment operations to db.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{
		db: db,
	}
}

// ErrCommentNotFound allows upper layers to map absence without exposing SQL.
var ErrCommentNotFound = errors.New("comment not found")

// Create inserts a validated comment for an existing post and author.
func (r *CommentRepository) Create(
	postID int64,
	authorID int64,
	body string,
) (int64, error) {
	result, err := r.db.Exec(
		`
			INSERT INTO comments (
				post_id,
				author_id,
				body,
				created_at
			)
			VALUES (?, ?, ?, ?)
		`,
		postID,
		authorID,
		body,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(
			err.Error(),
			"FOREIGN KEY constraint failed",
		) {
			return 0, ErrPostNotFound
		}

		return 0, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return commentID, nil
}

// PostIDForComment resolves the redirect target after a comment reaction.
func (r *CommentRepository) PostIDForComment(
	commentID int64,
) (int64, error) {
	var postID int64

	err := r.db.QueryRow(`
		SELECT post_id
		FROM comments
		WHERE id = ?
	`, commentID).Scan(&postID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrCommentNotFound
	}

	if err != nil {
		return 0, err
	}

	return postID, nil
}
