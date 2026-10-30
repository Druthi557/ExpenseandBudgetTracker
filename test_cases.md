# Test Cases — Personal Expense and Budget Tracker

**Status legend:** Stage A (automated) was **executed three times on 2026-10-30** against
the dedicated test database. Rows below reflect the **latest run (run 3, after all
approved fixes): 35 PASS · 0 FAIL · 0 blocked/skipped — exit 0**. Full three-run history
with every intermediate defect and fix is in the execution records below. Stage B
(manual) was **executed 2026-10-30 — all 17 cases PASS** (TC-M17 with a recorded
qualification; see the Stage B records). A test
may be marked Passed only after real execution with recorded evidence
(anti-fabrication rule, see CLAUDE.md §7).

**Stages:**
- **Stage A — automated**: Go `httptest` integration tests against a dedicated PostgreSQL test
  database built from migrations [implemented in Phase 3]. A few cases operate at the
  **database level** (direct SQL) to prove the constraints independent of the API.
- **Stage B — manual**: UI / end-to-end checks executed by a human after implementation,
  recorded with date, result, and evidence in `docs/evidence/`.

**Shared fixtures:**
- User A `a@example.test` / User B `b@example.test` (password `Passw0rd!x`).
- Categories for A: `Salary` (Income), `Food` (Expense), `Travel` (Expense).
- Money values are JSON strings with exactly 2 decimals.
- Cookie: `pebt_token` (HttpOnly). Mutating requests carry `X-Requested-With: XMLHttpRequest`.
- API base path: `/api/v1`.

---

# Stage A — Automated (35 cases)

## Authentication & sessions

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A01 | API | AC1 · FR1.1 · US1 | empty users table | `POST /auth/register` `{email:"a@example.test", password:"Passw0rd!x"}` + CSRF header | 201; `Set-Cookie: pebt_token=…; HttpOnly; SameSite=Lax`; body contains `expires_at`; **no token in body** | PASS | 201; HttpOnly+SameSite=Lax cookie; `expires_at` in body; no token in body | run 3 log 2026-10-30 |
| TC-A02 | API | AC14 · FR1.1 · US1 | user A registered | `POST /auth/register` same email again | 409 `EMAIL_TAKEN` | PASS | 409 `EMAIL_TAKEN` | run 3 log 2026-10-30 |
| TC-A03 | API | FR1.3 · US3 | logged-in client (valid cookie) | `POST /auth/logout` + CSRF header; then call any protected endpoint | 204; `Set-Cookie` clears token (`Max-Age=0`); subsequent protected call → 401 | PASS | 204; clearing cookie `Max-Age=0`; protected call 401 | run 3 log 2026-10-30 |
| TC-A04 | API | AC1 · FR1.2 · US2 | user A registered | `POST /auth/login` correct credentials | 200; HttpOnly session cookie set; `{expires_at}` in body | PASS | 200; cookie set; `expires_at` in body | run 3 log 2026-10-30 |
| TC-A31 | API | AC14 · FR1.1 · US1 | none | `POST /auth/register` with `email:"not-an-email"`, then with `password:"short"` | 422 `VALIDATION_ERROR` on both | PASS | 422 on both | run 3 log 2026-10-30 |
| TC-A32 | API | FR1.2 · AC14 · US2 | user A registered | `POST /auth/login` wrong password; then unknown email | 401 `INVALID_CREDENTIALS`; message **identical** in both cases | PASS | 401, identical message both cases | run 3 log 2026-10-30 |
| TC-A30 | API | AC1 · AC14 | logged-in state | access protected endpoint with (a) no cookie, (b) expired token cookie, (c) tampered cookie | 401 with appropriate code, each | PASS | 401 for each state | run 3 log 2026-10-30 |
| TC-A35 | API | AC1 · AC14 (D36) | logged-in client | `POST /categories` **without** `X-Requested-With`; repeat **with** header | 403 `CSRF_CHECK_FAILED`; then 201 | PASS | 403 without header; 201 with | run 3 log 2026-10-30 |
| TC-A25 | Database | AC1 · US1 | user registered via API | read `users` row directly in test DB | `password_hash` is bcrypt format (`$2…`), ≠ plaintext | PASS | bcrypt `$2…` hash stored; plaintext absent | run 3 log 2026-10-30 |

## Per-user isolation

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A05 | API | AC2 · FR1.5 · US6 | A has tx id *X*; B logged in | B: `GET /transactions/{X}` | 404 (no data leaked) | PASS | 404, no data leaked | run 3 log 2026-10-30 |
| TC-A17 | API | AC2 · US7 | A has tx id *X*; B logged in | B: `PUT /transactions/{X}`, then `DELETE /transactions/{X}` | 404 on both; A's data unchanged | PASS | 404 on both; A's data unchanged | run 3 log 2026-10-30 |
| TC-A33 | API | AC2 · US6 | A and B each own categories, txs, budgets | each user GETs `/transactions`, `/categories`, `/budgets` | every list contains only that user's rows | PASS | every list strictly per-user | run 3 log 2026-10-30 |
| TC-A23 | Database | AC2 · AC6 · AC7 · AC8 · AC12 | schema + users A,B; B has expense category | direct SQL bypassing API: (a) tx by A referencing B's category; (b) budget on an income category; (c) second identical budget row; (d) `DELETE` a category referenced by a tx | PostgreSQL rejects each: (a) 23503, (b) 23503, (c) 23505 unique, (d) 23001 RESTRICT | PASS | (a) 23503, (b) 23503, (c) 23505, (d) 23001 `restrict_violation` — expectations corrected per user ruling (was wrongly 23503 for (d)) | run 3 log 2026-10-30 |

## Categories

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A13 | API | FR2.1–2.4 · US4 | logged in | `POST` category → `PUT` rename → `GET` list → `DELETE` unused category | 201 / 200 / 200 / 204 | PASS | 201 / 200 / 200 / 204; lowercase wire keys | run 3 log 2026-10-30 |
| TC-A26 | API | FR2.5 · AC14 · US4 | category `Food` exists *(runs only if decision D18 stands)* | `POST` category `FOOD` (same user, different case) | 409 `CATEGORY_DUPLICATE` | PASS | 409 `CATEGORY_DUPLICATE` | run 3 log 2026-10-30 |
| TC-A27 | API | FR2.6 · US4 | mixed income/expense categories exist | `GET /categories?type=EXPENSE`; `?type=INCOME`; `?type=Bogus` | only matching type (200 each); invalid value → 422 | PASS | filters exact; `Bogus` → 422 | run 3 log 2026-10-30 |
| TC-A06 | API | AC12 · FR2.4 · US4 | category `Food` referenced by ≥1 tx | `DELETE /categories/{Food}` | 409 `CATEGORY_IN_USE`; category still present | PASS | 409 `CATEGORY_IN_USE` (23001 now mapped); category retained | run 3 log 2026-10-30 |

## Transactions

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A15 | API | FR3.1–3.3 · US5 | categories exist | `POST` income tx (with description); `POST` expense tx (no description); `GET` by id; `PUT` valid change; `DELETE` | 201 / 201 / 200 / 200 / 204; money echoed as 2-dp strings | PASS | 201 / 201 / 200 / 200 / 204; 2-dp strings round-trip | run 3 log 2026-10-30 |
| TC-A16 | API | AC6 · AC4 · AC5 · US7 | expense tx on `Food` exists | `PUT` switching `type=INCOME` keeping `Food`; `PUT` with income category; `PUT` amount `"-1"`; `PUT` date `"2026-02-30"` | 422 / 200 / 422 / 422 | PASS | 422 / 200 / 422 / 422 | run 3 log 2026-10-30 |
| TC-A07 | API | AC4 · AC13 · AC14 · US5, US8 | logged in | `POST` transaction with amount `"0"`, `"-5"`, `"10.999"`, `"abc"` (4 requests); same matrix for budget amount | 422 `VALIDATION_ERROR` each; **no silent rounding**; nothing stored | PASS | 422 `VALIDATION_ERROR` for every invalid amount; nothing stored | run 3 log 2026-10-30 |
| TC-A08 | API | AC5 · FR3.2 · US5 | logged in | `POST` transaction with `date:"2026-02-30"`, then `"not-a-date"` | 422 each | PASS | 422 each | run 3 log 2026-10-30 |
| TC-A09 | API | AC6 · FR3.2 · US5 | income category `Salary` exists | `POST` transaction `type:EXPENSE`, category `Salary` | 422 `TYPE_CONFLICT`; no row created | PASS | 422 `TYPE_CONFLICT`; no row created | run 3 log 2026-10-30 |
| TC-A14 | API | FR3.4–3.5 · US6 | ≥3 txs incl. two on the same date, across types/categories/months | `GET /transactions` (no params); then `?type=`, `?category_id=`, `?from=&to=` | default list newest-first with same-date ordered by id DESC; each filter returns exactly the matching subset | PASS | newest-first with id-DESC tie-break; every filter exact (SQL placeholder bug fixed in run 3) | run 3 log 2026-10-30 |
| TC-A28 | API | FR3.5 · AC14 · US6 | logged in | `GET /transactions?from=bad`; `?from=2026-11-01&to=2026-10-01`; `?category_id=<foreign id>` | 422 / 422 / 404 | PASS | 422 / 422 / 404 | run 3 log 2026-10-30 |

## Budgets & calculations

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A18 | API | FR4.1–4.4 · US8, US10 | expense category exists | `POST` budget → `PUT` amount `"200.00"` → `DELETE` | 201 / 200 / 204 | PASS | 201 / 200 / 204 | run 3 log 2026-10-30 |
| TC-A19 | API | AC4 · AC14 · US8 | expense category exists | `POST` budget with amount `"0"`, `"-5"`, `"10.999"` | 422 each | PASS | 422 each | run 3 log 2026-10-30 |
| TC-A10 | API | AC7 · FR4.1 · US8 | income category `Salary` exists | `POST` budget on `Salary` | 422 `TYPE_CONFLICT`; no row created | PASS | 422 `TYPE_CONFLICT`; no row created (code canonicalized per user ruling; `TYPE_CONFLICT` added to design envelope list) | run 3 log 2026-10-30 |
| TC-A11 | API | AC8 · FR4.1 · US8 | budget `Food` 2026-11 `"100.00"` exists | `POST` identical budget | 409 `BUDGET_DUPLICATE` | PASS | 409 `BUDGET_DUPLICATE` | run 3 log 2026-10-30 |
| TC-A12 | API | AC9 · AC10 · FR4.5 · US9 | budget `Food` Nov-2026 `"100.00"`; txs `Food` 2026-11-01 `"40.00"`, 2026-11-30 `"50.00"`, 2026-12-01 `"500.00"` | `GET /budgets?year=2026&month=11` | actual `"90.00"`, remaining `"10.00"` (Dec tx excluded — half-open month range); then update + delete a November tx and re-`GET /dashboard`: totals change immediately | PASS | actual `"90.00"`, remaining `"10.00"`, Dec excluded; dashboard reflects update/delete immediately | run 3 log 2026-10-30 |
| TC-A20 | API | AC11 · FR4.5–4.6 · US9 | budget `"100.00"`; November `Food` expenses total `"150.00"` | `GET /budgets?year=2026&month=11` | row shows actual `"150.00"`, remaining `"-50.00"`, `over_budget: true` | PASS | actual `"150.00"`, remaining `"-50.00"`, `over_budget: true` | run 3 log 2026-10-30 |

## Money & decimal exactness

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A22 | API | AC13 · US5, US9, US11 | logged in | create txs `"10.10"` + `"20.20"` + `"0.01"` (same category/month) and one `"99999999.99"`; read sums via dashboard + budget endpoints | sums exactly `"30.31"` (no `30.309999…` float artifacts); large value intact; **every** money field in every response is a 2-dp string; POST→GET round-trip identical | PASS | sums exactly `"30.31"`; large value intact; all money 2-dp strings; round-trip identical | run 3 log 2026-10-30 |

## Dashboard

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A21 | API | FR5.1–5.5 · AC9 · US11 | month seeded: 2 income txs, expenses across 2 categories, 6 txs total; 1 tx in another month | `GET /dashboard?year=…&month=…` | exact `total_income`, `total_expense`, `remaining_balance`; `expenses_by_category` sums per category; `recent_transactions` = 5 newest overall (`date DESC, id DESC`); other-month tx excluded from month totals | PASS | all aggregates exact; other-month excluded; recent = 5 newest | run 3 log 2026-10-30 |
| TC-A34 | API | FR5 · US11 | fresh user, no data | `GET /dashboard?year=2026&month=1` | zeros (`"0.00"` totals, `"0.00"` remaining), empty `expenses_by_category`, empty `recent_transactions`, no errors | PASS | zeros and empty lists, no errors | run 3 log 2026-10-30 |

## Errors, envelope & persistence

| ID | Level | Traces | Preconditions & data | Input / steps | Expected result | Status | Actual | Evidence |
|---|---|---|---|---|---|---|---|---|
| TC-A29 | API | AC14 | logged in | trigger one 409, one 404, one 422; `GET /api/v1/nope`; `POST` malformed JSON body | every error body = `{"error":{"code","message"}}`; unknown route → 404 envelope; malformed JSON → 400; no stack traces or internals | PASS | envelope shape held for all five triggers; no internals leaked | run 3 log 2026-10-30 |
| TC-A24 | API | AC16 | category + tx + budget created via API | stop the API process; restart; `GET` the same resources | all data intact (PostgreSQL is the sole persistent store) | PASS | data intact across server restart | run 3 log 2026-10-30 |

### Stage A execution records — 2026-10-30 (three runs, IST)

**Run 1 (~08:47) — before any fix.** Go 1.27.3 · PostgreSQL 18.6 · Windows 11 ·
`TEST_DATABASE_URL` → dedicated `pebt_test` (schema reset by the suite). Command:
`go test ./tests/... -count=1 -v`, all 35 cases, none skipped. **14 PASS (4 unreliable) ·
21 FAIL** · exit 1. Evidence: `docs/evidence/2026-10-30-stage-a-run.log`. Root causes:
(1) `store.Category` missing `json` tags → capitalized wire keys + `category_id=0`
cascade (19 failures); (2) TC-A03 sent DELETE to the POST-only logout route; (3) TC-A25
queried a never-registered email.

**Approved fixes #1 (prompt #24):** lowercase `json` tags on `store.Category` and
`store.Transaction`; TC-A03 uses POST; TC-A25 uses the registered email.

**Run 2 (~09:40): 31 PASS · 4 FAIL** · exit 1. Evidence:
`docs/evidence/2026-10-30-stage-a-run2.log`. Newly pinpointed: (4) filtered-list SQL bug —
`"$" + argPlaceholder(n)` → `$$n` → PG 42601 → 500 (server log:
`docs/evidence/2026-10-30-server-diagnosis.log`); (5) `mapDBError` missed 23001
`restrict_violation` → 500 instead of 409; (6) TC-A10 code mismatch (`TYPE_CONFLICT` vs
`VALIDATION_ERROR`); (7) TC-A23(d)/design.md:174 expected 23503, RESTRICT raises 23001.
**Additional critical defect found while diagnosing (not a test failure):** the server
panicked at startup whenever the SPA route registered (`GET /` vs `/api/` ServeMux
conflict) — `go run ./cmd/server` from `backend/` always panicked, blocking D37/TC-M17;
explicit `STATIC_DIR` was also accepted without an existence check.

**Approved fixes #2 (prompt #25) — A14:** removed the literal `$` at all 4 filter sites in
`store/transactions.go` (grep found 4, not 6; behavior unchanged). **A06:** `mapDBError`
now maps 23001 together with 23503 to 409 `CATEGORY_IN_USE`. **A10 (user ruling):
`TYPE_CONFLICT` is canonical** — test updated to expect it; `TYPE_CONFLICT` added to the
design error-envelope list; service unchanged. **A23(d):** test expects `23001`; catalogue
and design.md:174 corrected to 23001. **SPA startup:** router split into an API mux (owns
`/api/`, fallback = 404 envelope — unknown API routes can never reach the SPA) plus a root
mux serving the SPA for GET/HEAD (other methods keep the 404 envelope); `resolveStaticDir`
now existence-checks an explicit `STATIC_DIR` and `config.Load` fails with a clear
startup error. gofmt clean; `go build ./...` OK; `go vet ./...` OK.

**Run 3 (~10:10) — after all fixes: 35 PASS · 0 FAIL · 0 blocked** · **exit 0** · 17.3 s.
Evidence: `docs/evidence/2026-10-30-stage-a-run3.log`.

**Production server verification (~10:15):** started with the README command
(`cd backend && go run ./cmd/server`): no panic; SPA root 200; static asset 200
(text/css); client-route fallback `/budgets` → 200 index.html; valid API route 401
unauthenticated; unknown API route → 404 JSON envelope (never the SPA);
`STATIC_DIR=./definitely-missing-dir` → clear startup error, exit 1. Full transcript:
`docs/evidence/2026-10-30-prod-server-verification.log`. `pebt_dev` was connected but
never written to during verification.

---

# Stage B — Manual (17 cases)

Execution fields (executed-at, actual result, pass/fail/blocked, evidence path, defect/retest)
are recorded **only when the test is actually run**, in the record section below each case.
Evidence files go to `docs/evidence/`.

| ID | Scenario | Traces | Preconditions | Steps (summary) | Expected | Status |
|---|---|---|---|---|---|---|
| TC-M01 | Weak password blocked at registration | AC3 · US1 | register page open | type a short password, submit | inline policy error; no account created | PASS |
| TC-M02 | Duplicate email error toast | AC15 · US1 | user A exists | register again with same email | clear error toast/message | PASS |
| TC-M03 | Wrong password generic error | AC1 · AC15 · US2 | user A exists | login with wrong password | generic error, no hint which field was wrong | PASS |
| TC-M04 | Logout ends session in browser | FR1.3 · US3 | logged in | click logout; try opening a protected page | redirected to login; protected page inaccessible | PASS |
| TC-M05 | Category CRUD + type filter in UI | FR2 · AC3 · US4 | logged in | create, rename, filter by type, delete an unused category | all operations succeed and display correctly | PASS |
| TC-M06 | In-use category delete blocked with message | AC12 · AC15 · US4 | category with a transaction | attempt delete | 409 message shown; category remains | PASS |
| TC-M07 | Invalid amounts rejected inline | AC3 · AC4 · AC13 · US5 | transaction form open | enter `0`, negative, 3-decimal amount | inline errors; no submit | PASS |
| TC-M08 | Type mismatch blocked in UI | AC3 · AC6 · US5 | transaction form open | pick Income type with an expense category | category list pre-filtered / validation blocks submit | PASS |
| TC-M09 | Filters return expected rows | FR3.5 · US6 | several txs exist | apply type / category / date-range filters | only matching rows shown | PASS |
| TC-M10 | Income-category budget blocked | AC7 · US8 | budget form open | select an income category | not selectable / blocked with message | PASS |
| TC-M11 | Duplicate budget conflict message | AC8 · AC15 · US8 | budget exists | create same category+month+year again | clear duplicate error | PASS |
| TC-M12 | Overspend highlighted + negative remaining | AC11 · US9 | budget 100.00; expenses 150.00 | open budgets page | row highlighted; "Over budget" badge; remaining −50.00 visible | PASS |
| TC-M13 | Dashboard updates immediately after CRUD | AC9 · US11 | dashboard open | add / update / delete a transaction | totals and recent list update after each action | PASS |
| TC-M14 | Second account sees nothing of the first | AC2 · US6 | users A and B, data on both | log in as B; browse transactions/budgets/categories | only B's data visible anywhere | PASS |
| TC-M15 | Success notifications on create/update/delete | AC15 · US4–US8 | logged in | create, update, delete items across forms | success toast shown for each | PASS |
| TC-M16 | Invalid date rejected inline | AC3 · AC5 · US5 | transaction form open | enter an invalid date | inline error; no submit | PASS |
| TC-M17 | Fresh-clone README run | DEL3 · S3 | clean machine/clone | follow README only: setup, migrate, run backend+frontend | app runs end-to-end; README steps accurate | PASS (with qualification) |

---

## Execution records (Stage B)

> Records are appended below in the format:
> `TC-Mxx · executed <date time IST> · result PASS/FAIL/BLOCKED · evidence <path> · defect <id> / retest <result>`
>
> **TC-M01 · executed 2026-10-30 ~12:15 IST (approx.; before the TC-M02 run) · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed: a new, unregistered email with a deliberately short (3-character) password
> produced the inline error "Password must be at least 8 characters"; the form did not
> submit and the page did not navigate away; no account was created. Matches the expected
> inline policy error with no account creation (AC3).
>
> **TC-M02 · executed 2026-10-30 ~12:30 IST · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed: reused the TC-M01 email with a policy-valid password; registration rejected
> with the exact toast "An account with this email already exists"; the app stayed on the
> registration page; no account switch or login occurred. Expected 409 `EMAIL_TAKEN` +
> clear toast — matched.
>
> **TC-M03 · executed 2026-10-30 ~13:16 IST · result PASS · evidence user-provided screenshot (2026-10-30 ~13:16 IST; path not supplied to the record) · defect — / retest —**
> Observed: sign-in attempted with an existing account email and an incorrect password;
> the app displayed the generic toast "Invalid email or password" and remained on the
> login page without authenticating. No hint about which field was wrong — matches the
> expected generic failure (AC1/AC15; consistent with TC-A32's identical-message rule).
>
> **TC-M04 · executed 2026-10-30 ~13:33 IST · result PASS · evidence user-provided screenshot (login page after the protected-route checks; path not supplied to the record) · defect — / retest —**
> Observed: logout succeeded; direct navigation to `http://localhost:5173/` and to
> `http://localhost:5173/budgets` while logged out redirected to `/login` both times; no
> protected page displayed its content. Matches the expected redirect + inaccessibility
> (FR1.3).
>
> **TC-M05 · executed 2026-10-30 ~13:50 IST (approx.) · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed: created `Salary` (Income), `Food` (Expense), `Travel` (Expense) and `Temp Cat`
> (Expense); renamed `Temp Cat` → `Temp Cat Renamed`; Income filter showed only `Salary`,
> Expense filter showed `Food` + `Travel` + `Temp Cat Renamed`, All types showed all four;
> deleted `Temp Cat Renamed` successfully. `Salary`, `Food`, `Travel` remain as the shared
> fixtures for later cases. All operations succeeded and displayed correctly (FR2/AC3).
>
> **TC-M06 · executed 2026-10-30 ~14:10 IST · result PASS · evidence user-provided screenshot (~14:10 IST; error toast + Food still present; path not supplied to the record) · defect — / retest —**
> Observed: created a permanent Expense transaction on `Food` (₹40.00, 2026-10-15,
> "M06 seed"); attempted to delete `Food`; the app showed "Category is referenced by
> transactions or budgets and cannot be deleted" and `Food` remained in the category list
> after the attempt; the transaction was retained. Matches AC12 (delete blocked) + AC15
> (clear message).
>
> **TC-M07 · executed 2026-10-30 (after the ~14:10 TC-M06 run) · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed with Type = Expense, Category = `Travel`, a valid date: amount `0` →
> "Amount must be greater than zero"; `-5` → "Amount must be a positive number (digits
> only)"; `10.999` → "Amount can have at most 2 decimal places". All three blocked inline;
> no transaction submitted or created. Matches AC3/AC4/AC13.
>
> **TC-M08 · executed 2026-10-30 ~14:35 IST · result PASS · evidence user-provided screenshot (~14:35 IST; path not supplied to the record) · defect — / retest —**
> Observed: with Type = Income, the Category dropdown offered only `Salary`; `Food` and
> `Travel` (expense categories) were excluded (pre-filter, AC6). Submitting with category
> unselected showed "Choose a category" (and "Amount is required" for the empty amount);
> no transaction was submitted. Matches AC3/AC6.
>
> **TC-M09 · executed 2026-10-30 ~14:50 IST (approx.) · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed: with 3 transactions seeded (Salary income 2026-10-01; Food expense 40.00 and
> Travel expense 18.00, both 2026-10-15), the Type, Category, From and To date filters and
> Clear Filters all worked; each filter returned exactly the expected subset and the empty
> date range (2026-10-16 → 2026-10-31) returned no rows. Matches FR3.5.
>
> **TC-M10 · executed 2026-10-30 (after the ~14:50 TC-M09 run) · result PASS · evidence user-provided screenshot (taken during this manual session; path not supplied to the record) · defect — / retest —**
> Observed: on the Budgets page the "Category (expense only)" dropdown displayed only
> `Food` and `Travel`; `Salary` (Income) was not offered. No budget was created or saved.
> Matches AC7 (budgets on expense categories only, blocked at the UI).
>
> **TC-M11 · executed 2026-10-30 · result PASS · evidence user-observed browser result (screenshot path not supplied) · defect — / retest —**
> Observed: the `Food` budget for November 2026 at ₹100.00 was visible. Attempting to
> create the same category/month/year budget again displayed the exact message "A budget
> for this category, month and year already exists"; the original budget remained visible;
> no duplicate was created. Matches AC8/AC15.
>
> **TC-M12 · executed 2026-10-30 · result PASS · evidence user-observed browser result (screenshot path not supplied) · defect — / retest —**
> Observed: Food budget for November 2026 was ₹100.00. Two Food expenses of ₹75.00 each
> were added for 10 November and 20 November 2026. The Budgets page showed Actual
> ₹150.00, Remaining −₹50.00, the "Over budget" status, and visual row highlighting.
> Matches AC11.
>
> **TC-M13 · executed 2026-10-30 · result PASS · evidence user-observed browser results (screenshot paths not supplied) · defect — / retest —**
> Observed: starting October 2026 dashboard totals matched (income ₹3,000.00, expenses
> ₹58.00, remaining ₹2,942.00). Added a temporary Travel expense ₹5.00 dated 2026-10-20
> ("M13 dashboard test") → dashboard updated immediately to expenses ₹63.00 / remaining
> ₹2,937.00 without manual reload; edited it to ₹7.00 → ₹65.00 / ₹2,935.00; deleted only
> that temporary transaction → returned immediately to ₹58.00 / ₹2,942.00. Totals and
> recent list updated after each action (AC9).
>
> **TC-M15 · executed 2026-10-30 · result PASS · evidence user-observed browser results (screenshot paths not supplied) · defect — / retest —**
> Observed: all six successful actions displayed the expected success notifications —
> category created (`Toast Cat`), category updated (renamed to `Toast Cat 2`), category
> deleted (`Toast Cat 2`), transaction added (Travel expense ₹2.00, 2026-10-25, "M15
> toast test"), transaction updated (₹3.00), transaction deleted (same temporary
> transaction). All temporary test items were removed; no original categories or
> transactions were deleted. Matches AC15.
>
> **TC-M16 · executed 2026-10-30 · result PASS · evidence user-observed browser result (screenshot path not supplied) · defect — / retest —**
> Observed: with the transaction form open and the Date field empty, submitting displayed
> the inline validation message "Date is required" directly below the Date field;
> submission was blocked and no transaction was created. (As noted in the guide, the
> native `type="date"` control prevents typing impossible dates like 2026-02-30, so the
> empty-date path is the reachable invalid state in a browser; the backend rejection of
> impossible dates is separately proven by TC-A08.) Matches AC3/AC5.
>
> **TC-M14 · executed 2026-10-30 · result PASS · evidence user-observed browser results (screenshot paths not supplied) · defect — / retest —**
> Observed: registered a separate User B account and created B-only data (category "B
> Only", Expense; transaction "B account test", Expense, ₹9.00). As User B: Transactions
> showed only B's ₹9.00 transaction; Categories showed only "B Only" (A's Salary, Food
> and Travel not visible); Budgets showed none of A's budgets, including Food/November
> 2026. Logged back in as User A: the original Salary, Food and Travel categories remained
> visible and intact. Only B's data was visible anywhere as B — matches AC2.
>
> **TC-M17 · executed 2026-10-30 ~19:40–19:55 IST · result PASS (with qualification) · evidence user-observed command results recorded by Claude during the run; full details in prompt.log #27 · defect — / retest —**
> Observed: a fresh `git clone` of `https://github.com/Druthi557/ExpenseandBudgetTracker`
> into a temporary directory **matched the published `main` commit exactly**
> (`eb50b418b3ef3792fb0826dfb6c209285ef263f5`) and contained the expected README,
> `.env.example`, migrations, docs and source. Following the README from scratch:
> frontend dependencies installed successfully (66 packages, isolated to the clone);
> `npm run build` succeeded; the backend started clean on :8080 and health checks
> succeeded (SPA root 200, API auth-gated 401, register 201 with cookie, login 200);
> **all 35 automated tests passed** (`ok pebt/backend/tests`); **fresh migration
> application was verified through the test suite's schema reset** (drop-all + re-apply
> from empty, per the suite's documented behavior); the frontend development server and
> its `/api` proxy worked (Vite ready; page 200; proxied API 401).
>
> **Qualification (recorded honestly):** the README's literal database-creation commands
> (`psql` role/database creation) were **not independently executed** — PostgreSQL
> superuser credentials were unavailable in the session. Per explicit user authorization,
> the **designated disposable test database was used instead**, identity-verified before
> use; the application was pointed at it for startup/health checks and the automated
> suite for migration verification. The README instructions themselves are correct and
> complete; this was an environment-privilege limitation of the test session, not a
> documentation defect. No original-project data was used or modified; the original
> `.env` was never copied or displayed (a fresh secret was generated for the clone).

---

## Coverage summary

- 35 automated + 17 manual = **52 planned cases — all 52 executed, all PASS**
  (Stage A run 3 + Stage B complete, 2026-10-30; TC-M17 carries a recorded qualification
  regarding database-creation privileges). Full histories: Stage A (three runs, defects
  and fixes) and Stage B (per-case records) above.
- Every AC1–AC16 is covered by at least one case (see `docs/traceability.md`).
- AC3/AC15 frontend behavior is intentionally verified manually (no frontend test framework in scope).
- CON1–4 are verified by the implementation itself + version pins recorded in README.
