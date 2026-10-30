# Design — Personal Expense and Budget Tracker (Phase 2 v2.1)

**Status:** AWAITING USER REVIEW AND APPROVAL. This document materializes the consolidated
Phase 2 v2.1 package (approved-in-conversation drafts of 2026-10-28 + user rulings D9/D10/D14 of
2026-10-29). Nothing here is implemented; `[Proposed]` marks choices still governed by the
decision register (`docs/decisions.md`).

---

## 1. Architecture

Three tiers, no caching layer anywhere (AC9 relies on live recomputation):

```
React SPA (functional components + hooks, React Router, fetch wrapper)
      │  same-origin HTTP; cookies carried automatically
      ▼
Go REST API (/api/v1 — stdlib net/http router, layered handlers→services→repository)
      │  pgx/v5, explicit SQL, no ORM
      ▼
PostgreSQL 16 (sole persistent store — AC16)
```

**Deployment model [Proposed, D37]:** one origin. Development uses the Vite dev-server proxy
(`/api` → Go), so the browser talks to a single origin and cookies are first-party — no CORS.
Production serves the built React app (`frontend/dist`) from the Go binary — again one origin,
no CORS. Split-host hosting is documented as option B (requires `SameSite=None; Secure` +
exact-origin CORS with credentials) but is not recommended.

## 2. Data model (PostgreSQL)

Money: `NUMERIC(12,2)` everywhere (D26). Timestamps: `TIMESTAMPTZ`. `updated_at` maintained by
the application, not a trigger (U5).

### users
| Column | Type | Constraints |
|---|---|---|
| id | BIGINT GENERATED ALWAYS AS IDENTITY | PK |
| email | VARCHAR(255) | NOT NULL; **UNIQUE case-insensitive** via `CREATE UNIQUE INDEX ON users (lower(email))` (D7; plain index, no CITEXT dependency — U3) |
| password_hash | VARCHAR(255) | NOT NULL (bcrypt cost 12 — D6) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

### categories
| Column | Type | Constraints |
|---|---|---|
| id | BIGINT identity | PK |
| user_id | BIGINT NOT NULL | FK → users **ON DELETE CASCADE** (user deletion is not an assignment feature; cascade is a safety default — U6) |
| name | VARCHAR(100) | NOT NULL |
| type | VARCHAR(10) NOT NULL | CHECK (`INCOME`,`EXPENSE`) |
| | | `UNIQUE (user_id, lower(name))` — case-insensitive per-user duplicate guard, **if D18 stands** (U1; queries must use the identical `lower(name)` expression) |
| | | **`UNIQUE (id, user_id, type)`** — FK target enabling ownership+type composite keys |
| | | INDEX (user_id) |

### transactions
| Column | Type | Constraints |
|---|---|---|
| id | BIGINT identity | PK |
| user_id | BIGINT NOT NULL | FK → users CASCADE |
| type | VARCHAR(10) NOT NULL | CHECK (`INCOME`,`EXPENSE`) |
| category_id | BIGINT NOT NULL | composite FK below |
| amount | NUMERIC(12,2) NOT NULL | CHECK (amount > 0) → AC4 |
| tx_date | DATE NOT NULL | AC5 (a DATE is always a real calendar date; no time/timezone) |
| description | VARCHAR(500) | NULL |
| created_at / updated_at | TIMESTAMPTZ | NOT NULL |

**Composite FK (ownership + type — AC2/AC6):**
`FOREIGN KEY (category_id, user_id, type) REFERENCES categories (id, user_id, type) ON DELETE RESTRICT`
A transaction can only reference a category owned by the same user **and** of the same type —
enforced by PostgreSQL even if an API check were skipped. RESTRICT implements AC12.

### budgets
| Column | Type | Constraints |
|---|---|---|
| id | BIGINT identity | PK |
| user_id | BIGINT NOT NULL | FK → users CASCADE |
| category_id | BIGINT NOT NULL | composite FK below |
| cat_type | VARCHAR(10) NOT NULL | CHECK (`cat_type = 'EXPENSE'`) — constant carrier column |
| year | SMALLINT NOT NULL | CHECK (2000–2100) |
| month | SMALLINT NOT NULL | CHECK (1–12) |
| amount | NUMERIC(12,2) NOT NULL | CHECK (amount > 0) → AC4 |
| created_at / updated_at | TIMESTAMPTZ | NOT NULL |

**Composite FK (ownership + expense-only — AC2/AC7/AC12):**
`FOREIGN KEY (category_id, user_id, cat_type) REFERENCES categories (id, user_id, type) ON DELETE RESTRICT`
Because the CHECK pins `cat_type` to `'EXPENSE'`, this FK can only ever match expense categories
of the same user. (A constant cannot be embedded directly in an FK column list, hence the
carrier column — decision D19; a trigger alternative was rejected as complexity.)

**`UNIQUE (user_id, category_id, year, month)`** → AC8 (one budget per user/category/month/year).

**Indexes [Proposed]:** transactions `(user_id, tx_date DESC)`, `(user_id, type)`,
`(user_id, category_id, tx_date)` (serves budget aggregation); budgets `(user_id, year, month)`;
categories `(user_id)`.

**Category type changes (D27):** allowed only while unused — retyping a referenced category
violates the composite FKs, so the DB blocks it automatically (mapped to 409); no separate
validation code needed. No DB trigger proposed (U2): direct SQL retyping of a referenced
category is rejected by the same FKs, so the residual risk is nil for practical purposes.

### How the required calculations work

- **Calendar-month actual (AC10):**
  `SUM(amount) WHERE user_id = b.user_id AND category_id = b.category_id AND type='EXPENSE'
   AND tx_date >= make_date(b.year, b.month, 1)
   AND tx_date < make_date(b.year, b.month, 1) + INTERVAL '1 month'`
  Half-open range on a timezone-free `DATE` — month lengths and boundaries are exact (D24).
- **Remaining / overspend (AC11):** `remaining = budget.amount − actual` (may be negative);
  `over_budget = actual > budget.amount`. UI renders overspending distinctly.
- **Dashboard (FR5, AC9):** every request recomputes live — `total_income = SUM(type='INCOME')`,
  `total_expense = SUM(type='EXPENSE')`, `remaining_balance = income − expense`,
  `expenses_by_category = GROUP BY category_id` (selected month),
  `recent_transactions = ORDER BY tx_date DESC, id DESC LIMIT 5` (**overall** — D13).
  Nothing cached; the frontend refetches after every mutation.

## 3. API contract

All routes under `/api/v1` (D29). Authentication = `pebt_token` cookie (D10); no
`Authorization` header. Mutating requests (`POST/PUT/DELETE`) additionally require
`X-Requested-With: XMLHttpRequest` (CSRF, D36) → else **403**.

| # | Method & path | Purpose → success | Validation & key errors |
|---|---|---|---|
| 1 | POST `/auth/register` | create account, set session cookie → **201** `{expires_at}` | email format / password policy → 422; dup email → 409 `EMAIL_TAKEN` |
| 2 | POST `/auth/login` | verify, set session cookie → **200** `{expires_at}` | 401 `INVALID_CREDENTIALS` (identical message for unknown email / wrong password) |
| 3 | POST `/auth/logout` | clear cookie → **204** (`Set-Cookie` Max-Age=0) | 401 if no session |
| 4 | GET `/categories?type=` | list, FR2.6 filter → **200** | `type` optional; invalid value → 422 |
| 5 | POST `/categories` | create → **201** | name 1–100, enum type → 422; dup name → 409 (D18) |
| 6 | PUT `/categories/{id}` | rename → **200** | not-owned/missing → 404; dup → 409; retype-while-referenced → 409 (DB FK) |
| 7 | DELETE `/categories/{id}` | → **204** | referenced by tx/budget → 409 `CATEGORY_IN_USE` (AC12) |
| 8 | GET `/transactions?type=&category_id=&from=&to=` | list FR3.4–3.5, newest-first → **200** | bad filter values → 422; foreign category → 404 |
| 9 | POST `/transactions` | create → **201** | amount > 0, ≤2 dp → else 422; invalid date → 422; type≠category → 422 `TYPE_CONFLICT`; foreign/unknown category → 404 |
| 10 | GET `/transactions/{id}` | → **200** | not-owned/missing → 404 |
| 11 | PUT `/transactions/{id}` | full replace → **200** | every field re-validated; type change requires matching category (422); amount/date rules re-applied; 0 rows → 404 |
| 12 | DELETE `/transactions/{id}` | → **204** | not-owned → 404 |
| 13 | GET `/budgets?year=&month=` | list with computed `actual_amount`, `remaining`, `over_budget` → **200** | year/month required, range-checked → 422 |
| 14 | POST `/budgets` | create → **201** | non-expense category → 422 (AC7); duplicate → 409 `BUDGET_DUPLICATE` (AC8); amount rules as AC4 |
| 15 | PUT `/budgets/{id}` | amount-only edit → **200** (D38; changing category/month/year = delete + recreate) | amount rules; not-owned → 404 |
| 16 | DELETE `/budgets/{id}` | → **204** | not-owned → 404 |
| 17 | GET `/dashboard?year=&month=` | FR5 aggregates → **200** | year/month required → 422 |

No `GET /categories/{id}` (list suffices) and no `/auth/me` (D32 — login state is probed with
any cheap authenticated call). Both omissions are proposals.

### Error policy (D17/D28 — final)

Envelope: `{"error":{"code":"…","message":"…"}}`. Codes: `VALIDATION_ERROR`, `INVALID_CREDENTIALS`,
`TOKEN_MISSING`/`TOKEN_EXPIRED`/`TOKEN_INVALID`, `CSRF_CHECK_FAILED`, `NOT_FOUND`,
`EMAIL_TAKEN`, `CATEGORY_DUPLICATE`, `TYPE_CONFLICT`, `CATEGORY_IN_USE`, `BUDGET_DUPLICATE`, `BAD_REQUEST`, `INTERNAL`.

| Status | Meaning |
|---|---|
| 400 | malformed JSON body |
| 401 | no/expired/invalid session; generic login failure |
| 403 | CSRF header missing on a mutating request |
| 404 | missing **or** owned by another user; unknown route |
| 409 | state conflicts (dup email/category/budget, category in use, retype conflict) |
| 422 | any field or query-param validation failure (incl. >2 decimal places) |
| 500 | unexpected; no internals leaked |

## 4. Validation matrix (AC3 frontend / AC14 backend / database)

| Rule | Frontend | Backend | Database |
|---|---|---|---|
| Email required, RFC format | required + pattern | re-validate + unique check | UNIQUE lower(email) |
| Password ≥8 chars, ≥1 letter+digit, ≤72 bytes (D8) | live hints | re-validate; bcrypt | — |
| Category name 1–100, dup check | maxlength + pre-check | re-validate + 409 | UNIQUE(user_id, lower(name)) (D18) |
| Category type enum | radio/select | enum check | CHECK |
| Amount > 0, ≤2 dp (AC4/AC13) | number input min 0.01 step 0.01 | decimal parse; `10.999` → 422, never rounded | NUMERIC(12,2) CHECK > 0 |
| Date valid (AC5) | `type="date"` | strict DATE parse | DATE domain |
| Description ≤ 500 optional | maxlength | length check | VARCHAR(500) |
| Tx type = category type (AC6) | dropdown pre-filtered | check + 422 | composite FK |
| Budget → expense category only (AC7) | dropdown filtered | check + 422 | composite FK (`cat_type`) |
| Budget unique (AC8) | friendly pre-check | 409 | UNIQUE |
| Category in use (AC12) | delete disabled + tooltip | FK 23001 (restrict_violation) → 409 | FK RESTRICT |
| Budget month/year ranges | selectors | range check | CHECKs |
| CSRF on mutations | fetch wrapper adds header | middleware 403 | — |

## 5. Authentication & security (final)

| Topic | Design |
|---|---|
| Hashing | bcrypt cost 12 (D6); alternative argon2id noted, rejected for simplicity |
| Password policy | ≥8 chars, ≥1 letter + 1 digit, ≤72 bytes (bcrypt cap prevents silent truncation) — D8 |
| JWT | HS256; claims `sub`, `iat`, `exp`; `iss="pebt"`; 60-minute expiry (D9); secret = 32+ byte env var, `.env` git-ignored, `.env.example` committed without secrets |
| Transport | `Set-Cookie: pebt_token=<jwt>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=3600` (D10/D37). `COOKIE_SECURE=false` env fallback for local non-HTTPS dev only. **No token in any response body** — login/register return `{expires_at}` (D30) |
| Logout | `POST /auth/logout` → 204 + cookie cleared server-side (Max-Age=0) |
| CSRF | Mutating requests require `X-Requested-With: XMLHttpRequest` → else 403 (D36). SameSite=Lax is the first layer; the header is the second. Textbook alternative (double-submit token) noted as option B |
| Cross-user | All queries scoped by JWT-derived `user_id`; foreign → 404 (D16). Composite FKs make forged references impossible at the DB |
| Known limitation | Stateless logout: after cookie clearing, a copied token remains valid until its 60-min expiry. Server-side revocation (denylist) deliberately out of scope; mitigations: short expiry, HTTPS, HttpOnly. Documented in README |
| Never logged | passwords, tokens |

## 6. Money & serialization (final — D14/D15/D26)

`NUMERIC(12,2)` → `shopspring/decimal` (never float32/64 in the money path) → **explicit
`StringFixed(2)` into string DTO fields** → JSON `"amount":"123.45"` → React formats ₹ (D11)
and never parses money as a float. The explicit-DTO rule exists because the decimal library's
default JSON marshaling has varied across versions — the wire format is pinned in our own code.
API inputs with >2 decimal places are rejected 422 **before** any storage, so the column's
scale-2 rounding is unreachable through the API (no silent rounding).

## 7. Repository layout & migrations (planned Phase 3)

```
frontend/            React SPA (Vite) — functional components + hooks only
backend/
  cmd/server/        entry point
  internal/          handlers, services, repository, auth, money, middleware
  migrations/        golang-migrate SQL up/down files (DEL2)
.env.example         config template (no real secrets)
docs/                requirements, design, decisions, traceability, evidence
CLAUDE.md  README.md  prompt.log  user_stories.md  test_cases.md
```

## 8. Technology pins [Proposed — exact patches pinned at Phase 3 start]

Go 1.22+ (stdlib router, D21) · PostgreSQL 16.x · `jackc/pgx/v5` · `golang-migrate` v4 (D20) ·
`shopspring/decimal` · `golang-jwt/jwt/v5` + `x/crypto/bcrypt` · React ≥18.3 + Vite ≥5 +
react-router (functional components + hooks only, CON1) · Node 20 LTS · tests: Go `testing` +
`httptest` (D22); frontend behavior verified manually (Stage B).

## 9. Open design notes (carried, not hidden)

- **U1** category-name uniqueness scope — per user, case-insensitive (D18), optional vs doc.
- **U2** no DB trigger for type immutability — composite FKs already block retyping referenced categories.
- **U3** `lower(email)` unique index (no CITEXT extension dependency).
- **U5** `updated_at` app-maintained, not trigger-maintained.
- **U6** user deletion isn't an assignment feature; CASCADE is a safety default only.
