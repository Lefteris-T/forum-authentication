package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forum/internal/model"
)

var (
	ErrOAuthAccountNotFound       = errors.New("oauth account not found")
	ErrOAuthIdentityExists        = errors.New("oauth identity already exists")
	ErrOAuthProviderExistsForUser = errors.New("oauth provider already exists for user")
)

// OAuthAccountRepository persists external OAuth identities.
type OAuthAccountRepository struct {
	db *sql.DB
}

// NewOAuthAccountRepository binds OAuth account operations to db.
func NewOAuthAccountRepository(db *sql.DB) *OAuthAccountRepository {
	return &OAuthAccountRepository{
		db: db,
	}
}

func (r *OAuthAccountRepository) Create(
	userID int64,
	provider string,
	providerUserID string,
	email string,
) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO oauth_accounts (
			user_id,
			provider,
			provider_user_id,
			email,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)

		ON CONFLICT(provider, provider_user_id)
		DO NOTHING

		ON CONFLICT(user_id, provider)
		DO NOTHING
	`,
		userID,
		provider,
		providerUserID,
		email,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("create oauth account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get oauth account rows affected: %w", err)
	}

	if rowsAffected == 0 {
		var exists int

		err := r.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM oauth_accounts
				WHERE provider = ?
				  AND provider_user_id = ?
			)
		`,
			provider,
			providerUserID,
		).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf(
				"check existing oauth identity: %w",
				err,
			)
		}

		if exists == 1 {
			return 0, ErrOAuthIdentityExists
		}

		err = r.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM oauth_accounts
				WHERE user_id = ?
				  AND provider = ?
			)
		`,
			userID,
			provider,
		).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf(
				"check existing oauth provider for user: %w",
				err,
			)
		}

		if exists == 1 {
			return 0, ErrOAuthProviderExistsForUser
		}

		return 0, errors.New("oauth account was not created")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf(
			"get inserted oauth account id: %w",
			err,
		)
	}

	return id, nil
}

// FindByProviderUserID finds a local OAuth account using the provider's stable ID.
func (r *OAuthAccountRepository) FindByProviderUserID(
	provider string,
	providerUserID string,
) (model.OAuthAccount, error) {
	var account model.OAuthAccount
	var createdAt string

	err := r.db.QueryRow(`
		SELECT
			id,
			user_id,
			provider,
			provider_user_id,
			email,
			created_at
		FROM oauth_accounts
		WHERE provider = ?
		  AND provider_user_id = ?
	`,
		provider,
		providerUserID,
	).Scan(
		&account.ID,
		&account.UserID,
		&account.Provider,
		&account.ProviderUserID,
		&account.Email,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.OAuthAccount{}, ErrOAuthAccountNotFound
		}

		return model.OAuthAccount{}, fmt.Errorf(
			"find oauth account by provider user id: %w",
			err,
		)
	}

	account.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return model.OAuthAccount{}, fmt.Errorf(
			"parse oauth account created_at: %w",
			err,
		)
	}

	return account, nil
}
func (r *OAuthAccountRepository) FindByUserID(
	userID int64,
) (model.OAuthAccount, error) {
	var account model.OAuthAccount
	var createdAt string

	err := r.db.QueryRow(`
		SELECT
			id,
			user_id,
			provider,
			provider_user_id,
			email,
			created_at
		FROM oauth_accounts
		WHERE user_id = ?
	`,
		userID,
	).Scan(
		&account.ID,
		&account.UserID,
		&account.Provider,
		&account.ProviderUserID,
		&account.Email,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.OAuthAccount{}, ErrOAuthAccountNotFound
		}

		return model.OAuthAccount{}, fmt.Errorf(
			"find oauth account by user id: %w",
			err,
		)
	}

	account.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return model.OAuthAccount{}, fmt.Errorf(
			"parse oauth account created_at: %w",
			err,
		)
	}

	return account, nil
}
func (r *OAuthAccountRepository) CreateUserWithOAuthAccount(
	email string,
	username string,
	provider string,
	providerUserID string,
) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin oauth user transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	userResult, err := tx.Exec(`
		INSERT INTO users (
			email,
			username,
			password_hash,
			created_at
		)
		VALUES (?, ?, NULL, ?)
	`,
		email,
		username,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("create oauth user: %w", err)
	}

	userID, err := userResult.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get oauth user id: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO oauth_accounts (
			user_id,
			provider,
			provider_user_id,
			email,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		userID,
		provider,
		providerUserID,
		email,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("create oauth account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit oauth user transaction: %w", err)
	}

	return userID, nil
}
