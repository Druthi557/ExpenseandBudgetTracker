package service

import (
	"context"

	"pebt/backend/internal/money"
	"pebt/backend/internal/store"
)

// BudgetInput is the JSON body for budget create.
type BudgetInput struct {
	CategoryID int64 `json:"category_id"`
	Year       int   `json:"year"`
	Month      int   `json:"month"`
	Amount     Text  `json:"amount"`
}

// BudgetAmountInput is the JSON body for the amount-only budget edit (D38).
type BudgetAmountInput struct {
	Amount Text `json:"amount"`
}

// BudgetView is a budget row plus the computed month figures (FR4.5/FR4.6).
// All money values are exact 2-dp strings; Remaining may be negative (AC11).
type BudgetView struct {
	ID           int64  `json:"id"`
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	Amount       string `json:"amount"`
	ActualAmount string `json:"actual_amount"`
	Remaining    string `json:"remaining"`
	OverBudget   bool   `json:"over_budget"`
}

func toBudgetView(b store.BudgetRow) (BudgetView, error) {
	amount, err := money.FromDBText(b.Amount)
	if err != nil {
		return BudgetView{}, err
	}
	actual, err := money.FromDBText(b.Actual)
	if err != nil {
		return BudgetView{}, err
	}
	remaining := amount.Sub(actual)
	return BudgetView{
		ID:           b.ID,
		CategoryID:   b.CategoryID,
		CategoryName: b.CategoryName,
		Year:         b.Year,
		Month:        b.Month,
		Amount:       money.Format(amount),
		ActualAmount: money.Format(actual),
		Remaining:    money.Format(remaining),
		OverBudget:   actual.Cmp(amount) > 0,
	}, nil
}

// ListBudgets returns the user's budgets for one calendar month with the
// computed actual/remaining/over-budget figures (FR4.5, AC10, AC11).
func (s *Service) ListBudgets(ctx context.Context, userID int64, yearRaw, monthRaw string) ([]BudgetView, *Error) {
	year, month, e := parseYearMonth(yearRaw, monthRaw)
	if e != nil {
		return nil, e
	}
	rows, err := s.Store.ListBudgets(ctx, userID, year, month)
	if err != nil {
		return nil, internal(s.Log, err)
	}
	out := make([]BudgetView, 0, len(rows))
	for _, r := range rows {
		v, err := toBudgetView(r)
		if err != nil {
			return nil, internal(s.Log, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// CreateBudget validates and creates a monthly budget (FR4.1/FR4.2). Only
// expense categories are allowed (AC7); duplicates are rejected (AC8).
func (s *Service) CreateBudget(ctx context.Context, userID int64, in BudgetInput) (BudgetView, *Error) {
	if in.CategoryID <= 0 {
		return BudgetView{}, errf(422, "VALIDATION_ERROR", "Category is required")
	}
	if _, e := parseYearValue(in.Year); e != nil {
		return BudgetView{}, e
	}
	if _, e := parseMonthValue(in.Month); e != nil {
		return BudgetView{}, e
	}
	amount, e := s.MoneyText(string(in.Amount))
	if e != nil {
		return BudgetView{}, e
	}
	if e := s.requireCategory(ctx, userID, in.CategoryID, "EXPENSE"); e != nil {
		return BudgetView{}, e
	}
	id, err := s.Store.CreateBudget(ctx, userID, in.CategoryID, in.Year, in.Month, amount.StringFixed(2))
	if err != nil {
		return BudgetView{}, s.mapDBError(err, "")
	}
	return s.getBudgetView(ctx, userID, id, in.Year, in.Month)
}

// UpdateBudgetAmount edits only the amount (D38 — changing category/month/
// year is delete + recreate by design).
func (s *Service) UpdateBudgetAmount(ctx context.Context, userID int64, idRaw string, in BudgetAmountInput) (BudgetView, *Error) {
	id, e := parseID(idRaw)
	if e != nil {
		return BudgetView{}, e
	}
	amount, e := s.MoneyText(string(in.Amount))
	if e != nil {
		return BudgetView{}, e
	}
	meta, err := s.Store.GetBudget(ctx, userID, id)
	if err != nil {
		return BudgetView{}, internal(s.Log, err)
	}
	if meta == nil {
		return BudgetView{}, notFound("Budget")
	}
	ok, err := s.Store.UpdateBudgetAmount(ctx, userID, id, amount.StringFixed(2))
	if err != nil {
		return BudgetView{}, internal(s.Log, err)
	}
	if !ok {
		return BudgetView{}, notFound("Budget")
	}
	return s.getBudgetView(ctx, userID, id, meta.Year, meta.Month)
}

// DeleteBudget removes one of the user's budgets (FR4.4).
func (s *Service) DeleteBudget(ctx context.Context, userID int64, idRaw string) *Error {
	id, e := parseID(idRaw)
	if e != nil {
		return e
	}
	ok, err := s.Store.DeleteBudget(ctx, userID, id)
	if err != nil {
		return internal(s.Log, err)
	}
	if !ok {
		return notFound("Budget")
	}
	return nil
}

// getBudgetView rebuilds one budget view (with month figures) by listing the
// budget's month and picking the row.
func (s *Service) getBudgetView(ctx context.Context, userID, id int64, year, month int) (BudgetView, *Error) {
	rows, err := s.Store.ListBudgets(ctx, userID, year, month)
	if err != nil {
		return BudgetView{}, internal(s.Log, err)
	}
	for _, r := range rows {
		if r.ID == id {
			v, err := toBudgetView(r)
			if err != nil {
				return BudgetView{}, internal(s.Log, err)
			}
			return v, nil
		}
	}
	return BudgetView{}, notFound("Budget")
}
