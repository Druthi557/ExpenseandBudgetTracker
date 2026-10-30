package tests

import "testing"

// TC-A15 — transaction CRUD round-trip with money echoed as 2-dp strings.
func TestTC_A15_TxCRUD(t *testing.T) {
	c := newUser(t, "a15")
	salary := createCategory(t, c, "Salary", "INCOME")
	food := createCategory(t, c, "Food", "EXPENSE")

	r := c.post("/api/v1/transactions", map[string]any{
		"type": "INCOME", "category_id": salary, "amount": "5000.5", "date": "2026-11-01",
		"description": "November salary",
	})
	mustStatus(t, r, 201)
	inc := r.decodeObject(t)
	if inc["amount"] != "5000.50" {
		t.Fatalf("amount = %v, want \"5000.50\"", inc["amount"])
	}
	if inc["date"] != "2026-11-01" {
		t.Fatalf("date = %v, want 2026-11-01", inc["date"])
	}
	incID := asID(inc["id"])

	r = c.post("/api/v1/transactions", map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "42", "date": "2026-11-02",
	})
	mustStatus(t, r, 201)
	exp := r.decodeObject(t)
	if exp["amount"] != "42.00" {
		t.Fatalf("amount = %v, want \"42.00\"", exp["amount"])
	}
	expID := asID(exp["id"])

	r = c.get("/api/v1/transactions/" + itoa(incID))
	mustStatus(t, r, 200)
	if got := r.decodeObject(t)["description"]; got != "November salary" {
		t.Fatalf("description round-trip failed: %v", got)
	}

	r = c.put("/api/v1/transactions/"+itoa(expID), map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "43.10", "date": "2026-11-03",
		"description": "updated",
	})
	mustStatus(t, r, 200)
	updated := r.decodeObject(t)
	if updated["amount"] != "43.10" || updated["date"] != "2026-11-03" || updated["description"] != "updated" {
		t.Fatalf("update failed: %v", updated)
	}

	r = c.del("/api/v1/transactions/" + itoa(expID))
	mustStatus(t, r, 204)
}

// TC-A16 — update validation: type/category mismatch 422; a consistent
// type+category switch 200; negative amount 422; impossible date 422.
func TestTC_A16_TxUpdateValidation(t *testing.T) {
	c := newUser(t, "a16")
	food := createCategory(t, c, "Food", "EXPENSE")
	salary := createCategory(t, c, "Salary", "INCOME")
	txID := createTransaction(t, c, "EXPENSE", food, "10.00", "2026-11-10", "")

	// INCOME keeping Food → 422 (AC6)
	r := c.put("/api/v1/transactions/"+itoa(txID), map[string]any{
		"type": "INCOME", "category_id": food, "amount": "10.00", "date": "2026-11-10",
	})
	mustStatus(t, r, 422)

	// switching both to a consistent pair → 200
	r = c.put("/api/v1/transactions/"+itoa(txID), map[string]any{
		"type": "INCOME", "category_id": salary, "amount": "10.00", "date": "2026-11-10",
	})
	mustStatus(t, r, 200)

	// amount -1 → 422 (AC4)
	r = c.put("/api/v1/transactions/"+itoa(txID), map[string]any{
		"type": "INCOME", "category_id": salary, "amount": "-1", "date": "2026-11-10",
	})
	mustStatus(t, r, 422)

	// date 2026-02-30 → 422 (AC5)
	r = c.put("/api/v1/transactions/"+itoa(txID), map[string]any{
		"type": "INCOME", "category_id": salary, "amount": "10.00", "date": "2026-02-30",
	})
	mustStatus(t, r, 422)
}

// TC-A07 — the invalid-amount matrix for transactions and budgets: 0, -5,
// 10.999 (3 dp) and "abc" all 422, never rounded, nothing stored (AC4/AC13).
func TestTC_A07_AmountValidation(t *testing.T) {
	c := newUser(t, "a07")
	food := createCategory(t, c, "Food", "EXPENSE")

	for _, amount := range []string{"0", "-5", "10.999", "abc"} {
		r := c.post("/api/v1/transactions", map[string]any{
			"type": "EXPENSE", "category_id": food, "amount": amount, "date": "2026-11-01",
		})
		mustStatus(t, r, 422)
		if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
			t.Fatalf("tx amount %q: code = %q, want VALIDATION_ERROR", amount, code)
		}
	}
	// nothing was stored
	rows := c.get("/api/v1/transactions").decodeArray(t)
	if len(rows) != 0 {
		t.Fatalf("invalid amounts were stored: %v", rows)
	}

	for _, amount := range []string{"0", "-5", "10.999"} {
		r := c.post("/api/v1/budgets", map[string]any{
			"category_id": food, "year": 2026, "month": 11, "amount": amount,
		})
		mustStatus(t, r, 422)
		if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
			t.Fatalf("budget amount %q: code = %q, want VALIDATION_ERROR", amount, code)
		}
	}
	budgets := c.get("/api/v1/budgets?year=2026&month=11").decodeArray(t)
	if len(budgets) != 0 {
		t.Fatalf("invalid budget amounts were stored: %v", budgets)
	}
}

// TC-A08 — invalid dates rejected on create (AC5).
func TestTC_A08_DateValidation(t *testing.T) {
	c := newUser(t, "a08")
	food := createCategory(t, c, "Food", "EXPENSE")

	for _, date := range []string{"2026-02-30", "not-a-date"} {
		r := c.post("/api/v1/transactions", map[string]any{
			"type": "EXPENSE", "category_id": food, "amount": "5.00", "date": date,
		})
		mustStatus(t, r, 422)
		if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
			t.Fatalf("date %q: code = %q, want VALIDATION_ERROR", date, code)
		}
	}
}

// TC-A09 — expense transaction on an income category: 422 TYPE_CONFLICT, no
// row created (AC6).
func TestTC_A09_TypeConflict(t *testing.T) {
	c := newUser(t, "a09")
	salary := createCategory(t, c, "Salary", "INCOME")

	r := c.post("/api/v1/transactions", map[string]any{
		"type": "EXPENSE", "category_id": salary, "amount": "5.00", "date": "2026-11-01",
	})
	mustStatus(t, r, 422)
	if code, _ := r.errorOf(t); code != "TYPE_CONFLICT" {
		t.Fatalf("code = %q, want TYPE_CONFLICT", code)
	}
	if rows := c.get("/api/v1/transactions").decodeArray(t); len(rows) != 0 {
		t.Fatalf("conflicting transaction was stored: %v", rows)
	}
}

// TC-A14 — list ordering (tx_date DESC, id DESC) and every filter (FR3.4/3.5).
func TestTC_A14_ListOrderFilters(t *testing.T) {
	c := newUser(t, "a14")
	salary := createCategory(t, c, "Salary", "INCOME")
	food := createCategory(t, c, "Food", "EXPENSE")
	travel := createCategory(t, c, "Travel", "EXPENSE")

	salaryID := createTransaction(t, c, "INCOME", salary, "3000.00", "2026-01-05", "")
	foodID := createTransaction(t, c, "EXPENSE", food, "40.00", "2026-03-10", "")
	travelID := createTransaction(t, c, "EXPENSE", travel, "18.00", "2026-03-10", "") // same date, later id

	// Default order: newest first, same-date tie-break by id DESC.
	rows := c.get("/api/v1/transactions").decodeArray(t)
	if len(rows) != 3 {
		t.Fatalf("list length = %d, want 3", len(rows))
	}
	got := []int64{asID(rows[0]["id"]), asID(rows[1]["id"]), asID(rows[2]["id"])}
	want := []int64{travelID, foodID, salaryID}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v (newest-first, id DESC tie-break)", got, want)
		}
	}

	// ?type=
	rows = c.get("/api/v1/transactions?type=INCOME").decodeArray(t)
	if len(rows) != 1 || asID(rows[0]["id"]) != salaryID {
		t.Fatalf("type filter wrong: %v", rows)
	}

	// ?category_id=
	rows = c.get("/api/v1/transactions?category_id=" + itoa(food)).decodeArray(t)
	if len(rows) != 1 || asID(rows[0]["id"]) != foodID {
		t.Fatalf("category filter wrong: %v", rows)
	}

	// ?from=&to= (inclusive range)
	rows = c.get("/api/v1/transactions?from=2026-03-01&to=2026-03-31").decodeArray(t)
	if len(rows) != 2 {
		t.Fatalf("date-range filter returned %d rows, want 2", len(rows))
	}
}

// TC-A28 — filter parameter validation: bad from → 422; from after to → 422;
// foreign category_id → 404 (AC14, D16).
func TestTC_A28_FilterValidation(t *testing.T) {
	c := newUser(t, "a28")
	other := newUser(t, "a28b")
	food := createCategory(t, c, "Food", "EXPENSE")
	_ = food

	r := c.get("/api/v1/transactions?from=bad")
	mustStatus(t, r, 422)

	r = c.get("/api/v1/transactions?from=2026-11-01&to=2026-10-01")
	mustStatus(t, r, 422)

	foreignCat := createCategory(t, other, "Foreign", "EXPENSE")
	r = c.get("/api/v1/transactions?category_id=" + itoa(foreignCat))
	mustStatus(t, r, 404)
}
