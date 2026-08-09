package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeedCategories(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	rows, err := db.Query(`
		SELECT name
		FROM categories
		ORDER BY id
	`)
	if err != nil {
		t.Fatalf("query categories: %v", err)
	}
	defer rows.Close()

	var got []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan category: %v", err)
		}

		got = append(got, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	want := []string{
		"General",
		"Go",
		"JavaScript",
		"DevOps",
	}

	if len(got) != len(want) {
		t.Fatalf("category count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(
				"category %d = %q, want %q",
				i,
				got[i],
				want[i],
			)
		}
	}
}
func TestSeedCategoriesDoesNotDuplicateOnSecondRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	var count int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM categories
	`).Scan(&count)
	if err != nil {
		t.Fatalf("count categories: %v", err)
	}

	if count != 4 {
		t.Fatalf("category count = %d, want 4", count)
	}
}
func TestExistingDatabaseReceivesSeedMigrationOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	migrationsDir := t.TempDir()

	firstMigration := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY
		);
	`

	secondMigration := `
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
	`

	thirdMigration := `
		INSERT INTO categories (name)
		VALUES
			('General'),
			('Go'),
			('JavaScript'),
			('DevOps');
	`

	if err := os.WriteFile(
		filepath.Join(migrationsDir, "001_auth.sql"),
		[]byte(firstMigration),
		0o644,
	); err != nil {
		t.Fatalf("write migration 001: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(migrationsDir, "002_forum_content.sql"),
		[]byte(secondMigration),
		0o644,
	); err != nil {
		t.Fatalf("write migration 002: %v", err)
	}

	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatalf("initial Migrate() error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(migrationsDir, "003_seed_categories.sql"),
		[]byte(thirdMigration),
		0o644,
	); err != nil {
		t.Fatalf("write migration 003: %v", err)
	}

	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatalf("third Migrate() error: %v", err)
	}

	var categoryCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM categories
	`).Scan(&categoryCount)
	if err != nil {
		t.Fatalf("count categories: %v", err)
	}

	if categoryCount != 4 {
		t.Fatalf(
			"category count = %d, want 4",
			categoryCount,
		)
	}

	var migrationCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM schema_migrations
		WHERE version = 3
	`).Scan(&migrationCount)
	if err != nil {
		t.Fatalf("count migration 003: %v", err)
	}

	if migrationCount != 1 {
		t.Fatalf(
			"migration 003 count = %d, want 1",
			migrationCount,
		)
	}
}
