// Package repository contains parameterized SQLite persistence operations.
package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"forum/internal/model"
)

var (
	ErrCategoryNotFound  = errors.New("category not found")
	ErrDuplicateCategory = errors.New("duplicate category")
)

// CategoryRepository reads and validates the fixed category catalogue.
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository binds category operations to db.
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{
		db: db,
	}
}

// All returns categories in stable name order for forms and filters.
func (r *CategoryRepository) All() ([]model.Category, error) {
	rows, err := r.db.Query(`
		SELECT id, name
		FROM categories
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category

	for rows.Next() {
		var category model.Category

		if err := rows.Scan(
			&category.ID,
			&category.Name,
		); err != nil {
			return nil, err
		}

		categories = append(
			categories,
			category,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// ValidateIDs confirms that every submitted category exists and rejects an
// empty selection or repeated identifiers.
func (r *CategoryRepository) ValidateIDs(
	ids []int64,
) error {
	seen := make(map[int64]struct{})

	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return ErrDuplicateCategory
		}

		seen[id] = struct{}{}

		var exists int

		err := r.db.QueryRow(
			`
				SELECT 1
				FROM categories
				WHERE id = ?
			`,
			id,
		).Scan(&exists)

		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(
				"%w: %d",
				ErrCategoryNotFound,
				id,
			)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
