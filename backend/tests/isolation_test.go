package tests

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TC-A05 — B cannot read A's transaction: 404, no data leaked (AC2, D16).
func TestTC_A05_CrossUserRead(t *testing.T) {
	a := newUser(t, "a05a")
	b := newUser(t, "a05b")

	food := createCategory(t, a, "Food", "EXPENSE")
	txID := createTransaction(t, a, "EXPENSE", food, "12.34", "2026-11-05", "secret lunch")

	r := b.get("/api/v1/transactions/" + itoa(txID))
	mustStatus(t, r, 404)
	if code, _ := r.errorOf(t); code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", code)
	}
	if contains(string(r.body), "secret lunch") {
		t.Fatal("A's data leaked to B")
	}
}

// TC-A17 — B cannot update or delete A's transaction; A's data unchanged.
func TestTC_A17_CrossUserWriteDelete(t *testing.T) {
	a := newUser(t, "a17a")
	b := newUser(t, "a17b")

	food := createCategory(t, a, "Food", "EXPENSE")
	txID := createTransaction(t, a, "EXPENSE", food, "12.34", "2026-11-05", "")

	r := b.put("/api/v1/transactions/"+itoa(txID), map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "99.99", "date": "2026-11-05",
	})
	mustStatus(t, r, 404)

	r = b.del("/api/v1/transactions/" + itoa(txID))
	mustStatus(t, r, 404)

	r = a.get("/api/v1/transactions/" + itoa(txID))
	mustStatus(t, r, 200)
	if got := r.decodeObject(t)["amount"]; got != "12.34" {
		t.Fatalf("A's transaction changed: %v", got)
	}
}

// TC-A33 — list endpoints only ever return the caller's rows (AC2, FR1.5).
func TestTC_A33_ListIsolation(t *testing.T) {
	a := newUser(t, "a33a")
	b := newUser(t, "a33b")

	aFood := createCategory(t, a, "Food", "EXPENSE")
	aSalary := createCategory(t, a, "Salary", "INCOME")
	createTransaction(t, a, "EXPENSE", aFood, "10.00", "2026-11-01", "")
	createTransaction(t, a, "INCOME", aSalary, "100.00", "2026-11-02", "")
	createBudget(t, a, aFood, 2026, 11, "50.00")

	bTravel := createCategory(t, b, "Travel", "EXPENSE")
	createTransaction(t, b, "EXPENSE", bTravel, "20.00", "2026-11-03", "")
	createBudget(t, b, bTravel, 2026, 11, "80.00")

	aNames := setOf(a.get("/api/v1/categories").decodeArray(t), "name")
	if aNames["Travel"] || !aNames["Food"] {
		t.Fatalf("A sees wrong categories: %v", aNames)
	}
	bNames := setOf(b.get("/api/v1/categories").decodeArray(t), "name")
	if bNames["Food"] || bNames["Salary"] || !bNames["Travel"] {
		t.Fatalf("B sees wrong categories: %v", bNames)
	}

	aTx := a.get("/api/v1/transactions").decodeArray(t)
	if len(aTx) != 2 {
		t.Fatalf("A transaction list length = %d, want 2", len(aTx))
	}
	bTx := b.get("/api/v1/transactions").decodeArray(t)
	if len(bTx) != 1 {
		t.Fatalf("B transaction list length = %d, want 1", len(bTx))
	}

	aBudgets := a.get("/api/v1/budgets?year=2026&month=11").decodeArray(t)
	if len(aBudgets) != 1 || aBudgets[0]["category_name"] != "Food" {
		t.Fatalf("A budgets wrong: %v", aBudgets)
	}
	bBudgets := b.get("/api/v1/budgets?year=2026&month=11").decodeArray(t)
	if len(bBudgets) != 1 || bBudgets[0]["category_name"] != "Travel" {
		t.Fatalf("B budgets wrong: %v", bBudgets)
	}
}

// TC-A23 — database-level constraint proofs, bypassing the API entirely:
// PostgreSQL must reject cross-user references, income budgets, duplicate
// budgets and referenced-category deletes (AC2/AC6/AC7/AC8/AC12).
func TestTC_A23_DBConstraints(t *testing.T) {
	ctx := context.Background()
	a := newUser(t, "a23a")
	b := newUser(t, "a23b")

	aID := userIDForEmail(t, a.email)
	bExpense := createCategory(t, b, "BExpense", "EXPENSE")
	aIncome := createCategory(t, a, "AIncome", "INCOME")

	// (a) A's transaction referencing B's category → 23503
	_, err := pool.Exec(ctx,
		`INSERT INTO transactions (user_id, type, category_id, amount, tx_date, updated_at)
		 VALUES ($1, 'EXPENSE', $2, '1.00', '2026-11-01', now())`,
		aID, bExpense)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-user tx insert: got %v, want 23503", err)
	}

	// (b) budget on an income category → 23503 (cat_type carrier pins EXPENSE)
	_, err = pool.Exec(ctx,
		`INSERT INTO budgets (user_id, category_id, year, month, amount, updated_at)
		 VALUES ($1, $2, 2026, 11, '1.00', now())`,
		aID, aIncome)
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("income budget insert: got %v, want 23503", err)
	}

	// (c) second identical budget row → 23505
	aExpense := createCategory(t, a, "AExpense", "EXPENSE")
	createBudget(t, a, aExpense, 2026, 12, "5.00")
	_, err = pool.Exec(ctx,
		`INSERT INTO budgets (user_id, category_id, year, month, amount, updated_at)
		 VALUES ($1, $2, 2026, 12, '9.00', now())`,
		aID, aExpense)
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("duplicate budget insert: got %v, want 23505", err)
	}

	// (d) DELETE a category referenced by a transaction → 23001 (restrict_violation)
	createTransaction(t, a, "EXPENSE", aExpense, "2.00", "2026-12-01", "")
	_, err = pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, aExpense)
	if !errors.As(err, &pgErr) || pgErr.Code != "23001" {
		t.Fatalf("referenced category delete: got %v, want 23001", err)
	}
}

// ---------- tiny assertion helpers ----------

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func setOf(rows []map[string]any, key string) map[string]bool {
	out := map[string]bool{}
	for _, r := range rows {
		if v, ok := r[key].(string); ok {
			out[v] = true
		}
	}
	return out
}
