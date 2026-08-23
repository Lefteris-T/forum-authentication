package repository

import (
	"database/sql"
	"errors"

	"forum/internal/model"
)

// ReactionRepository applies like/dislike state transitions transactionally.
type ReactionRepository struct {
	db *sql.DB
}

// NewReactionRepository binds reaction operations to db.
func NewReactionRepository(
	db *sql.DB,
) *ReactionRepository {
	return &ReactionRepository{
		db: db,
	}
}

// SetPostReaction toggles an identical reaction off or switches an opposite
// reaction in one transaction after confirming that the post exists.
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

// SetCommentReaction applies the same transition rules to a comment.
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
	// No row means insert, the same value means delete, and the opposite value
	// means update. This shared state machine keeps post and comment behavior
	// identical.
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
