package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forum/internal/model"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

func (r *SessionRepository) Replace(
	id string,
	userID int64,
	expiresAt time.Time,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin session transaction: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
		DELETE FROM sessions
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("delete old session: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO sessions (
			id,
			user_id,
			expires_at,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		id,
		userID,
		expiresAt.UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert new session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session transaction: %w", err)
	}

	return nil
}

func (r *SessionRepository) ByID(id string) (model.Session, error) {
	var session model.Session
	var expiresAt string
	var createdAt string

	err := r.db.QueryRow(`
		SELECT
			id,
			user_id,
			expires_at,
			created_at
		FROM sessions
		WHERE id = ?
	`,
		id,
	).Scan(
		&session.ID,
		&session.UserID,
		&expiresAt,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Session{}, ErrSessionNotFound
		}

		return model.Session{}, fmt.Errorf("find session by id: %w", err)
	}

	session.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return model.Session{}, fmt.Errorf("parse session expires_at: %w", err)
	}

	session.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return model.Session{}, fmt.Errorf("parse session created_at: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) Delete(id string) error {
	_, err := r.db.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (r *SessionRepository) DeleteExpired(now time.Time) error {
	_, err := r.db.Exec(`
		DELETE FROM sessions
		WHERE expires_at <= ?
	`,
		now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}
