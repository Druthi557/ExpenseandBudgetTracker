package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"pebt/backend/internal/auth"
)

// TC-A01 — register: 201, HttpOnly SameSite=Lax cookie, {expires_at} body,
// no token in the body (AC1, D10, D30).
func TestTC_A01_Register(t *testing.T) {
	c := newClient("").bound(t)
	email := freshEmail("a01")
	r := c.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": email, "password": testPasswd}, true)
	mustStatus(t, r, 201)

	// Cookie attributes (D10): HttpOnly, SameSite=Lax, Path=/, Max-Age=3600.
	var session *http.Cookie
	for _, ck := range r.cookies {
		if ck.Name == "pebt_token" {
			session = ck
		}
	}
	if session == nil {
		t.Fatal("pebt_token cookie missing on register response")
	}
	if !session.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if session.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", session.SameSite)
	}
	if session.MaxAge != 3600 {
		t.Errorf("Max-Age = %d, want 3600", session.MaxAge)
	}
	if session.Path != "/" {
		t.Errorf("Path = %q, want /", session.Path)
	}

	// Body: {expires_at} only — no token field (D30).
	m := r.decodeObject(t)
	if _, ok := m["expires_at"]; !ok {
		t.Error("expires_at missing from response body")
	}
	if _, ok := m["token"]; ok {
		t.Error("token must never appear in a response body")
	}
	body := string(r.body)
	if strings.Contains(body, session.Value) {
		t.Error("raw JWT leaked in response body")
	}
}

// TC-A02 — duplicate email: 409 EMAIL_TAKEN (AC14).
func TestTC_A02_DuplicateEmail(t *testing.T) {
	email := freshEmail("a02")
	first := newClient("").bound(t)
	mustStatus(t, first.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": email, "password": testPasswd}, true), 201)

	second := newClient("").bound(t)
	r := second.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": email, "password": testPasswd}, true)
	mustStatus(t, r, 409)
	if code, _ := r.errorOf(t); code != "EMAIL_TAKEN" {
		t.Fatalf("code = %q, want EMAIL_TAKEN", code)
	}

	// Same address, different case, is still the same account (D7).
	r = second.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": strings.ToUpper(email), "password": testPasswd}, true)
	mustStatus(t, r, 409)
}

// TC-A04 — login with correct credentials: 200, HttpOnly cookie, {expires_at}.
func TestTC_A04_Login(t *testing.T) {
	email := freshEmail("a04")
	reg := newClient("").bound(t)
	mustStatus(t, reg.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": email, "password": testPasswd}, true), 201)

	c := newClient("").bound(t)
	r := c.do(t, "POST", "/api/v1/auth/login",
		map[string]any{"email": email, "password": testPasswd}, true)
	mustStatus(t, r, 200)
	if cookieValue(r, "pebt_token") == "" {
		t.Fatal("login did not set the pebt_token cookie")
	}
	for _, ck := range r.cookies {
		if ck.Name == "pebt_token" && !ck.HttpOnly {
			t.Error("login cookie must be HttpOnly")
		}
	}
	if _, ok := r.decodeObject(t)["expires_at"]; !ok {
		t.Error("expires_at missing from login response")
	}
}

// TC-A03 — logout: 204, cookie cleared (Max-Age=0), protected call then 401.
func TestTC_A03_Logout(t *testing.T) {
	c := newUser(t, "a03")

	r := c.post("/api/v1/auth/logout", nil)
	mustStatus(t, r, 204)

	var clearing *http.Cookie
	for _, ck := range r.cookies {
		if ck.Name == "pebt_token" {
			clearing = ck
		}
	}
	if clearing == nil {
		t.Fatal("logout did not send a clearing cookie")
	}
	if clearing.MaxAge > 0 {
		t.Errorf("clearing cookie Max-Age = %d, want <= 0", clearing.MaxAge)
	}

	// The browser would drop the cookie; mirror that and confirm the API
	// refuses unauthenticated access (FR1.3).
	c.token = ""
	r = c.get("/api/v1/categories")
	mustStatus(t, r, 401)
}

// TC-A31 — register field validation: malformed email and weak password both
// 422 VALIDATION_ERROR (AC14, D8).
func TestTC_A31_RegisterValidation(t *testing.T) {
	c := newClient("").bound(t)

	r := c.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": "not-an-email", "password": testPasswd}, true)
	mustStatus(t, r, 422)
	if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
		t.Fatalf("email case: code = %q, want VALIDATION_ERROR", code)
	}

	r = c.do(t, "POST", "/api/v1/auth/register",
		map[string]any{"email": freshEmail("a31"), "password": "short"}, true)
	mustStatus(t, r, 422)
	if code, _ := r.errorOf(t); code != "VALIDATION_ERROR" {
		t.Fatalf("password case: code = %q, want VALIDATION_ERROR", code)
	}
}

// TC-A32 — login failures are generic and identical (no account-existence
// hint) (AC1, D28).
func TestTC_A32_LoginInvalid(t *testing.T) {
	email := freshEmail("a32")
	newUser(t, "a32")

	c := newClient("").bound(t)
	wrongPw := c.do(t, "POST", "/api/v1/auth/login",
		map[string]any{"email": email, "password": "WrongPass1"}, true)
	mustStatus(t, wrongPw, 401)
	code1, msg1 := wrongPw.errorOf(t)

	unknown := c.do(t, "POST", "/api/v1/auth/login",
		map[string]any{"email": freshEmail("a32-unknown"), "password": testPasswd}, true)
	mustStatus(t, unknown, 401)
	code2, msg2 := unknown.errorOf(t)

	if code1 != "INVALID_CREDENTIALS" || code2 != "INVALID_CREDENTIALS" {
		t.Fatalf("codes = %q / %q, want INVALID_CREDENTIALS twice", code1, code2)
	}
	if msg1 != msg2 {
		t.Fatalf("messages differ: %q vs %q — must be identical", msg1, msg2)
	}
}

// TC-A30 — protected endpoint with no / expired / tampered cookie: 401 with
// distinct codes (AC1, AC14).
func TestTC_A30_TokenStates(t *testing.T) {
	// (a) no cookie
	anon := newClient("").bound(t)
	r := anon.get("/api/v1/categories")
	mustStatus(t, r, 401)
	if code, _ := r.errorOf(t); code != "TOKEN_MISSING" {
		t.Fatalf("no-cookie code = %q, want TOKEN_MISSING", code)
	}

	// (b) expired token: mint one with a shifted clock (auth.Now hook).
	c := newUser(t, "a30")
	userID := userIDForEmail(t, c.email)
	auth.Now = func() time.Time { return time.Now().Add(-2 * auth.TokenTTL) }
	expired, _, err := auth.IssueToken(userID, jwtSecret)
	auth.Now = time.Now
	if err != nil {
		t.Fatalf("mint expired token: %v", err)
	}
	c.token = expired
	r = c.get("/api/v1/categories")
	mustStatus(t, r, 401)
	if code, _ := r.errorOf(t); code != "TOKEN_EXPIRED" {
		t.Fatalf("expired code = %q, want TOKEN_EXPIRED", code)
	}

	// (c) tampered token
	c.token = strings.TrimSuffix(expired, "x") + "y"
	r = c.get("/api/v1/categories")
	mustStatus(t, r, 401)
	if code, _ := r.errorOf(t); code != "TOKEN_INVALID" {
		t.Fatalf("tampered code = %q, want TOKEN_INVALID", code)
	}
}

// TC-A35 — mutating request without the CSRF header: 403; with it: 201 (D36).
func TestTC_A35_CSRF(t *testing.T) {
	c := newUser(t, "a35")

	r := c.postNoCSRF("/api/v1/categories", map[string]any{"name": "CSRFless", "type": "EXPENSE"})
	mustStatus(t, r, 403)
	if code, _ := r.errorOf(t); code != "CSRF_CHECK_FAILED" {
		t.Fatalf("code = %q, want CSRF_CHECK_FAILED", code)
	}

	r = c.post("/api/v1/categories", map[string]any{"name": "CSRFless", "type": "EXPENSE"})
	mustStatus(t, r, 201)
}

// TC-A25 — password storage is bcrypt at the database level (AC1).
func TestTC_A25_BcryptStorage(t *testing.T) {
	c := newUser(t, "a25") // client remembers the exact registered email

	var hash string
	err := pool.QueryRow(context.Background(),
		`SELECT password_hash FROM users WHERE lower(email) = $1`, c.email).Scan(&hash)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("password_hash %q is not bcrypt format", hash)
	}
	if strings.Contains(hash, testPasswd) {
		t.Fatal("plaintext password found in storage")
	}
}

// ---------- shared helpers for token/DB plumbing ----------

func userIDForEmail(t *testing.T, email string) int64 {
	t.Helper()
	if email == "" {
		t.Fatal("userIDForEmail needs the registration email")
	}
	var id int64
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE lower(email) = $1`, email).Scan(&id)
	if err != nil {
		t.Fatalf("lookup user: %v", err)
	}
	return id
}
