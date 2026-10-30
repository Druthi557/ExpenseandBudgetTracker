package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// User is a row of the users table. PasswordHash is only ever compared
// server-side; it must never reach a response DTO.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time // TIMESTAMPTZ
}

// CreateUser inserts a new account and returns its id. A duplicate email
// surfaces as a PostgreSQL unique violation (23505) for the caller to map.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, passwordHash,
	).Scan(&id)
	return id, err
}

// GetUserByEmail looks a user up case-insensitively by email.
// Returns (nil, nil) when no row matches.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE lower(email) = lower($1)`,
		email,
	)
	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
