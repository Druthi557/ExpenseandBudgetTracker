package store

import (
	"context"
)

// MonthTotals carries the FR5.1–5.3 aggregates as exact NUMERIC text.
type MonthTotals struct {
	IncomeText  string
	ExpenseText string
}

// MonthTotals computes total income and expenses for one calendar month.
// Every request recomputes live — nothing cached (AC9).
func (s *Store) MonthTotals(ctx context.Context, userID int64, year, month int) (MonthTotals, error) {
	var t MonthTotals
	err := s.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type = 'INCOME'),  0)::text AS income,
			COALESCE(SUM(amount) FILTER (WHERE type = 'EXPENSE'), 0)::text AS expense
		FROM transactions
		WHERE user_id = $1
		  AND tx_date >= make_date($2, $3, 1)
		  AND tx_date <  make_date($2, $3, 1) + INTERVAL '1 month'`,
		userID, year, month,
	).Scan(&t.IncomeText, &t.ExpenseText)
	return t, err
}

// CategoryTotal is one row of FR5.4: expense totals grouped by category for
// the selected month.
type CategoryTotal struct {
	CategoryID int64
	Name       string
	TotalText  string
}

// ExpensesByCategory aggregates the month's expenses per category.
func (s *Store) ExpensesByCategory(ctx context.Context, userID int64, year, month int) ([]CategoryTotal, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT t.category_id, c.name, SUM(t.amount)::text AS total
		FROM transactions t
		JOIN categories c ON c.id = t.category_id AND c.user_id = t.user_id
		WHERE t.user_id = $1
		  AND t.type = 'EXPENSE'
		  AND t.tx_date >= make_date($2, $3, 1)
		  AND t.tx_date <  make_date($2, $3, 1) + INTERVAL '1 month'
		GROUP BY t.category_id, c.name
		ORDER BY SUM(t.amount) DESC, c.name`,
		userID, year, month,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CategoryTotal{}
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.CategoryID, &ct.Name, &ct.TotalText); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

// RecentTransactions returns the user's N most recent transactions overall —
// any month (D13) — newest first with id DESC as the same-date tie-break.
func (s *Store) RecentTransactions(ctx context.Context, userID int64, limit int32) ([]Transaction, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+transactionColumns+`
		FROM transactions t
		JOIN categories c ON c.id = t.category_id AND c.user_id = t.user_id
		WHERE t.user_id = $1
		ORDER BY t.tx_date DESC, t.id DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransactions(rows)
}
