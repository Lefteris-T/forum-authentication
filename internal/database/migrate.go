package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migrate applies numbered SQL files in order and skips versions already
// recorded in schema_migrations.
func Migrate(db *sql.DB, dir string) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	files := make([]string, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		files = append(files, entry.Name())
	}

	sort.Strings(files)

	for _, name := range files {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}

		if applied {
			continue
		}

		path := filepath.Join(dir, name)

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := applyMigration(db, version, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}

func ensureMigrationTable(db *sql.DB) error {
	// This bookkeeping table records schema history, not forum domain data.
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	return nil
}

func migrationVersion(name string) (int, error) {
	// A filename such as 002_forum_content.sql has migration version 2.
	parts := strings.SplitN(name, "_", 2)

	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid migration filename %q", name)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf(
			"invalid migration version in %q: %w",
			name,
			err,
		)
	}

	return version, nil
}

func migrationApplied(db *sql.DB, version int) (bool, error) {
	var count int

	err := db.QueryRow(
		`
		SELECT COUNT(*)
		FROM schema_migrations
		WHERE version = ?
		`,
		version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf(
			"check migration version %d: %w",
			version,
			err,
		)
	}

	return count > 0, nil
}

func applyMigration(db *sql.DB, version int, migration string) error {
	// The schema change and its version record are atomic. A failure rolls both
	// back, leaving the migration safe to retry on the next startup.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}

	defer tx.Rollback()

	if _, err := tx.Exec(migration); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	if _, err := tx.Exec(
		`
		INSERT INTO schema_migrations (version)
		VALUES (?)
		`,
		version,
	); err != nil {
		return fmt.Errorf("record migration version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}

	return nil
}
