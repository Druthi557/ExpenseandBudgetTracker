package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Transaction is a row of the transactions table joined with its category
// name. Amount is NUMERIC(12,2) received as exact text (e.g. "40.00").
type Transaction struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Amount       string    `json:"amount"`
	Date         time.Time `json:"date"`
	Description  *string   `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TxFilter carries the FR3.5 list filters; nil fields mean "no filter".
type TxFilter struct {
	Type       *string
	CategoryID *int64
	From       *time.Time
	To         *time.Time
}

const transactionColumns = `
	t.id, t.user_id, t.type, t.category_id, c.name AS category_name,
	t.amount::text AS amount, t.tx_date, t.description, t.created_at, t.updated_at`

// ListTransactions returns the user's transactions newest-first (D12:
// tx_date DESC, id DESC) with the requested filters applied.
func (s *Store) ListTransactions(ctx context.Context, userID int64, f TxFilter) ([]Transaction, error) {
	var sb strings.Builder
	sb.WriteString("SELECT " + transactionColumns + `
		FROM transactions t
		JOIN categories c ON c.id = t.category_id AND c.user_id = t.user_id
		WHERE t.user_id = $1`)

	args := []any{userID}
	if f.Type != nil {
		args = append(args, *f.Type)
		sb.WriteString(" AND t.type = " + argPlaceholder(len(args)))
	}
	if f.CategoryID != nil {
		args = append(args, *f.CategoryID)
		sb.WriteString(" AND t.category_id = " + argPlaceholder(len(args)))
	}
	if f.From != nil {
		args = append(args, *f.From)
		sb.WriteString(" AND t.tx_date >= " + argPlaceholder(len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		sb.WriteString(" AND t.tx_date <= " + argPlaceholder(len(args)))
	}
	sb.WriteString(" ORDER BY t.tx_date DESC, t.id DESC")

	rows, err := s.Pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}

// GetTransaction returns one of the user's transactions, or (nil, nil).
func (s *Store) GetTransaction(ctx context.Context, userID, id int64) (*Transaction, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT `+transactionColumns+`
		FROM transactions t
		JOIN categories c ON c.id = t.category_id AND c.user_id = t.user_id
		WHERE t.id = $1 AND t.user_id = $2`,
		id, userID,
	)
	var tx Transaction
	err := row.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.CategoryID, &tx.CategoryName,
		&tx.Amount, &tx.Date, &tx.Description, &tx.CreatedAt, &tx.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// CreateTransaction inserts a transaction (amount as exact numeric text).
// The composite FK (category_id, user_id, type) makes cross-user or
// wrong-type references impossible at the database level (AC2/AC6).
func (s *Store) CreateTransaction(ctx context.Context, userID int64, typ string, categoryID int64,
	amount string, date time.Time, description *string) (int64, error) {

	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, category_id, amount, tx_date, description, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		RETURNING id`,
		userID, typ, categoryID, amount, date, description,
	).Scan(&id)
	return id, err
}

// UpdateTransaction fully replaces a transaction. Returns false when the row
// does not exist for this user.
func (s *Store) UpdateTransaction(ctx context.Context, userID, id int64, typ string,
	categoryID int64, amount string, date time.Time, description *string) (bool, error) {

	tag, err := s.Pool.Exec(ctx, `
		UPDATE transactions
		SET type = $1, category_id = $2, amount = $3, tx_date = $4, description = $5, updated_at = now()
		WHERE id = $6 AND user_id = $7`,
		typ, categoryID, amount, date, description, id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteTransaction removes one of the user's transactions. Returns false
// when the row does not exist for this user.
func (s *Store) DeleteTransaction(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := s.Pool.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func argPlaceholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func scanTransactions(rows pgx.Rows) ([]Transaction, error) {
	txs := []Transaction{}
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.CategoryID, &tx.CategoryName,
			&tx.Amount, &tx.Date, &tx.Description, &tx.CreatedAt, &tx.UpdatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}
