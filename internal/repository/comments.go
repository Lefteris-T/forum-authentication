package repository

import (
	"database/sql"
	"strings"
	"time"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{
		db: db,
	}
}

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
