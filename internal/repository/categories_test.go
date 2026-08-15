package repository

import (
	"path/filepath"
	"testing"

	"forum/internal/database"
)

func TestCategoryRepositoryAllReturnsOrderedCategories(t *testing.T) {
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

	repo := NewCategoryRepository(db)

	categories, err := repo.All()
	if err != nil {
		t.Fatalf("All(): %v", err)
	}

	if len(categories) != 4 {
		t.Fatalf(
			"len(categories) = %d, want 4",
			len(categories),
		)
	}

	want := []string{
		"General",
		"Go",
		"JavaScript",
		"DevOps",
	}

	for i, category := range categories {
		if category.Name != want[i] {
			t.Fatalf(
				"categories[%d].Name = %q, want %q",
				i,
				category.Name,
				want[i],
			)
		}
	}
}
func TestCategoryRepositoryValidateIDs(t *testing.T) {
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

	repo := NewCategoryRepository(db)

	tests := []struct {
		name    string
		ids     []int64
		wantErr bool
	}{
		{
			name:    "known ids",
			ids:     []int64{1, 2},
			wantErr: false,
		},
		{
			name:    "unknown id",
			ids:     []int64{1, 999},
			wantErr: true,
		},
		{
			name:    "duplicate id",
			ids:     []int64{1, 1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.ValidateIDs(tt.ids)

			if tt.wantErr && err == nil {
				t.Fatal("ValidateIDs() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf(
					"ValidateIDs() error = %v, want nil",
					err,
				)
			}
		})
	}
}
