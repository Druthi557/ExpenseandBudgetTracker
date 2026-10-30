// Package tests runs the Stage A automated suite (test_cases.md, TC-A01–A35):
// httptest integration tests against a dedicated PostgreSQL test database
// built from the embedded migrations. Some cases operate at the database
// level (direct SQL) to prove the constraints independent of the API.
//
// Required environment: TEST_DATABASE_URL pointing at an empty dedicated
// database (the suite resets its schema at startup).
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"pebt/backend"
	"pebt/backend/internal/api"
	"pebt/backend/internal/config"
	"pebt/backend/internal/dbmigrate"
	"pebt/backend/internal/service"
	"pebt/backend/internal/store"
)

var (
	testDBURL  string
	jwtSecret  = []byte("test-secret-0123456789-0123456789-0123456789") // 42 bytes
	pool       *pgxpool.Pool
	ts         *httptest.Server
	testLogger = log.New(io.Discard, "", 0)
	testPasswd = "Passw0rd!x"
	emailSeq   atomic.Int64
)

func TestMain(m *testing.M) {
	testDBURL = os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL is not set — the Stage A suite needs a dedicated test database (see .env.example)")
		os.Exit(1)
	}

	fsys, err := fs.Sub(backend.MigrationsFS, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "embedded migrations: %v\n", err)
		os.Exit(1)
	}
	if err := dbmigrate.DropSchema(testDBURL); err != nil {
		fmt.Fprintf(os.Stderr, "resetting schema: %v\n", err)
		os.Exit(1)
	}
	if err := dbmigrate.Up(testDBURL, fsys); err != nil {
		fmt.Fprintf(os.Stderr, "applying migrations: %v\n", err)
		os.Exit(1)
	}

	pool, err = pgxpool.New(context.Background(), testDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "test pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	ts = newServer()
	code := m.Run()
	ts.Close()
	os.Exit(code)
}

// newServer builds a fresh API server over the shared pool (used by TC-A24's
// restart scenario too).
func newServer() *httptest.Server {
	cfg := &config.Config{
		DatabaseURL:  testDBURL,
		JWTSecret:    jwtSecret,
		CookieSecure: false, // httptest serves plain HTTP
		Port:         "0",
	}
	srv := api.New(service.New(store.New(pool), testLogger), cfg, testLogger)
	return httptest.NewServer(srv.NewRouter())
}

// ---------- client helpers ----------

// apiClient is a minimal API client carrying one session cookie (mirroring a
// browser: the cookie value is dropped when the server clears it).
type apiClient struct {
	t     *testing.T
	base  string
	token string
	email string
}

// newClient returns an unbound client; bind it with bound(t) before calls.
func newClient(base string) *apiClient {
	if base == "" {
		base = ts.URL
	}
	return &apiClient{base: base}
}

func (c *apiClient) bound(t *testing.T) *apiClient {
	cc := *c
	cc.t = t
	return &cc
}

type response struct {
	status  int
	header  http.Header
	body    []byte
	cookies []*http.Cookie
}

func (c *apiClient) do(t *testing.T, method, path string, body any, withCSRF bool) *response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		var b []byte
		if raw, ok := body.([]byte); ok {
			b = raw // pre-encoded body (e.g. malformed JSON tests)
		} else {
			var err error
			b, err = json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
		}
		reader = bytes.NewReader(b)
	}
	httpReq, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if withCSRF {
		httpReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	if c.token != "" {
		httpReq.AddCookie(&http.Cookie{Name: "pebt_token", Value: c.token})
	}
	res, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return &response{status: res.StatusCode, header: res.Header, body: raw, cookies: res.Cookies()}
}

// Verb wrappers carry the CSRF header on every request by default, like the
// frontend fetch wrapper; individual tests drop it explicitly where needed.
func (c *apiClient) get(path string) *response            { return c.do(c.t, "GET", path, nil, true) }
func (c *apiClient) post(path string, body any) *response { return c.do(c.t, "POST", path, body, true) }
func (c *apiClient) put(path string, body any) *response  { return c.do(c.t, "PUT", path, body, true) }
func (c *apiClient) del(path string) *response            { return c.do(c.t, "DELETE", path, nil, true) }
func (c *apiClient) postNoCSRF(path string, body any) *response {
	return c.do(c.t, "POST", path, body, false)
}

func (r *response) decodeObject(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.body, &m); err != nil {
		t.Fatalf("body is not a JSON object: %v (%s)", err, r.body)
	}
	return m
}

func (r *response) decodeArray(t *testing.T) []map[string]any {
	t.Helper()
	var arr []map[string]any
	if err := json.Unmarshal(r.body, &arr); err != nil {
		t.Fatalf("body is not a JSON array: %v (%s)", err, r.body)
	}
	return arr
}

func (r *response) errorOf(t *testing.T) (code, message string) {
	t.Helper()
	m := r.decodeObject(t)
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected {\"error\":{…}} envelope, got: %s", r.body)
	}
	code, _ = errObj["code"].(string)
	message, _ = errObj["message"].(string)
	return code, message
}

func mustStatus(t *testing.T, r *response, want int) {
	t.Helper()
	if r.status != want {
		t.Fatalf("status = %d, want %d (body: %s)", r.status, want, r.body)
	}
}

func asID(v any) int64 {
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return int64(f)
}

// freshEmail returns a collision-free fixture email for this run.
func freshEmail(prefix string) string {
	return fmt.Sprintf("%s%d@example.test", prefix, emailSeq.Add(1))
}

// newUser registers an account and returns its logged-in client (fixtures:
// password Passw0rd!x per test_cases.md). The client remembers its email for
// later DB-level assertions.
func newUser(t *testing.T, prefix string) *apiClient {
	t.Helper()
	c := newClient("").bound(t)
	email := freshEmail(prefix)
	r := c.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": email, "password": testPasswd}, true)
	mustStatus(t, r, 201)
	token := cookieValue(r, "pebt_token")
	if token == "" {
		t.Fatal("register did not set the pebt_token cookie")
	}
	c.token = token
	c.email = email
	return c
}

func cookieValue(r *response, name string) string {
	for _, ck := range r.cookies {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func createCategory(t *testing.T, c *apiClient, name, typ string) int64 {
	t.Helper()
	r := c.post("/api/v1/categories", map[string]any{"name": name, "type": typ})
	mustStatus(t, r, 201)
	return asID(r.decodeObject(t)["id"])
}

func createTransaction(t *testing.T, c *apiClient, typ string, categoryID int64, amount, date, description string) int64 {
	t.Helper()
	body := map[string]any{
		"type": typ, "category_id": categoryID, "amount": amount, "date": date,
	}
	if description != "" {
		body["description"] = description
	}
	r := c.post("/api/v1/transactions", body)
	mustStatus(t, r, 201)
	return asID(r.decodeObject(t)["id"])
}

func createBudget(t *testing.T, c *apiClient, categoryID int64, year, month int, amount string) int64 {
	t.Helper()
	r := c.post("/api/v1/budgets", map[string]any{
		"category_id": categoryID, "year": year, "month": month, "amount": amount,
	})
	mustStatus(t, r, 201)
	return asID(r.decodeObject(t)["id"])
}
