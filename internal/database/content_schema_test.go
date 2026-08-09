package database

import (
	"path/filepath"
	"testing"
)

func TestContentMigrationCreatesForumTables(t *testing.T) {
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
		"posts",
		"comments",
		"categories",
		"post_categories",
		"post_reactions",
		"comment_reactions",
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
func TestContentSchemaConstraints(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Create one user.
	result, err := db.Exec(`
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
		t.Fatalf("insert user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error: %v", err)
	}

	// Invalid author foreign key.
	_, err = db.Exec(`
		INSERT INTO posts (
			author_id,
			title,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		9999,
		"title",
		"body",
		"2026-08-09T10:00:00Z",
	)
	if err == nil {
		t.Fatal("post with unknown author was accepted")
	}

	// Valid post.
	postResult, err := db.Exec(`
		INSERT INTO posts (
			author_id,
			title,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		userID,
		"title",
		"body",
		"2026-08-09T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}

	postID, err := postResult.LastInsertId()
	if err != nil {
		t.Fatalf("post LastInsertId() error: %v", err)
	}

	// Valid category.
	categoryResult, err := db.Exec(`
		INSERT INTO categories (name)
		VALUES (?)
	`, "Go")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	categoryID, err := categoryResult.LastInsertId()
	if err != nil {
		t.Fatalf("category LastInsertId() error: %v", err)
	}

	// First post/category relation is valid.
	_, err = db.Exec(`
		INSERT INTO post_categories (
			post_id,
			category_id
		)
		VALUES (?, ?)
	`,
		postID,
		categoryID,
	)
	if err != nil {
		t.Fatalf("insert post category: %v", err)
	}

	// Same pair must not be inserted twice.
	_, err = db.Exec(`
		INSERT INTO post_categories (
			post_id,
			category_id
		)
		VALUES (?, ?)
	`,
		postID,
		categoryID,
	)
	if err == nil {
		t.Fatal("duplicate post/category relation was accepted")
	}

	// Reaction value can only be -1 or 1.
	_, err = db.Exec(`
		INSERT INTO post_reactions (
			user_id,
			post_id,
			value
		)
		VALUES (?, ?, ?)
	`,
		userID,
		postID,
		5,
	)
	if err == nil {
		t.Fatal("invalid reaction value was accepted")
	}

	// Valid like.
	_, err = db.Exec(`
		INSERT INTO post_reactions (
			user_id,
			post_id,
			value
		)
		VALUES (?, ?, ?)
	`,
		userID,
		postID,
		1,
	)
	if err != nil {
		t.Fatalf("insert reaction: %v", err)
	}

	// Same user cannot have a second reaction on same post.
	_, err = db.Exec(`
		INSERT INTO post_reactions (
			user_id,
			post_id,
			value
		)
		VALUES (?, ?, ?)
	`,
		userID,
		postID,
		-1,
	)
	if err == nil {
		t.Fatal("duplicate post reaction was accepted")
	}
}
func TestContentSchemaCascadesDeletes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	userResult, err := db.Exec(`
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
		t.Fatalf("insert user: %v", err)
	}

	userID, err := userResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId user: %v", err)
	}

	postResult, err := db.Exec(`
		INSERT INTO posts (
			author_id,
			title,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		userID,
		"title",
		"body",
		"2026-08-09T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}

	postID, err := postResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId post: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO comments (
			post_id,
			author_id,
			body,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		postID,
		userID,
		"comment",
		"2026-08-09T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}

	_, err = db.Exec(`
		DELETE FROM posts
		WHERE id = ?
	`, postID)
	if err != nil {
		t.Fatalf("delete post: %v", err)
	}

	var commentCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM comments
		WHERE post_id = ?
	`, postID).Scan(&commentCount)
	if err != nil {
		t.Fatalf("count comments: %v", err)
	}

	if commentCount != 0 {
		t.Fatalf("comments after post delete = %d, want 0", commentCount)
	}
}
