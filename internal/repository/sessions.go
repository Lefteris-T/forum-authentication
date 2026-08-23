package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forum/internal/model"
)

// ErrSessionNotFound is returned without leaking database-specific errors.
var ErrSessionNotFound = errors.New("session not found")

// SessionRepository stores server-side session records.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository binds session operations to db.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

// Replace atomically removes a user's previous session and creates the new one,
// enforcing the single-active-session rule across browser logins.
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

// ByID resolves the server-side state associated with a session cookie.
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

// Delete invalidates one session during logout.
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

// DeleteExpired removes records that can no longer authenticate requests.
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
