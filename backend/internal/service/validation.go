package service

import (
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// validateEmail enforces the email format rule and returns the normalized
// (trimmed, lowercased) address.
func validateEmail(raw Text) (string, *Error) {
	email := strings.ToLower(strings.TrimSpace(string(raw)))
	if email == "" {
		return "", errf(422, "VALIDATION_ERROR", "Email is required")
	}
	if len(email) > 255 {
		return "", errf(422, "VALIDATION_ERROR", "Email must be at most 255 characters")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, "@") {
		return "", errf(422, "VALIDATION_ERROR", "Email must be a valid email address")
	}
	return email, nil
}

// validatePassword enforces the D8 policy: 8–72 bytes, ≥1 letter, ≥1 digit.
func validatePassword(raw Text) (string, *Error) {
	pw := string(raw)
	if len(pw) < 8 {
		return "", errf(422, "VALIDATION_ERROR", "Password must be at least 8 characters")
	}
	if len(pw) > 72 {
		return "", errf(422, "VALIDATION_ERROR", "Password must be at most 72 characters")
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "", errf(422, "VALIDATION_ERROR", "Password must contain at least one letter and one digit")
	}
	return pw, nil
}

// validateCategoryName trims and bounds-checks a category name (1–100 chars).
func validateCategoryName(raw Text) (string, *Error) {
	name := strings.TrimSpace(string(raw))
	if name == "" {
		return "", errf(422, "VALIDATION_ERROR", "Category name is required")
	}
	if len([]rune(name)) > 100 {
		return "", errf(422, "VALIDATION_ERROR", "Category name must be at most 100 characters")
	}
	return name, nil
}

// validateCategoryType checks the INCOME/EXPENSE enum.
func validateCategoryType(raw Text) (string, *Error) {
	typ := string(raw)
	if typ != "INCOME" && typ != "EXPENSE" {
		return "", errf(422, "VALIDATION_ERROR", "Type must be INCOME or EXPENSE")
	}
	return typ, nil
}

// validateDate parses a strict ISO calendar date (AC5). PostgreSQL DATE
// semantics: no time-of-day, no timezone; 2026-02-30 is invalid.
func validateDate(raw Text) (time.Time, *Error) {
	s := strings.TrimSpace(string(raw))
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, errf(422, "VALIDATION_ERROR", "Date must be a valid calendar date in YYYY-MM-DD format")
	}
	return t, nil
}

// validateDescription trims an optional description (≤500 chars) and maps
// empty to NULL.
func validateDescription(raw Text) (*string, *Error) {
	desc := strings.TrimSpace(string(raw))
	if desc == "" {
		return nil, nil
	}
	if len([]rune(desc)) > 500 {
		return nil, errf(422, "VALIDATION_ERROR", "Description must be at most 500 characters")
	}
	return &desc, nil
}

// parseID converts a path id; malformed or non-positive ids are treated as
// missing resources (404, existence masking D16).
func parseID(raw string) (int64, *Error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, notFound("Resource")
	}
	return id, nil
}

// parseYearMonth validates required year/month query values (budget list and
// dashboard).
func parseYearMonth(yearRaw, monthRaw string) (int, int, *Error) {
	year, e := parseYear(yearRaw, "year")
	if e != nil {
		return 0, 0, e
	}
	month, e := parseMonth(monthRaw, "month")
	if e != nil {
		return 0, 0, e
	}
	return year, month, nil
}

func parseYear(raw, field string) (int, *Error) {
	if strings.TrimSpace(raw) == "" {
		return 0, errf(422, "VALIDATION_ERROR", "%s is required", field)
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 2000 || v > 2100 {
		return 0, errf(422, "VALIDATION_ERROR", "%s must be a year between 2000 and 2100", field)
	}
	return v, nil
}

func parseMonth(raw, field string) (int, *Error) {
	if strings.TrimSpace(raw) == "" {
		return 0, errf(422, "VALIDATION_ERROR", "%s is required", field)
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 1 || v > 12 {
		return 0, errf(422, "VALIDATION_ERROR", "%s must be a month between 1 and 12", field)
	}
	return v, nil
}

// parseYearValue/parseMonthValue validate year/month taken from JSON bodies.
func parseYearValue(v int) (int, *Error) {
	if v < 2000 || v > 2100 {
		return 0, errf(422, "VALIDATION_ERROR", "Year must be between 2000 and 2100")
	}
	return v, nil
}

func parseMonthValue(v int) (int, *Error) {
	if v < 1 || v > 12 {
		return 0, errf(422, "VALIDATION_ERROR", "Month must be between 1 and 12")
	}
	return v, nil
}
