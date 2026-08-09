package database

import (
	"path/filepath"
	"testing"
)

func TestAuthMigrationCreatesUsersAndSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	expectedTables := []string{
		"users",
		"sessions",
	}

	for _, table := range expectedTables {
		var count int

		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM sqlite_master
			WHERE type = 'table' AND name = ?
		`, table).Scan(&count)
		if err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}

		if count != 1 {
			t.Fatalf("table %s count = %d, want 1", table, count)
		}
	}
}
func TestAuthSchemaRejectsDuplicateEmailUsernameAndSessionUser(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error: %v", err)
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
		"user@example.com",
		"lefteris",
		"hash",
		"2026-08-09T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert first user: %v", err)
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
		"user@example.com",
		"another-user",
		"hash",
		"2026-08-09T10:00:00Z",
	)
	if err == nil {
		t.Fatal("duplicate email was accepted")
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
		"another@example.com",
		"lefteris",
		"hash",
		"2026-08-09T10:00:00Z",
	)
	if err == nil {
		t.Fatal("duplicate username was accepted")
	}

	var userID int64

	err = db.QueryRow(`
		SELECT id
		FROM users
		WHERE email = ?
	`, "user@example.com").Scan(&userID)
	if err != nil {
		t.Fatalf("select user id: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sessions (
			id,
			user_id,
			expires_at,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		"session-1",
		userID,
		"2026-08-10T10:00:00Z",
		"2026-08-09T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert first session: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sessions (
			id,
			user_id,
			expires_at,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		"session-2",
		userID,
		"2026-08-10T10:00:00Z",
		"2026-08-09T10:00:00Z",
	)
	if err == nil {
		t.Fatal("second session for same user was accepted")
	}
}
