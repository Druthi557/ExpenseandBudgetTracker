package tests

import "testing"

// TC-A13 — full category CRUD round-trip: 201 / 200 / 200 / 204 (FR2).
func TestTC_A13_CategoryCRUD(t *testing.T) {
	c := newUser(t, "a13")

	r := c.post("/api/v1/categories", map[string]any{"name": "Groceries", "type": "EXPENSE"})
	mustStatus(t, r, 201)
	created := r.decodeObject(t)
	if created["name"] != "Groceries" || created["type"] != "EXPENSE" {
		t.Fatalf("created category wrong: %v", created)
	}
	id := asID(created["id"])

	r = c.put("/api/v1/categories/"+itoa(id), map[string]any{"name": "Groceries & more", "type": "EXPENSE"})
	mustStatus(t, r, 200)
	if got := r.decodeObject(t)["name"]; got != "Groceries & more" {
		t.Fatalf("rename failed: %v", got)
	}

	r = c.get("/api/v1/categories")
	mustStatus(t, r, 200)
	if names := setOf(r.decodeArray(t), "name"); !names["Groceries & more"] {
		t.Fatalf("list missing renamed category")
	}

	r = c.del("/api/v1/categories/" + itoa(id))
	mustStatus(t, r, 204)
}

// TC-A26 — duplicate name, different case, same user: 409 CATEGORY_DUPLICATE
// (D18). AC14.
func TestTC_A26_CategoryDuplicate(t *testing.T) {
	c := newUser(t, "a26")
	createCategory(t, c, "Food", "EXPENSE")

	r := c.post("/api/v1/categories", map[string]any{"name": "FOOD", "type": "EXPENSE"})
	mustStatus(t, r, 409)
	if code, _ := r.errorOf(t); code != "CATEGORY_DUPLICATE" {
		t.Fatalf("code = %q, want CATEGORY_DUPLICATE", code)
	}
}

// TC-A27 — ?type= filter returns only matching rows; invalid value → 422.
func TestTC_A27_CategoryTypeFilter(t *testing.T) {
	c := newUser(t, "a27")
	createCategory(t, c, "Salary", "INCOME")
	createCategory(t, c, "Freelance", "INCOME")
	createCategory(t, c, "Food", "EXPENSE")
	createCategory(t, c, "Travel", "EXPENSE")

	r := c.get("/api/v1/categories?type=EXPENSE")
	mustStatus(t, r, 200)
	rows := r.decodeArray(t)
	if len(rows) != 2 {
		t.Fatalf("EXPENSE filter returned %d rows, want 2", len(rows))
	}
	for _, row := range rows {
		if row["type"] != "EXPENSE" {
			t.Fatalf("non-expense row in filtered list: %v", row)
		}
	}

	r = c.get("/api/v1/categories?type=INCOME")
	mustStatus(t, r, 200)
	if len(r.decodeArray(t)) != 2 {
		t.Fatalf("INCOME filter returned wrong count")
	}

	r = c.get("/api/v1/categories?type=Bogus")
	mustStatus(t, r, 422)
	if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want VALIDATION_ERROR", code)
	}
}

// TC-A06 — deleting a referenced category: 409 CATEGORY_IN_USE, category
// still present afterwards (AC12).
func TestTC_A06_CategoryInUse(t *testing.T) {
	c := newUser(t, "a06")
	food := createCategory(t, c, "Food", "EXPENSE")
	createTransaction(t, c, "EXPENSE", food, "5.00", "2026-11-01", "")

	r := c.del("/api/v1/categories/" + itoa(food))
	mustStatus(t, r, 409)
	if code, _ := r.errorOf(t); code != "CATEGORY_IN_USE" {
		t.Fatalf("code = %q, want CATEGORY_IN_USE", code)
	}

	r = c.get("/api/v1/categories")
	found := false
	for _, row := range r.decodeArray(t) {
		if asID(row["id"]) == food {
			found = true
		}
	}
	if !found {
		t.Fatal("referenced category was deleted despite RESTRICT")
	}
}
