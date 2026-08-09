package database

import (
	"path/filepath"
	"testing"
)

func TestOpenTemporaryDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() returned error: %v", err)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	_, err := Open("")

	if err == nil {
		t.Fatal("Open() error = nil, want error")
	}
}
func TestOpenEnablesForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	defer db.Close()

	var enabled int

	err = db.QueryRow("PRAGMA foreign_keys").Scan(&enabled)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys returned error: %v", err)
	}

	if enabled != 1 {
		t.Fatalf("foreign_keys = %d, want 1", enabled)
	}
}
func TestOpenRejectsInvalidPath(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"missing",
		"forum.db",
	)

	_, err := Open(path)

	if err == nil {
		t.Fatal("Open() error = nil, want error")
	}
}
