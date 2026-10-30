package tests

import "testing"

// TC-A18 — budget create → amount-only edit (D38) → delete.
func TestTC_A18_BudgetCRUD(t *testing.T) {
	c := newUser(t, "a18")
	food := createCategory(t, c, "Food", "EXPENSE")

	r := c.post("/api/v1/budgets", map[string]any{
		"category_id": food, "year": 2026, "month": 11, "amount": "100.00",
	})
	mustStatus(t, r, 201)
	budget := r.decodeObject(t)
	id := asID(budget["id"])
	if budget["amount"] != "100.00" || budget["actual_amount"] != "0.00" ||
		budget["remaining"] != "100.00" || budget["over_budget"] != false {
		t.Fatalf("fresh budget view wrong: %v", budget)
	}

	r = c.put("/api/v1/budgets/"+itoa(id), map[string]any{"amount": "200.00"})
	mustStatus(t, r, 200)
	if got := r.decodeObject(t)["amount"]; got != "200.00" {
		t.Fatalf("budget edit failed: %v", got)
	}

	r = c.del("/api/v1/budgets/" + itoa(id))
	mustStatus(t, r, 204)
}

// TC-A19 — budget amount validation matrix: 0, -5, 10.999 all 422 (AC4).
func TestTC_A19_BudgetAmountValidation(t *testing.T) {
	c := newUser(t, "a19")
	food := createCategory(t, c, "Food", "EXPENSE")

	for _, amount := range []string{"0", "-5", "10.999"} {
		r := c.post("/api/v1/budgets", map[string]any{
			"category_id": food, "year": 2026, "month": 12, "amount": amount,
		})
		mustStatus(t, r, 422)
		if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
			t.Fatalf("amount %q: code = %q, want VALIDATION_ERROR", amount, code)
		}
	}
}

// TC-A10 — budget on an income category: 422, no row created (AC7).
func TestTC_A10_BudgetIncomeCategory(t *testing.T) {
	c := newUser(t, "a10")
	salary := createCategory(t, c, "Salary", "INCOME")

	r := c.post("/api/v1/budgets", map[string]any{
		"category_id": salary, "year": 2026, "month": 11, "amount": "100.00",
	})
	mustStatus(t, r, 422)
	if code, _ := r.errorOf(t); code != "TYPE_CONFLICT" {
		t.Fatalf("code = %q, want TYPE_CONFLICT", code)
	}
	if rows := c.get("/api/v1/budgets?year=2026&month=11").decodeArray(t); len(rows) != 0 {
		t.Fatalf("income budget was stored: %v", rows)
	}
}

// TC-A11 — duplicate budget (same category+month+year): 409 BUDGET_DUPLICATE (AC8).
func TestTC_A11_BudgetDuplicate(t *testing.T) {
	c := newUser(t, "a11")
	food := createCategory(t, c, "Food", "EXPENSE")
	createBudget(t, c, food, 2026, 11, "100.00")

	r := c.post("/api/v1/budgets", map[string]any{
		"category_id": food, "year": 2026, "month": 11, "amount": "150.00",
	})
	mustStatus(t, r, 409)
	if code, _ := r.errorOf(t); code != "BUDGET_DUPLICATE" {
		t.Fatalf("code = %q, want BUDGET_DUPLICATE", code)
	}
}

// TC-A12 — calendar-month actual (AC10) with a half-open range: December
// transactions excluded; live recomputation after update/delete (AC9).
func TestTC_A12_BudgetMonthCalculation(t *testing.T) {
	c := newUser(t, "a12")
	food := createCategory(t, c, "Food", "EXPENSE")
	createBudget(t, c, food, 2026, 11, "100.00")

	nov1 := createTransaction(t, c, "EXPENSE", food, "40.00", "2026-11-01", "")
	_ = createTransaction(t, c, "EXPENSE", food, "50.00", "2026-11-30", "")
	createTransaction(t, c, "EXPENSE", food, "500.00", "2026-12-01", "") // next month — must be excluded

	r := c.get("/api/v1/budgets?year=2026&month=11")
	mustStatus(t, r, 200)
	rows := r.decodeArray(t)
	if len(rows) != 1 {
		t.Fatalf("budget rows = %d, want 1", len(rows))
	}
	b := rows[0]
	if b["actual_amount"] != "90.00" {
		t.Fatalf("actual = %v, want \"90.00\" (Dec tx must be excluded)", b["actual_amount"])
	}
	if b["remaining"] != "10.00" {
		t.Fatalf("remaining = %v, want \"10.00\"", b["remaining"])
	}

	// Update a November tx 40.00 → 60.00: actual becomes 110.00 immediately.
	r = c.put("/api/v1/transactions/"+itoa(nov1), map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "60.00", "date": "2026-11-01",
	})
	mustStatus(t, r, 200)
	b = c.get("/api/v1/budgets?year=2026&month=11").decodeArray(t)[0]
	if b["actual_amount"] != "110.00" || b["remaining"] != "-10.00" || b["over_budget"] != true {
		t.Fatalf("after update: actual=%v remaining=%v over=%v", b["actual_amount"], b["remaining"], b["over_budget"])
	}

	// Delete it: actual drops to 50.00 immediately.
	mustStatus(t, c.del("/api/v1/transactions/"+itoa(nov1)), 204)
	b = c.get("/api/v1/budgets?year=2026&month=11").decodeArray(t)[0]
	if b["actual_amount"] != "50.00" || b["remaining"] != "50.00" {
		t.Fatalf("after delete: actual=%v remaining=%v", b["actual_amount"], b["remaining"])
	}
}

// TC-A20 — overspending: actual > budget → negative remaining and
// over_budget=true (AC11).
func TestTC_A20_OverBudget(t *testing.T) {
	c := newUser(t, "a20")
	food := createCategory(t, c, "Food", "EXPENSE")
	createBudget(t, c, food, 2026, 11, "100.00")
	createTransaction(t, c, "EXPENSE", food, "70.00", "2026-11-03", "")
	createTransaction(t, c, "EXPENSE", food, "80.00", "2026-11-07", "") // total 150.00

	r := c.get("/api/v1/budgets?year=2026&month=11")
	mustStatus(t, r, 200)
	b := r.decodeArray(t)[0]
	if b["actual_amount"] != "150.00" {
		t.Fatalf("actual = %v, want \"150.00\"", b["actual_amount"])
	}
	if b["remaining"] != "-50.00" {
		t.Fatalf("remaining = %v, want \"-50.00\"", b["remaining"])
	}
	if b["over_budget"] != true {
		t.Fatalf("over_budget = %v, want true", b["over_budget"])
	}
}
