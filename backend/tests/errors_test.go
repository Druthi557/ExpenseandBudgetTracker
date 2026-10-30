package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TC-A29 — the error envelope contract: one 409, one 404, one 422, unknown
// route 404 envelope, malformed JSON 400; never any internals (AC14, D17/D28).
func TestTC_A29_ErrorEnvelope(t *testing.T) {
	c := newUser(t, "a29")
	food := createCategory(t, c, "Food", "EXPENSE")

	// 409 conflict
	createCategory(t, c, "Dupe", "EXPENSE")
	r := c.post("/api/v1/categories", map[string]any{"name": "Dupe", "type": "EXPENSE"})
	mustStatus(t, r, 409)
	code, msg := r.errorOf(t)
	if code == "" || msg == "" {
		t.Fatalf("409 envelope incomplete: %s", r.body)
	}

	// 404 missing
	r = c.get("/api/v1/transactions/999999")
	mustStatus(t, r, 404)
	code, msg = r.errorOf(t)
	if code != "NOT_FOUND" || msg == "" {
		t.Fatalf("404 envelope wrong: code=%q msg=%q", code, msg)
	}

	// 422 validation
	r = c.post("/api/v1/transactions", map[string]any{
		"type": "EXPENSE", "category_id": food, "amount": "0", "date": "2026-11-01",
	})
	mustStatus(t, r, 422)
	code, _ = r.errorOf(t)
	if code != "VALIDATION_ERROR" {
		t.Fatalf("422 code = %q, want VALIDATION_ERROR", code)
	}

	// Unknown API route → 404 envelope
	r = c.get("/api/v1/nope")
	mustStatus(t, r, 404)
	code, _ = r.errorOf(t)
	if code != "NOT_FOUND" {
		t.Fatalf("unknown route code = %q, want NOT_FOUND", code)
	}

	// Malformed JSON body → 400
	r = c.do(t, "POST", "/api/v1/categories", []byte("{not json"), true)
	mustStatus(t, r, 400)
	code, _ = r.errorOf(t)
	if code != "BAD_REQUEST" {
		t.Fatalf("malformed JSON code = %q, want BAD_REQUEST", code)
	}

	// No stack traces or internals anywhere above.
	if strings.Contains(string(r.body), "goroutine") || strings.Contains(string(r.body), "SQLSTATE") ||
		strings.Contains(string(r.body), ".go:") {
		t.Fatalf("internals leaked in error body: %s", r.body)
	}
}

// TC-A24 — persistence (AC16): data created on one server instance survives
// its full shutdown (including its connection pool) and is fully visible
// through a freshly started instance — PostgreSQL is the sole persistent
// store, so a restart loses nothing. Subsequent tests continue against the
// restarted instance.
func TestTC_A24_Persistence(t *testing.T) {
	c := newUser(t, "a24")
	food := createCategory(t, c, "Food", "EXPENSE")
	txID := createTransaction(t, c, "EXPENSE", food, "7.77", "2026-11-09", "persisted")
	createBudget(t, c, food, 2026, 11, "60.00")

	// Full "restart": stop the API instance and drop its pool, then boot a
	// fresh instance with a fresh pool over the same database.
	ts.Close()
	pool.Close()
	newPool, err := pgxpool.New(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("restart pool: %v", err)
	}
	pool = newPool
	ts = newServer()
	c.base = ts.URL // the client now talks to the restarted instance

	r := c.get("/api/v1/categories")
	mustStatus(t, r, 200)
	if names := setOf(r.decodeArray(t), "name"); !names["Food"] {
		t.Fatal("category lost after restart")
	}

	r = c.get("/api/v1/transactions/" + itoa(txID))
	mustStatus(t, r, 200)
	tx := r.decodeObject(t)
	if tx["amount"] != "7.77" || tx["description"] != "persisted" {
		t.Fatalf("transaction changed across restart: %v", tx)
	}

	r = c.get("/api/v1/budgets?year=2026&month=11")
	mustStatus(t, r, 200)
	budgets := r.decodeArray(t)
	if len(budgets) != 1 || budgets[0]["amount"] != "60.00" || budgets[0]["actual_amount"] != "7.77" {
		t.Fatalf("budget wrong after restart: %v", budgets)
	}
}
