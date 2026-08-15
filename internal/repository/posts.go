package repository

import (
	"database/sql"
	"time"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{
		db: db,
	}
}

func (r *PostRepository) Create(
	authorID int64,
	title string,
	body string,
	categoryIDs []int64,
) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`
			INSERT INTO posts (
				author_id,
				title,
				body,
				created_at
			)
			VALUES (?, ?, ?, ?)
		`,
		authorID,
		title,
		body,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, categoryID := range categoryIDs {
		_, err := tx.Exec(
			`
				INSERT INTO post_categories (
					post_id,
					category_id
				)
				VALUES (?, ?)
			`,
			postID,
			categoryID,
		)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return postID, nil
}
