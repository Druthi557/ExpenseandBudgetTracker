// Package store is the repository layer: explicit SQL against PostgreSQL via
// pgx/v5, no ORM. Money crosses this layer as NUMERIC(12,2) text so values
// stay exact end-to-end. Every query is scoped by user_id (AC2).
package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store groups the repository methods over a connection pool.
type Store struct {
	Pool *pgxpool.Pool
}

// New returns a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}
