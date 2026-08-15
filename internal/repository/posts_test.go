package repository

import (
	"path/filepath"
	"testing"

	"forum/internal/database"
)

func TestPostRepositoryCreateWithOneCategory(t *testing.T) {
	dbPath := filepath.Join(
		t.TempDir(),
		"forum.db",
	)

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open(): %v", err)
	}
	defer db.Close()

	err = database.Migrate(
		db,
		filepath.Join("..", "..", "migrations"),
	)
	if err != nil {
		t.Fatalf("database.Migrate(): %v", err)
	}

	users := NewUserRepository(db)

	userID, err := users.Create(
		"lefteris@example.com",
		"lefteris",
		"password-hash",
	)
	if err != nil {
		t.Fatalf("users.Create(): %v", err)
	}

	posts := NewPostRepository(db)

	postID, err := posts.Create(
		userID,
		"My first post",
		"This is the body",
		[]int64{1},
	)
	if err != nil {
		t.Fatalf("posts.Create(): %v", err)
	}

	if postID == 0 {
		t.Fatal("postID = 0, want non-zero")
	}

	var gotTitle string
	var gotAuthorID int64

	err = db.QueryRow(`
		SELECT title, author_id
		FROM posts
		WHERE id = ?
	`, postID).Scan(
		&gotTitle,
		&gotAuthorID,
	)
	if err != nil {
		t.Fatalf("query post: %v", err)
	}

	if gotTitle != "My first post" {
		t.Fatalf(
			"title = %q, want %q",
			gotTitle,
			"My first post",
		)
	}

	if gotAuthorID != userID {
		t.Fatalf(
			"author_id = %d, want %d",
			gotAuthorID,
			userID,
		)
	}

	var categoryID int64

	err = db.QueryRow(`
		SELECT category_id
		FROM post_categories
		WHERE post_id = ?
	`, postID).Scan(&categoryID)
	if err != nil {
		t.Fatalf("query post category: %v", err)
	}

	if categoryID != 1 {
		t.Fatalf(
			"category_id = %d, want 1",
			categoryID,
		)
	}
}
func TestPostRepositoryCreateWithSeveralCategories(t *testing.T) {
	dbPath := filepath.Join(
		t.TempDir(),
		"forum.db",
	)

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open(): %v", err)
	}
	defer db.Close()

	err = database.Migrate(
		db,
		filepath.Join("..", "..", "migrations"),
	)
	if err != nil {
		t.Fatalf("database.Migrate(): %v", err)
	}

	users := NewUserRepository(db)

	userID, err := users.Create(
		"lefteris@example.com",
		"lefteris",
		"password-hash",
	)
	if err != nil {
		t.Fatalf("users.Create(): %v", err)
	}

	posts := NewPostRepository(db)

	postID, err := posts.Create(
		userID,
		"Post with categories",
		"Body",
		[]int64{1, 2, 3},
	)
	if err != nil {
		t.Fatalf("posts.Create(): %v", err)
	}

	rows, err := db.Query(`
		SELECT category_id
		FROM post_categories
		WHERE post_id = ?
		ORDER BY category_id
	`, postID)
	if err != nil {
		t.Fatalf("query post categories: %v", err)
	}
	defer rows.Close()

	var got []int64

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			t.Fatalf("rows.Scan(): %v", err)
		}

		got = append(got, id)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(): %v", err)
	}

	want := []int64{1, 2, 3}

	if len(got) != len(want) {
		t.Fatalf(
			"category count = %d, want %d",
			len(got),
			len(want),
		)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(
				"category[%d] = %d, want %d",
				i,
				got[i],
				want[i],
			)
		}
	}
}
func TestPostRepositoryCreateRollsBackOnUnknownCategory(t *testing.T) {
	dbPath := filepath.Join(
		t.TempDir(),
		"forum.db",
	)

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open(): %v", err)
	}
	defer db.Close()

	err = database.Migrate(
		db,
		filepath.Join("..", "..", "migrations"),
	)
	if err != nil {
		t.Fatalf("database.Migrate(): %v", err)
	}

	users := NewUserRepository(db)

	userID, err := users.Create(
		"lefteris@example.com",
		"lefteris",
		"password-hash",
	)
	if err != nil {
		t.Fatalf("users.Create(): %v", err)
	}

	posts := NewPostRepository(db)

	_, err = posts.Create(
		userID,
		"Should rollback",
		"This post must not remain",
		[]int64{1, 999},
	)
	if err == nil {
		t.Fatal("posts.Create() error = nil, want error")
	}

	var postCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE title = ?
	`, "Should rollback").Scan(&postCount)
	if err != nil {
		t.Fatalf("count posts: %v", err)
	}

	if postCount != 0 {
		t.Fatalf(
			"post count = %d, want 0 after rollback",
			postCount,
		)
	}

	var linkCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM post_categories
	`).Scan(&linkCount)
	if err != nil {
		t.Fatalf("count post_categories: %v", err)
	}

	if linkCount != 0 {
		t.Fatalf(
			"post_categories count = %d, want 0 after rollback",
			linkCount,
		)
	}
}
