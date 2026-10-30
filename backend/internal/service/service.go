// Package service is the business layer between HTTP handlers and the
// repository: input validation (AC14), money/date rules (AC4/AC5/AC13),
// type checks (AC6/AC7), database error mapping to the pinned HTTP policy
// (D28) and DTO assembly with exact 2-dp money strings (D14).
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"pebt/backend/internal/money"
	"pebt/backend/internal/store"
)

// Service implements the application's use cases.
type Service struct {
	Store *store.Store
	Log   *log.Logger
}

// New builds a Service; a nil logger falls back to the standard logger.
func New(st *store.Store, lg *log.Logger) *Service {
	if lg == nil {
		lg = log.Default()
	}
	return &Service{Store: st, Log: lg}
}

// Error is a domain error carrying its HTTP mapping (status + error code).
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func errf(status int, code, format string, a ...any) *Error {
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, a...)}
}

func notFound(what string) *Error {
	// Foreign resources are indistinguishable from missing ones (D16).
	return errf(404, "NOT_FOUND", "%s not found", what)
}

func internal(lg *log.Logger, err error) *Error {
	lg.Printf("internal error: %v", err)
	return errf(500, "INTERNAL", "Something went wrong. Please try again.")
}

// Text is a JSON string field that also accepts bare JSON scalars by keeping
// their raw text; type/semantic validation happens downstream so such inputs
// get a 422 rather than a 400.
type Text string

// UnmarshalJSON accepts a JSON string (any other JSON value is kept as raw
// text for validation; null leaves the field empty).
func (t *Text) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*t = ""
		return nil
	}
	if len(s) >= 2 && s[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*t = Text(v)
		return nil
	}
	*t = Text(s)
	return nil
}

// MoneyText parses a raw amount string into a validated decimal, mapping any
// problem to a 422 VALIDATION_ERROR with a field-specific message (AC4/AC13).
func (s *Service) MoneyText(raw string) (decimal.Decimal, *Error) {
	d, problem := money.Parse(raw)
	if problem != "" {
		return decimal.Decimal{}, errf(422, "VALIDATION_ERROR", "%s", problem)
	}
	return d, nil
}

// mapDBError converts PostgreSQL constraint violations into the pinned state-
// conflict responses; anything else becomes an anonymous 500.
func (s *Service) mapDBError(err error, inUseMessage string) *Error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505": // unique_violation
			switch pgErr.ConstraintName {
			case "users_email_lower_unique_idx":
				return errf(409, "EMAIL_TAKEN", "An account with this email already exists")
			case "categories_user_lower_name_unique_idx":
				return errf(409, "CATEGORY_DUPLICATE", "You already have a category with this name")
			case "budgets_user_cat_year_month_unique":
				return errf(409, "BUDGET_DUPLICATE", "A budget for this category, month and year already exists")
			}
		case pgErr.Code == "23503" || pgErr.Code == "23001": // foreign_key_violation / restrict_violation (ON DELETE RESTRICT)
			if inUseMessage != "" {
				return errf(409, "CATEGORY_IN_USE", "%s", inUseMessage)
			}
		}
	}
	return internal(s.Log, err)
}
