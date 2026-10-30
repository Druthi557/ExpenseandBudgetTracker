package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Category is a row of the categories table.
type Category struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Type   string `json:"type"` // INCOME | EXPENSE
}

// ListCategories returns the user's categories, optionally filtered by type.
func (s *Store) ListCategories(ctx context.Context, userID int64, typ *string) ([]Category, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, user_id, name, type
		FROM categories
		WHERE user_id = $1 AND ($2::text IS NULL OR type = $2)
		ORDER BY name`,
		userID, typ,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cats := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// GetCategory returns one of the user's categories, or (nil, nil) when absent.
func (s *Store) GetCategory(ctx context.Context, userID, id int64) (*Category, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, name, type FROM categories WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	c := &Category{}
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Type)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// CreateCategory inserts a category and returns its id. A duplicate name for
// the same user (case-insensitive) surfaces as a unique violation (23505).
func (s *Store) CreateCategory(ctx context.Context, userID int64, name, typ string) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3) RETURNING id`,
		userID, name, typ,
	).Scan(&id)
	return id, err
}

// UpdateCategory replaces name and type. Returns false when the row does not
// exist for this user. Retyping a referenced category violates the composite
// foreign keys (23503).
func (s *Store) UpdateCategory(ctx context.Context, userID, id int64, name, typ string) (bool, error) {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE categories SET name = $1, type = $2 WHERE id = $3 AND user_id = $4`,
		name, typ, id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteCategory removes an unused category. Returns false when the row does
// not exist for this user; PostgreSQL RESTRICT (23503) blocks referenced ones.
func (s *Store) DeleteCategory(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := s.Pool.Exec(ctx,
		`DELETE FROM categories WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
