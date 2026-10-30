package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"pebt/backend/internal/money"
	"pebt/backend/internal/store"
)

// TxInput is the JSON body for transaction create/update (full replace).
type TxInput struct {
	Type        Text  `json:"type"`
	CategoryID  int64 `json:"category_id"`
	Amount      Text  `json:"amount"`
	Date        Text  `json:"date"`
	Description Text  `json:"description"`
}

// TxView is the transaction DTO: money as exact 2-dp strings (D14),
// date as ISO YYYY-MM-DD.
type TxView struct {
	ID           int64     `json:"id"`
	Type         string    `json:"type"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Amount       string    `json:"amount"`
	Date         string    `json:"date"`
	Description  *string   `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toTxView(tx *store.Transaction) (TxView, error) {
	amount, err := money.FromDBText(tx.Amount)
	if err != nil {
		return TxView{}, err
	}
	return TxView{
		ID:           tx.ID,
		Type:         tx.Type,
		CategoryID:   tx.CategoryID,
		CategoryName: tx.CategoryName,
		Amount:       money.Format(amount),
		Date:         tx.Date.Format("2006-01-02"),
		Description:  tx.Description,
		CreatedAt:    tx.CreatedAt,
		UpdatedAt:    tx.UpdatedAt,
	}, nil
}

// ListTransactions implements FR3.4/FR3.5: newest-first list with optional
// type, category and inclusive date-range filters.
func (s *Service) ListTransactions(ctx context.Context, userID int64,
	typeRaw, categoryIDRaw, fromRaw, toRaw string) ([]TxView, *Error) {

	filter := store.TxFilter{}
	if typeRaw != "" {
		typ, e := validateCategoryType(Text(typeRaw))
		if e != nil {
			return nil, e
		}
		filter.Type = &typ
	}
	if strings.TrimSpace(categoryIDRaw) != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(categoryIDRaw), 10, 64)
		if err != nil || id <= 0 {
			return nil, errf(422, "VALIDATION_ERROR", "category_id must be a positive integer")
		}
		if e := s.requireCategory(ctx, userID, id, ""); e != nil {
			return nil, e
		}
		filter.CategoryID = &id
	}
	if strings.TrimSpace(fromRaw) != "" {
		t, e := validateDate(Text(fromRaw))
		if e != nil {
			return nil, errf(422, "VALIDATION_ERROR", "from must be a valid date in YYYY-MM-DD format")
		}
		filter.From = &t
	}
	if strings.TrimSpace(toRaw) != "" {
		t, e := validateDate(Text(toRaw))
		if e != nil {
			return nil, errf(422, "VALIDATION_ERROR", "to must be a valid date in YYYY-MM-DD format")
		}
		filter.To = &t
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, errf(422, "VALIDATION_ERROR", "from must not be after to")
	}

	txs, err := s.Store.ListTransactions(ctx, userID, filter)
	if err != nil {
		return nil, internal(s.Log, err)
	}
	out := make([]TxView, 0, len(txs))
	for i := range txs {
		v, err := toTxView(&txs[i])
		if err != nil {
			return nil, internal(s.Log, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// GetTransaction returns one of the user's transactions (FR3.3).
func (s *Service) GetTransaction(ctx context.Context, userID int64, idRaw string) (TxView, *Error) {
	id, e := parseID(idRaw)
	if e != nil {
		return TxView{}, e
	}
	tx, err := s.Store.GetTransaction(ctx, userID, id)
	if err != nil {
		return TxView{}, internal(s.Log, err)
	}
	if tx == nil {
		return TxView{}, notFound("Transaction")
	}
	view, err := toTxView(tx)
	if err != nil {
		return TxView{}, internal(s.Log, err)
	}
	return view, nil
}

// CreateTransaction validates every field (AC4/AC5/AC6/AC14), verifies the
// category is owned (404) and of matching type (422 TYPE_CONFLICT), then
// stores the transaction.
func (s *Service) CreateTransaction(ctx context.Context, userID int64, in TxInput) (TxView, *Error) {
	fields, e := s.validateTxFields(ctx, userID, in)
	if e != nil {
		return TxView{}, e
	}
	id, err := s.Store.CreateTransaction(ctx, userID, fields.typ, fields.categoryID,
		fields.amount, fields.date, fields.description)
	if err != nil {
		return TxView{}, s.mapDBError(err, "")
	}
	return s.GetTransaction(ctx, userID, strconv.FormatInt(id, 10))
}

// UpdateTransaction fully replaces a transaction (FR3.3): every field is
// re-validated, exactly like create.
func (s *Service) UpdateTransaction(ctx context.Context, userID int64, idRaw string, in TxInput) (TxView, *Error) {
	id, e := parseID(idRaw)
	if e != nil {
		return TxView{}, e
	}
	fields, e := s.validateTxFields(ctx, userID, in)
	if e != nil {
		return TxView{}, e
	}
	ok, err := s.Store.UpdateTransaction(ctx, userID, id, fields.typ, fields.categoryID,
		fields.amount, fields.date, fields.description)
	if err != nil {
		return TxView{}, s.mapDBError(err, "")
	}
	if !ok {
		return TxView{}, notFound("Transaction")
	}
	return s.GetTransaction(ctx, userID, idRaw)
}

// DeleteTransaction removes one of the user's transactions (FR3.3).
func (s *Service) DeleteTransaction(ctx context.Context, userID int64, idRaw string) *Error {
	id, e := parseID(idRaw)
	if e != nil {
		return e
	}
	ok, err := s.Store.DeleteTransaction(ctx, userID, id)
	if err != nil {
		return internal(s.Log, err)
	}
	if !ok {
		return notFound("Transaction")
	}
	return nil
}

type txFields struct {
	typ         string
	categoryID  int64
	amount      string
	date        time.Time
	description *string
}

// validateTxFields applies the shared create/update rules and resolves the
// category, enforcing ownership (404) and type match (422).
func (s *Service) validateTxFields(ctx context.Context, userID int64, in TxInput) (txFields, *Error) {
	typ, e := validateCategoryType(in.Type)
	if e != nil {
		return txFields{}, errf(422, "VALIDATION_ERROR", "Transaction type must be INCOME or EXPENSE")
	}
	if in.CategoryID <= 0 {
		return txFields{}, errf(422, "VALIDATION_ERROR", "Category is required")
	}
	amount, e := s.MoneyText(string(in.Amount))
	if e != nil {
		return txFields{}, e
	}
	date, e := validateDate(in.Date)
	if e != nil {
		return txFields{}, e
	}
	description, e := validateDescription(in.Description)
	if e != nil {
		return txFields{}, e
	}
	if e := s.requireCategory(ctx, userID, in.CategoryID, typ); e != nil {
		return txFields{}, e
	}
	return txFields{
		typ:         typ,
		categoryID:  in.CategoryID,
		amount:      amount.StringFixed(2),
		date:        date,
		description: description,
	}, nil
}

// requireCategory loads a category owned by the user and, when expectedType
// is non-empty, checks the type matches (AC6/AC7 semantics at the API layer;
// the composite FK enforces the same at the database).
func (s *Service) requireCategory(ctx context.Context, userID, categoryID int64, expectedType string) *Error {
	cat, err := s.Store.GetCategory(ctx, userID, categoryID)
	if err != nil {
		return internal(s.Log, err)
	}
	if cat == nil {
		return notFound("Category")
	}
	if expectedType != "" && cat.Type != expectedType {
		return errf(422, "TYPE_CONFLICT",
			"Transaction type %s does not match the category type %s", expectedType, cat.Type)
	}
	return nil
}
