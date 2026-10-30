package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// BudgetRow is a budgets row joined with its category name and the calendar-
// month actual spend (AC10). Amount and Actual are exact NUMERIC text.
type BudgetRow struct {
	ID           int64
	CategoryID   int64
	CategoryName string
	Year         int
	Month        int
	Amount       string
	Actual       string
}

// ListBudgets returns the user's budgets for one calendar month with the
// actual expense sum computed over a half-open date range (design §2):
// tx_date >= make_date(year, month, 1) AND < first day + 1 month.
func (s *Store) ListBudgets(ctx context.Context, userID int64, year, month int) ([]BudgetRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT b.id, b.category_id, c.name, b.year, b.month,
		       b.amount::text AS amount,
		       COALESCE(SUM(t.amount), 0)::text AS actual
		FROM budgets b
		JOIN categories c ON c.id = b.category_id AND c.user_id = b.user_id
		LEFT JOIN transactions t
		       ON t.category_id = b.category_id AND t.user_id = b.user_id
		      AND t.type = 'EXPENSE'
		      AND t.tx_date >= make_date(b.year, b.month, 1)
		      AND t.tx_date <  make_date(b.year, b.month, 1) + INTERVAL '1 month'
		WHERE b.user_id = $1 AND b.year = $2 AND b.month = $3
		GROUP BY b.id, c.name
		ORDER BY c.name`,
		userID, year, month,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []BudgetRow{}
	for rows.Next() {
		var b BudgetRow
		if err := rows.Scan(&b.ID, &b.CategoryID, &b.CategoryName, &b.Year, &b.Month,
			&b.Amount, &b.Actual); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BudgetMeta identifies an existing budget row without the computed figures.
type BudgetMeta struct {
	ID         int64
	CategoryID int64
	Year       int
	Month      int
}

// GetBudget returns one of the user's budgets, or (nil, nil) when absent.
func (s *Store) GetBudget(ctx context.Context, userID, id int64) (*BudgetMeta, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, category_id, year, month FROM budgets WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	var b BudgetMeta
	err := row.Scan(&b.ID, &b.CategoryID, &b.Year, &b.Month)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBudget inserts a budget row. A duplicate (user, category, year,
// month) surfaces as unique violation 23505 (AC8); a wrong-type or foreign
// category as FK violation 23503 (AC7/AC2).
func (s *Store) CreateBudget(ctx context.Context, userID, categoryID int64, year, month int, amount string) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO budgets (user_id, category_id, year, month, amount, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		RETURNING id`,
		userID, categoryID, year, month, amount,
	).Scan(&id)
	return id, err
}

// UpdateBudgetAmount edits only the amount (D38). Returns false when the row
// does not exist for this user.
func (s *Store) UpdateBudgetAmount(ctx context.Context, userID, id int64, amount string) (bool, error) {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE budgets SET amount = $1, updated_at = now() WHERE id = $2 AND user_id = $3`,
		amount, id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteBudget removes one of the user's budgets. Returns false when absent.
func (s *Store) DeleteBudget(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := s.Pool.Exec(ctx,
		`DELETE FROM budgets WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
