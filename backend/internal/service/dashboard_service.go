package service

import (
	"context"

	"pebt/backend/internal/money"
)

// DashboardView carries the FR5 aggregates for the selected month. Money is
// exact 2-dp strings; totals are "0.00" for months without activity.
type DashboardView struct {
	Year               int                 `json:"year"`
	Month              int                 `json:"month"`
	TotalIncome        string              `json:"total_income"`
	TotalExpense       string              `json:"total_expense"`
	RemainingBalance   string              `json:"remaining_balance"`
	ExpensesByCategory []CategoryTotalView `json:"expenses_by_category"`
	RecentTransactions []TxView            `json:"recent_transactions"`
}

// CategoryTotalView is one FR5.4 row: the month's expense total per category.
type CategoryTotalView struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Total        string `json:"total"`
}

// Dashboard computes all five aggregates live on every request (AC9):
// income, expenses, remaining balance and by-category totals for the selected
// month, plus the five most recent transactions overall (D13).
func (s *Service) Dashboard(ctx context.Context, userID int64, yearRaw, monthRaw string) (DashboardView, *Error) {
	year, month, e := parseYearMonth(yearRaw, monthRaw)
	if e != nil {
		return DashboardView{}, e
	}

	totals, err := s.Store.MonthTotals(ctx, userID, year, month)
	if err != nil {
		return DashboardView{}, internal(s.Log, err)
	}
	income, err := money.FromDBText(totals.IncomeText)
	if err != nil {
		return DashboardView{}, internal(s.Log, err)
	}
	expense, err := money.FromDBText(totals.ExpenseText)
	if err != nil {
		return DashboardView{}, internal(s.Log, err)
	}

	cats, err := s.Store.ExpensesByCategory(ctx, userID, year, month)
	if err != nil {
		return DashboardView{}, internal(s.Log, err)
	}
	catViews := make([]CategoryTotalView, 0, len(cats))
	for _, c := range cats {
		total, err := money.FromDBText(c.TotalText)
		if err != nil {
			return DashboardView{}, internal(s.Log, err)
		}
		catViews = append(catViews, CategoryTotalView{
			CategoryID:   c.CategoryID,
			CategoryName: c.Name,
			Total:        money.Format(total),
		})
	}

	recent, err := s.Store.RecentTransactions(ctx, userID, 5)
	if err != nil {
		return DashboardView{}, internal(s.Log, err)
	}
	recentViews := make([]TxView, 0, len(recent))
	for i := range recent {
		v, err := toTxView(&recent[i])
		if err != nil {
			return DashboardView{}, internal(s.Log, err)
		}
		recentViews = append(recentViews, v)
	}

	return DashboardView{
		Year:               year,
		Month:              month,
		TotalIncome:        money.Format(income),
		TotalExpense:       money.Format(expense),
		RemainingBalance:   money.Format(income.Sub(expense)),
		ExpensesByCategory: catViews,
		RecentTransactions: recentViews,
	}, nil
}
