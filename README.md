# Personal Expense and Budget Tracker

Web application to record income and expenses, organize transactions into categories, set
monthly budgets, and monitor spending — with per-user data isolation, JWT authentication,
decimal-safe money handling, and a monthly dashboard.

**Status:** Phase 3 (implementation). Backend and frontend code are complete and compile
clean (`gofmt`/`go vet`/`go build` pass; frontend production build passes). The automated
test suite (TC-A01–A35) is **written but not yet executed** — it needs the two databases
below to exist; results will be recorded in `test_cases.md` only after a real run.

## Features

- Register / login / logout with secure password storage (bcrypt) and JWT session cookie
- Categories with Income/Expense types, CRUD + filter
- Transactions (amount, category, date, optional description) with type/category validation,
  newest-first list, filters by type / category / date range
- Monthly budgets per expense category with budget vs. actual vs. remaining and overspend highlighting
- Dashboard per selected month: total income, total expenses, remaining balance,
  expense totals by category, five most recent transactions

## Prerequisites

| Tool | Developed on | Works with |
|---|---|---|
| Go | 1.27.3 | 1.22+ (stdlib `net/http` router; no framework) |
| PostgreSQL | 18.6 | 15+ |
| Node.js + npm | 22.1.0 / 10.8.0 | 20 LTS+ (builds the React frontend via Vite) |

## Configuration

Copy `.env.example` to `.env` (repo root) and fill in real values. `.env` is git-ignored.

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string for the app database (e.g. `pebt_dev`) |
| `TEST_DATABASE_URL` | Dedicated, disposable database for the automated suite (e.g. `pebt_test`) — its schema is reset on every test run |
| `JWT_SECRET` | 32+ byte random secret — **never committed** (generate e.g. with `openssl rand -base64 48`) |
| `COOKIE_SECURE` | `true` in production (HTTPS); `false` only for local non-HTTPS development |
| `PORT` | backend listen port (default `8080`) |
| `STATIC_DIR` | optional; built-SPA directory served by the Go binary (defaults to `../frontend/dist` when it exists) |

## PostgreSQL setup

Create one login role and two databases (the test database is disposable — the suite
drops and recreates its schema on every run):

```bash
psql -U postgres -c "CREATE ROLE pebt LOGIN PASSWORD '<choose-a-password>'"
psql -U postgres -c "CREATE DATABASE pebt_dev  OWNER pebt"
psql -U postgres -c "CREATE DATABASE pebt_test OWNER pebt"
```

On Windows, `scripts/setup-db.ps1` automates this: it reads the password already stored
in `.env` and prompts only for the `postgres` superuser password.

## Migrations

Versioned plain-SQL up/down files live in `backend/migrations/` and are **embedded in the
Go binary** (decision D20), so no external migration CLI is needed:

```bash
cd backend
go run ./cmd/migrate up      # apply all pending migrations
go run ./cmd/migrate down    # roll back the latest
go run ./cmd/migrate reset   # drop the schema, then apply everything (dev/test only)
```

The files stay golang-migrate-compatible if you prefer the CLI.

## Run backend

```bash
cd backend
go run ./cmd/server          # listens on :8080 (or $PORT)
```

## Run frontend

Development (hot reload):

```bash
cd frontend
npm install
npm run dev        # Vite dev server; /api is proxied to :8080 (same-origin cookies)
```

Production (single origin, no CORS):

```bash
cd frontend
npm run build      # outputs frontend/dist
cd ../backend
go run ./cmd/server  # serves the API and the built SPA together
```

## Testing

```bash
cd backend
go test ./tests/ -count=1   # Stage A suite (TC-A01–A35) against $TEST_DATABASE_URL
```

`TEST_DATABASE_URL` must point at an empty dedicated database; the suite resets its schema
at startup (`main_test.go`) and never touches `DATABASE_URL`'s database.

Manual UI checks: follow `test_cases.md` Stage B (TC-M01–M17) and record results with
evidence in `docs/evidence/`. Per the anti-fabrication rule, tests are marked Passed only
after real execution with recorded evidence.

## Security & configuration notes

- Passwords hashed with bcrypt (cost 12); never logged.
- Auth token: JWT (HS256, 60-min expiry) in an **HttpOnly, Secure, SameSite=Lax cookie**.
- CSRF protection: mutating API calls must carry `X-Requested-With: XMLHttpRequest`.
- Cross-user access returns 404 (indistinguishable from not-found).
- **Known limitation:** logout clears the browser cookie, but a JWT is technically valid until
  its 60-minute expiry (stateless design; server-side revocation was deliberately out of scope).
- Money is exact decimal end-to-end (`NUMERIC(12,2)` + `shopspring/decimal` + JSON strings);
  floating-point is never used for money.

## Submission

- GitHub repository containing `frontend/` (React) and `backend/` (Go) — DEL1
- PostgreSQL schema delivered as versioned migrations in `backend/migrations/` — DEL2
- This README with working install/run instructions — DEL3
- Pre-submission checklist lives in this section once Phase 5 starts; the fresh-clone run
  (TC-M17) must pass first.

## Documentation index

`CLAUDE.md` (charter/rules) · `prompt.log` (prompt record) · `user_stories.md` ·
`test_cases.md` · `docs/requirements.md` · `docs/design.md` · `docs/decisions.md` ·
`docs/traceability.md` · `docs/evidence/`
