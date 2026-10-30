// Package money implements the exact-decimal money rules (AC13, D14/D26):
// strings on the wire, at most 2 decimal places, never floating-point.
package money

import (
	"strings"

	"github.com/shopspring/decimal"
)

// maxAmount matches the NUMERIC(12,2) column: values up to 9,999,999,999.99.
var maxAmount = decimal.NewFromInt(10_000_000_000)

// Parse validates a raw money string and returns its exact decimal value.
// The returned string is empty on success, otherwise a human-readable
// validation message (mapped to a 422 response by callers).
func Parse(raw string) (decimal.Decimal, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return decimal.Decimal{}, "amount is required"
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, "amount must be a valid decimal number"
	}
	if d.Sign() <= 0 {
		return decimal.Decimal{}, "amount must be greater than zero"
	}
	if !d.Equal(d.Round(2)) {
		return decimal.Decimal{}, "amount must have at most 2 decimal places"
	}
	if d.Abs().Cmp(maxAmount) >= 0 {
		return decimal.Decimal{}, "amount is too large (maximum 9999999999.99)"
	}
	return d, ""
}

// Format renders a decimal in the wire format: a string with exactly 2 dp.
func Format(d decimal.Decimal) string {
	return d.StringFixed(2)
}

// FromDBText parses a NUMERIC(12,2) value received as text from PostgreSQL.
// Such values are always valid decimals; an error is unexpected and surfaced
// to the caller (mapped to 500) rather than silently coerced.
func FromDBText(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}
