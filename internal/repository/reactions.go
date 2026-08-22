package repository

import (
	"database/sql"
	"errors"

	"forum/internal/model"
)

type ReactionRepository struct {
	db *sql.DB
}

func NewReactionRepository(
	db *sql.DB,
) *ReactionRepository {
	return &ReactionRepository{
		db: db,
	}
}

func (r *ReactionRepository) SetPostReaction(
	userID int64,
	postID int64,
	value model.Reaction,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int

	err = tx.QueryRow(`
		SELECT 1
		FROM posts
		WHERE id = ?
	`, postID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrPostNotFound
	}

	if err != nil {
		return err
	}

	err = applyReactionTransition(
		tx,
		userID,
		postID,
		value,
		`
			SELECT value
			FROM post_reactions
			WHERE user_id = ?
			  AND post_id = ?
		`,
		`
			INSERT INTO post_reactions (
				user_id,
				post_id,
				value
			)
			VALUES (?, ?, ?)
		`,
		`
			DELETE FROM post_reactions
			WHERE user_id = ?
			  AND post_id = ?
		`,
		`
			UPDATE post_reactions
			SET value = ?
			WHERE user_id = ?
			  AND post_id = ?
		`,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ReactionRepository) SetCommentReaction(
	userID int64,
	commentID int64,
	value model.Reaction,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int

	err = tx.QueryRow(`
		SELECT 1
		FROM comments
		WHERE id = ?
	`, commentID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrCommentNotFound
	}

	if err != nil {
		return err
	}

	err = applyReactionTransition(
		tx,
		userID,
		commentID,
		value,
		`
			SELECT value
			FROM comment_reactions
			WHERE user_id = ?
			  AND comment_id = ?
		`,
		`
			INSERT INTO comment_reactions (
				user_id,
				comment_id,
				value
			)
			VALUES (?, ?, ?)
		`,
		`
			DELETE FROM comment_reactions
			WHERE user_id = ?
			  AND comment_id = ?
		`,
		`
			UPDATE comment_reactions
			SET value = ?
			WHERE user_id = ?
			  AND comment_id = ?
		`,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func applyReactionTransition(
	tx *sql.Tx,
	userID int64,
	targetID int64,
	value model.Reaction,
	selectQuery string,
	insertQuery string,
	deleteQuery string,
	updateQuery string,
) error {
	var currentValue int

	err := tx.QueryRow(
		selectQuery,
		userID,
		targetID,
	).Scan(&currentValue)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.Exec(
			insertQuery,
			userID,
			targetID,
			value,
		)

		return err
	}

	if err != nil {
		return err
	}

	if currentValue == int(value) {
		_, err = tx.Exec(
			deleteQuery,
			userID,
			targetID,
		)

		return err
	}

	_, err = tx.Exec(
		updateQuery,
		value,
		userID,
		targetID,
	)

	return err
}
