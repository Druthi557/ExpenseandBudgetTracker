package tests

import (
	"regexp"
	"testing"
)

// moneyString matches the D14 wire format: an exact 2-decimal-place string.
var moneyString = regexp.MustCompile(`^-?\d+\.\d{2}$`)

func assertMoneyString(t *testing.T, field string, v any) {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("money field %s = %T (%v), want a string", field, v, v)
	}
	if !moneyString.MatchString(s) {
		t.Fatalf("money field %s = %q, want exactly 2 decimal places", field, s)
	}
}

// TC-A22 — decimal exactness: 10.10 + 20.20 + 0.01 sums to exactly 30.31 via
// budget and dashboard endpoints; a large value survives intact; every money
// field in every response is a 2-dp string; POST→GET round-trips (AC13).
func TestTC_A22_MoneyExactness(t *testing.T) {
	c := newUser(t, "a22")
	food := createCategory(t, c, "Food", "EXPENSE")
	createBudget(t, c, food, 2026, 11, "1000.00")
	createBudget(t, c, food, 2026, 12, "99999999.99")

	// POST → GET round-trip for the odd cent value.
	r := c.post("/api/v1/transactions", map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "0.01", "date": "2026-11-15",
	})
	mustStatus(t, r, 201)
	txID := asID(r.decodeObject(t)["id"])
	if got := c.get("/api/v1/transactions/" + itoa(txID)).decodeObject(t)["amount"]; got != "0.01" {
		t.Fatalf("round-trip amount = %v, want \"0.01\"", got)
	}

	createTransaction(t, c, "EXPENSE", food, "10.10", "2026-11-01", "")
	createTransaction(t, c, "EXPENSE", food, "20.20", "2026-11-10", "")
	createTransaction(t, c, "EXPENSE", food, "99999999.99", "2026-12-05", "")

	// Budget actual for November: exactly 30.31 (no 30.309999… artifacts).
	r = c.get("/api/v1/budgets?year=2026&month=11")
	mustStatus(t, r, 200)
	nov := r.decodeArray(t)[0]
	if nov["actual_amount"] != "30.31" {
		t.Fatalf("budget actual = %v, want exactly \"30.31\"", nov["actual_amount"])
	}
	if nov["remaining"] != "969.69" {
		t.Fatalf("budget remaining = %v, want \"969.69\"", nov["remaining"])
	}

	// Budget actual for December: the large value intact.
	dec := c.get("/api/v1/budgets?year=2026&month=12").decodeArray(t)[0]
	if dec["actual_amount"] != "99999999.99" {
		t.Fatalf("large actual = %v, want \"99999999.99\"", dec["actual_amount"])
	}

	// Dashboard December total: large value intact, exact string.
	r = c.get("/api/v1/dashboard?year=2026&month=12")
	mustStatus(t, r, 200)
	dash := r.decodeObject(t)
	if dash["total_expense"] != "99999999.99" {
		t.Fatalf("dashboard total_expense = %v, want \"99999999.99\"", dash["total_expense"])
	}

	// Every money field in every response on this page is a 2-dp string.
	for _, key := range []string{"total_income", "total_expense", "remaining_balance"} {
		assertMoneyString(t, key, dash[key])
	}
	for _, ct := range dash["expenses_by_category"].([]any) {
		assertMoneyString(t, "expenses_by_category.total", ct.(map[string]any)["total"])
	}
	for _, tx := range dash["recent_transactions"].([]any) {
		assertMoneyString(t, "recent_transactions.amount", tx.(map[string]any)["amount"])
	}
}

// TC-A21 — dashboard aggregates for a seeded month: exact totals, by-category
// sums, five most recent overall (D13), other-month tx excluded (FR5, AC9).
func TestTC_A21_Dashboard(t *testing.T) {
	c := newUser(t, "a21")
	salary := createCategory(t, c, "Salary", "INCOME")
	food := createCategory(t, c, "Food", "EXPENSE")
	travel := createCategory(t, c, "Travel", "EXPENSE")

	createTransaction(t, c, "INCOME", salary, "3000.00", "2026-11-02", "")
	inc2ID := createTransaction(t, c, "INCOME", salary, "2000.00", "2026-11-20", "")
	foodNovID := createTransaction(t, c, "EXPENSE", food, "400.00", "2026-11-05", "")
	travel1ID := createTransaction(t, c, "EXPENSE", travel, "160.00", "2026-11-15", "")
	food2ID := createTransaction(t, c, "EXPENSE", food, "25.50", "2026-11-21", "")
	travel2ID := createTransaction(t, c, "EXPENSE", travel, "74.50", "2026-11-28", "")
	otherMonthID := createTransaction(t, c, "EXPENSE", food, "999.00", "2026-12-05", "")

	r := c.get("/api/v1/dashboard?year=2026&month=11")
	mustStatus(t, r, 200)
	d := r.decodeObject(t)

	if d["total_income"] != "5000.00" {
		t.Fatalf("total_income = %v, want \"5000.00\"", d["total_income"])
	}
	if d["total_expense"] != "660.00" {
		t.Fatalf("total_expense = %v, want \"660.00\"", d["total_expense"])
	}
	if d["remaining_balance"] != "4340.00" {
		t.Fatalf("remaining_balance = %v, want \"4340.00\"", d["remaining_balance"])
	}

	// By-category totals: Food 425.50, Travel 234.50 (selected month only).
	byCat := map[string]string{}
	for _, item := range d["expenses_by_category"].([]any) {
		ct := item.(map[string]any)
		byCat[ct["category_name"].(string)] = ct["total"].(string)
	}
	if byCat["Food"] != "425.50" || byCat["Travel"] != "234.50" {
		t.Fatalf("expenses_by_category = %v", byCat)
	}

	// Recent 5: newest first across ALL months (overall, D13).
	recent := d["recent_transactions"].([]any)
	if len(recent) != 5 {
		t.Fatalf("recent_transactions length = %d, want 5", len(recent))
	}
	wantOrder := []int64{otherMonthID, travel2ID, food2ID, inc2ID, travel1ID}
	for i, item := range recent {
		if got := asID(item.(map[string]any)["id"]); got != wantOrder[i] {
			t.Fatalf("recent[%d] id = %d, want %d (full list order wrong)", i, got, wantOrder[i])
		}
	}
	_ = foodNovID
}

// TC-A34 — a fresh user with no data gets zeros, not errors (FR5).
func TestTC_A34_EmptyDashboard(t *testing.T) {
	c := newUser(t, "a34")

	r := c.get("/api/v1/dashboard?year=2026&month=1")
	mustStatus(t, r, 200)
	d := r.decodeObject(t)

	if d["total_income"] != "0.00" || d["total_expense"] != "0.00" || d["remaining_balance"] != "0.00" {
		t.Fatalf("totals not zeroed: %v", d)
	}
	if len(d["expenses_by_category"].([]any)) != 0 {
		t.Fatalf("expenses_by_category should be empty, got %v", d["expenses_by_category"])
	}
	if len(d["recent_transactions"].([]any)) != 0 {
		t.Fatalf("recent_transactions should be empty, got %v", d["recent_transactions"])
	}
}
