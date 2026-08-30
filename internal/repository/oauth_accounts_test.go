package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"forum/internal/database"
)

func TestOAuthAccountRepositoryCreateAndFindByProviderUserID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	userResult, err := db.Exec(`
		INSERT INTO users (
			email,
			username,
			password_hash,
			created_at
		)
		VALUES (?, ?, NULL, ?)
	`,
		"oauth@example.com",
		"oauth-user",
		"2026-08-29T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert user error: %v", err)
	}

	userID, err := userResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	id, err := repo.Create(
		userID,
		"github",
		"123456",
		"oauth@example.com",
	)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	account, err := repo.FindByProviderUserID(
		"github",
		"123456",
	)
	if err != nil {
		t.Fatalf("FindByProviderUserID() error: %v", err)
	}

	if account.ID != id {
		t.Errorf(
			"account.ID = %d, want %d",
			account.ID,
			id,
		)
	}

	if account.UserID != userID {
		t.Errorf(
			"account.UserID = %d, want %d",
			account.UserID,
			userID,
		)
	}

	if account.Provider != "github" {
		t.Errorf(
			"account.Provider = %q, want %q",
			account.Provider,
			"github",
		)
	}

	if account.ProviderUserID != "123456" {
		t.Errorf(
			"account.ProviderUserID = %q, want %q",
			account.ProviderUserID,
			"123456",
		)
	}

	if account.Email != "oauth@example.com" {
		t.Errorf(
			"account.Email = %q, want %q",
			account.Email,
			"oauth@example.com",
		)
	}
}
func TestOAuthAccountRepositoryFindByUserID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	userResult, err := db.Exec(`
		INSERT INTO users (
			email,
			username,
			password_hash,
			created_at
		)
		VALUES (?, ?, NULL, ?)
	`,
		"oauth@example.com",
		"oauth-user",
		"2026-08-29T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert user error: %v", err)
	}

	userID, err := userResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	_, err = repo.Create(
		userID,
		"github",
		"123456",
		"oauth@example.com",
	)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	account, err := repo.FindByUserID(userID)
	if err != nil {
		t.Fatalf("FindByUserID() error: %v", err)
	}

	if account.UserID != userID {
		t.Errorf(
			"account.UserID = %d, want %d",
			account.UserID,
			userID,
		)
	}

	if account.Provider != "github" {
		t.Errorf(
			"account.Provider = %q, want %q",
			account.Provider,
			"github",
		)
	}
}
func TestOAuthAccountRepositoryReturnsNotFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	_, err = repo.FindByProviderUserID(
		"github",
		"missing-id",
	)

	if !errors.Is(err, ErrOAuthAccountNotFound) {
		t.Fatalf(
			"FindByProviderUserID() error = %v, want %v",
			err,
			ErrOAuthAccountNotFound,
		)
	}
}
func TestOAuthAccountRepositoryReturnsStableConflictErrors(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	user1, err := db.Exec(`
		INSERT INTO users (email, username, password_hash, created_at)
		VALUES (?, ?, NULL, ?)
	`, "one@example.com", "one", "2026-08-29T00:00:00Z")
	if err != nil {
		t.Fatalf("insert user1 error: %v", err)
	}

	user1ID, _ := user1.LastInsertId()

	user2, err := db.Exec(`
		INSERT INTO users (email, username, password_hash, created_at)
		VALUES (?, ?, NULL, ?)
	`, "two@example.com", "two", "2026-08-29T00:00:00Z")
	if err != nil {
		t.Fatalf("insert user2 error: %v", err)
	}

	user2ID, _ := user2.LastInsertId()

	repo := NewOAuthAccountRepository(db)

	_, err = repo.Create(
		user1ID,
		"github",
		"123456",
		"one@example.com",
	)
	if err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	t.Run("same provider identity", func(t *testing.T) {
		_, err := repo.Create(
			user2ID,
			"github",
			"123456",
			"two@example.com",
		)

		if !errors.Is(err, ErrOAuthIdentityExists) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrOAuthIdentityExists,
			)
		}
	})

	t.Run("same provider for same user", func(t *testing.T) {
		_, err := repo.Create(
			user1ID,
			"github",
			"different-id",
			"one@example.com",
		)

		if !errors.Is(err, ErrOAuthProviderExistsForUser) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrOAuthProviderExistsForUser,
			)
		}
	})
}
func TestOAuthAccountRepositoryCreateUserWithOAuthAccount(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	userID, err := repo.CreateUserWithOAuthAccount(
		"oauth@example.com",
		"oauth-user",
		"github",
		"123456",
	)
	if err != nil {
		t.Fatalf("CreateUserWithOAuthAccount() error: %v", err)
	}

	var (
		email        string
		username     string
		passwordHash sql.NullString
	)

	err = db.QueryRow(`
		SELECT
			email,
			username,
			password_hash
		FROM users
		WHERE id = ?
	`,
		userID,
	).Scan(
		&email,
		&username,
		&passwordHash,
	)
	if err != nil {
		t.Fatalf("query created user error: %v", err)
	}

	if email != "oauth@example.com" {
		t.Errorf(
			"email = %q, want %q",
			email,
			"oauth@example.com",
		)
	}

	if username != "oauth-user" {
		t.Errorf(
			"username = %q, want %q",
			username,
			"oauth-user",
		)
	}

	if passwordHash.Valid {
		t.Errorf(
			"password_hash = %q, want NULL",
			passwordHash.String,
		)
	}

	var (
		oauthUserID    int64
		provider       string
		providerUserID string
		oauthEmail     string
	)

	err = db.QueryRow(`
		SELECT
			user_id,
			provider,
			provider_user_id,
			email
		FROM oauth_accounts
		WHERE user_id = ?
	`,
		userID,
	).Scan(
		&oauthUserID,
		&provider,
		&providerUserID,
		&oauthEmail,
	)
	if err != nil {
		t.Fatalf("query oauth account error: %v", err)
	}

	if oauthUserID != userID {
		t.Errorf(
			"oauth user_id = %d, want %d",
			oauthUserID,
			userID,
		)
	}

	if provider != "github" {
		t.Errorf(
			"provider = %q, want %q",
			provider,
			"github",
		)
	}

	if providerUserID != "123456" {
		t.Errorf(
			"provider_user_id = %q, want %q",
			providerUserID,
			"123456",
		)
	}

	if oauthEmail != "oauth@example.com" {
		t.Errorf(
			"oauth email = %q, want %q",
			oauthEmail,
			"oauth@example.com",
		)
	}
}
func TestOAuthAccountRepositoryCreateUserWithOAuthAccountRollsBackOnOAuthFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	// Existing OAuth identity that will force the second insert to fail.
	existingUserResult, err := db.Exec(`
		INSERT INTO users (
			email,
			username,
			password_hash,
			created_at
		)
		VALUES (?, ?, NULL, ?)
	`,
		"existing@example.com",
		"existing-user",
		"2026-08-29T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert existing user error: %v", err)
	}

	existingUserID, err := existingUserResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO oauth_accounts (
			user_id,
			provider,
			provider_user_id,
			email,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		existingUserID,
		"github",
		"123456",
		"existing@example.com",
		"2026-08-29T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert existing oauth account error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	_, err = repo.CreateUserWithOAuthAccount(
		"new@example.com",
		"new-user",
		"github",
		"123456",
	)
	if err == nil {
		t.Fatal("CreateUserWithOAuthAccount() error = nil, want error")
	}

	var count int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE email = ?
	`, "new@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("count new user error: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"new user count = %d, want 0 after rollback",
			count,
		)
	}
}
func TestOAuthAccountRepositoryConcurrentDuplicateCreate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	type result struct {
		err error
	}

	results := make(chan result, 2)

	for i := 0; i < 2; i++ {
		i := i

		go func() {
			_, err := repo.CreateUserWithOAuthAccount(
				fmt.Sprintf("oauth%d@example.com", i),
				fmt.Sprintf("oauth-user-%d", i),
				"github",
				"same-provider-id",
			)

			results <- result{err: err}
		}()
	}

	var (
		successes int
		failures  int
	)

	for i := 0; i < 2; i++ {
		res := <-results

		if res.err == nil {
			successes++
		} else {
			failures++
		}
	}

	if successes != 1 {
		t.Fatalf("successful creates = %d, want 1", successes)
	}

	if failures != 1 {
		t.Fatalf("failed creates = %d, want 1", failures)
	}

	var oauthCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM oauth_accounts
		WHERE provider = ?
		  AND provider_user_id = ?
	`,
		"github",
		"same-provider-id",
	).Scan(&oauthCount)
	if err != nil {
		t.Fatalf("count oauth accounts error: %v", err)
	}

	if oauthCount != 1 {
		t.Fatalf("oauth account count = %d, want 1", oauthCount)
	}

	var userCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE email IN (?, ?)
	`,
		"oauth0@example.com",
		"oauth1@example.com",
	).Scan(&userCount)
	if err != nil {
		t.Fatalf("count users error: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("user count = %d, want 1", userCount)
	}
}
func TestOAuthAccountRepositoryCreateUserWithOAuthAccountReturnsUsernameConflict(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("database.Migrate() error: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (
			email,
			username,
			password_hash,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		"existing@example.com",
		"octocat",
		"some-hash",
		"2026-08-30T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert existing user error: %v", err)
	}

	repo := NewOAuthAccountRepository(db)

	_, err = repo.CreateUserWithOAuthAccount(
		"oauth@example.com",
		"octocat",
		"github",
		"123456",
	)

	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf(
			"CreateUserWithOAuthAccount() error = %v, want %v",
			err,
			ErrUsernameExists,
		)
	}

	var count int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE email = ?
	`, "oauth@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("count oauth user error: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"oauth user count = %d, want 0 after username conflict",
			count,
		)
	}
}
