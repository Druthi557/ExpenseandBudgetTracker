# CLAUDE.md — Project Charter & Control File

**Project:** Personal Expense and Budget Tracker (SDLC student assignment)
**Location:** `C:\Users\Druthi D\ExpenseandBudgetTracker`
**Last updated:** 2026-10-30

---

## 1. Purpose & scope

Develop a web-based Personal Expense and Budget Tracker that lets users record income and
expenses, organize transactions into categories, set monthly budgets, and monitor spending.
Expected outcome (assignment's own words): a working application demonstrating secure
authentication, CRUD APIs, database relationships, date-based filtering, aggregation queries,
and budget calculation logic.

Out of scope (Phase 1 proposal, unchallenged since): recurring transactions, multi-currency,
export/reporting, email verification, mobile apps, cloud deployment.

## 2. Source-of-truth rule (mandatory)

`druthi 1.docx` in this folder is the **single authoritative assignment document**.
- Every requirement, technology choice, validation rule, and deliverable must be traceable to it.
- The assignment takes precedence over assumptions, general knowledge, and unrelated project memory.
- Unspecified details are open questions or clearly labeled proposals — never presented as requirements.
- Conflicts or ambiguities are surfaced for user approval, not silently resolved.

## 3. Mandatory technology stack (doc-mandated)

| ID | Layer | Requirement |
|----|-------|-------------|
| CON1 | Frontend | React.js, functional components + hooks only |
| CON2 | Backend | Go with REST APIs |
| CON3 | Database | PostgreSQL with appropriate relationships and constraints |
| CON4 | Auth | JWT-based authentication and hashed passwords |

Submission deliverables: **DEL1** GitHub URL with React frontend + Go backend ·
**DEL2** PostgreSQL schema or migrations · **DEL3** README with installation/execution instructions.

## 4. Confirmed requirements & acceptance criteria

Authoritative detail: [`docs/requirements.md`](docs/requirements.md).
Summary: FR1 auth (register/login/logout, separate accounts, own-records-only) ·
FR2 categories (CRUD, name+type, filter by type) · FR3 transactions (CRUD, amount/category/date/
optional description, ordered by date, filters) · FR4 monthly budgets (expense categories,
budget/actual/remaining, overspend highlighted) · FR5 dashboard (5 aggregates, selected month).

**AC1** secure password storage + token auth · **AC2** no cross-user access · **AC3** frontend
validation on all forms · **AC4** amounts > 0 · **AC5** valid dates · **AC6** transaction type must
match category type · **AC7** budgets on expense categories only · **AC8** no duplicate budgets
(category+month+year) · **AC9** dashboard reflects add/update/delete immediately · **AC10** budgets
from calendar-month expenses · **AC11** remaining may go negative; overspending clearly indicated ·
**AC12** categories in use cannot be deleted · **AC13** decimal, never floating-point money ·
**AC14** backend validation + meaningful HTTP errors · **AC15** clear success/error notifications ·
**AC16** all data persistent in PostgreSQL.

## 5. Security, validation & correctness rules (non-negotiable in implementation)

1. **Per-user isolation:** `user_id` comes only from the authenticated session (JWT claim), never
   from request body/URL. Composite foreign keys `(category_id, user_id, type) →
   categories(id, user_id, type)` make cross-user references impossible at the DB level.
   Foreign resources return 404 (no existence leak).
2. **Type integrity:** transaction type = category type and budget → expense category only,
   enforced by composite FKs (AC6/AC7) + friendly 422 from the API.
3. **Referential safety:** both category-referencing FKs are `ON DELETE RESTRICT` (AC12);
   API maps the violation to 409 `CATEGORY_IN_USE`.
4. **Money:** PostgreSQL `NUMERIC(12,2)`; Go `shopspring/decimal`; **no float32/float64 anywhere
   in the money path**; JSON wire format = strings with exactly 2 dp (`"123.45"`), produced by
   explicit `StringFixed(2)` DTO fields; inputs with >2 dp rejected 422 before any rounding.
5. **Auth:** bcrypt (cost 12) hashes; JWT HS256, claims `sub`/`iat`/`exp`, 60-minute expiry,
   secret from env var only; token delivered in an **HttpOnly, Secure, SameSite=Lax cookie**
   (`pebt_token`); logout clears the cookie server-side; mutating requests require the
   `X-Requested-With: XMLHttpRequest` header (CSRF guard, else 403). Known limitation: a JWT
   remains technically valid until expiry after logout — accepted and documented (stateless design).
6. **Validation:** every field validated on the frontend (AC3) AND backend (AC14). One status
   policy: 400 malformed JSON · 401 auth · 403 CSRF · 404 missing/foreign · 409 state conflicts ·
   422 field/semantic validation · 500 unexpected (no internals leaked).
7. **No silent data changes:** amounts are never rounded implicitly; dates are strict `DATE`.

## 6. Coding conventions

- React: functional components + hooks only (CON1); plain `fetch` wrapper adding
  `credentials: 'same-origin'` and the CSRF header; toasts for success/error (AC15);
  refetch lists + dashboard after every mutation (AC9).
- Go: stdlib `net/http` router (Go 1.22+); `pgx/v5` driver; explicit SQL, no ORM;
  layered handlers → services → repository; every query scoped `WHERE user_id = $1`.
- Migrations: `golang-migrate`, plain SQL up/down in `/migrations`.
- Money formatting only via the decimal helpers — never `float`.

## 7. Testing expectations

- Automated: Go `httptest` + dedicated PostgreSQL test DB built from migrations.
  Catalogue: [`test_cases.md`](test_cases.md) Stage A (TC-A01–A35).
- Manual: Stage B records (TC-M01–M17) created only when actually executed, with evidence
  stored in `docs/evidence/`.
- **Anti-fabrication rule:** a test may be marked Passed only after real execution with recorded
  evidence. Planned tests stay "Not Run". Never invent results, dates, or defects.

## 8. Documentation & traceability map

| File | Purpose |
|---|---|
| `CLAUDE.md` (this file) | Charter, rules, phase status, approval log |
| `prompt.log` | Append-only record of user prompts |
| `README.md` | Install/run/testing instructions + submission steps (draft until Phase 3) |
| `user_stories.md` | US1–US11 (+US-P1/P2), traceable to FRs/ACs |
| `test_cases.md` | Stage A automated + Stage B manual catalogue, execution records |
| `docs/requirements.md` | FR/AC/CON/DEL baseline + documented ambiguities |
| `docs/design.md` | Architecture, DB schema, API contract, validation matrix, security |
| `docs/decisions.md` | Decision register + risks + open questions/assumptions |
| `docs/traceability.md` | RTM: requirement → story → design → component → test |
| `docs/evidence/` | Screenshots/logs referenced by executed test records |

Traceability rule: IDs are stable across documents — FR1–FR5, AC1–AC16, CON1–4, DEL1–3,
US1–US11, TC-A01–A35, TC-M01–M17, D1–D37. The RTM is updated whenever design or tests change.
Approved decisions are never silently changed here or in `docs/decisions.md`.

## 9. SDLC phases & approval gates

| Phase | Status (2026-10-30) | Exit requires |
|---|---|---|
| 1 — Planning | **COMPLETE — approved 2026-10-27** (see Approval Log) | user approval ✔ |
| 2 — Requirements & Design | **APPROVED 2026-10-29** (delegated decision; rationale in Approval Log §11) | review done ✔ |
| 3 — Implementation | **IN PROGRESS** — authorized in a prior session (its prompt is unrecoverable; noted in `prompt.log` #19); resumed 2026-10-30 after a 429 interruption. Backend + frontend code complete; automated suite NOT yet executed (needs the dev/test databases) | code complete + Stage A suite executed and recorded |
| 4 — Verification | Not started | Phase 3 complete |
| 5 — Submission prep | Not started | Phase 4 passed |
| 6 — Closure | Not started | Phase 5 complete |

**Gate rule:** completion of one phase is NOT authorization for the next. No commits or
pushes before the user asks for them (DEL1's GitHub URL is a Phase 5 deliverable).

**Permitted right now:** Phase 3 implementation work (code, migrations-as-embedded-files,
README finalization, test execution against `TEST_DATABASE_URL`); destructive database
commands (`migrate reset`/`down`, schema drops) only against the disposable test database
or with explicit user approval.

## 10. Current status block

- 2026-10-27: assignment doc located and fully read (3,422 text chars; verified again 2026-10-29).
  Phase 1 plan presented and **approved** with Q1–Q8 clarifications (see Approval Log).
- 2026-10-28: Phase 2 drafted (two comprehensive passes; consolidated). Document re-verified.
- 2026-10-29: Phase 2 verification completed (test coverage, RTM gaps closed, inconsistencies
  resolved); unified v2.1 package produced with user rulings D9/D10/D14; documentation pack
  created on disk with user authorization (creation ≠ approval).
- 2026-10-29 (later): **Final Phase 2 review of the saved documents against `druthi 1.docx` —
  Phase 2 APPROVED** (delegated decision; evidence + rationale in Approval Log §11 and
  docs/decisions.md status note). Decision register D1–D38 ratified; **all 52 tests remain
  Not Run**; **Phase 3 remains BLOCKED** pending separate user authorization.
- Approved decisions so far: doc-mandated requirements + Phase 1 plan (2026-10-27) + **Phase 2
  decision register D1–D38 (2026-10-29)** — rulings D9/D10/D14 by the user, all other rows
  ratified as approved defaults. `docs/decisions.md` is the authoritative register.
- Open questions: none blocking implementation. Assumptions listed in `docs/decisions.md` §3.
- 2026-10-30 (resume session after 429): interrupted edits to `backend/internal/config/config.go`
  and `backend/cmd/migrate/main.go` verified complete and clean (no duplicates); `gofmt`
  applied to three files that needed it; `go build ./...` and `go vet ./...` pass; frontend
  production build passes. Fixed one defect: Budgets page did not refetch after creating a
  budget for the already-selected month (AC9). README finalized with verified versions.
  Stage A suite (TC-A01–A35) written for all 35 cases but **Not Run** — the `pebt` role and
  `pebt_dev`/`pebt_test` databases do not exist yet; `scripts/setup-db.ps1` added for setup
  (requires the user's postgres superuser password). **No test has been executed.**
- 2026-10-30 (later, user-approved steps): `scripts/setup-db.ps1` run by the user — `pebt`
  role (LOGIN only, no SUPERUSER/CREATEDB/CREATEROLE) + `pebt_dev`/`pebt_test` created;
  read-only verification passed against .env and design. Exposed-in-transcript JWT_SECRET
  rotated in `.env` (value never displayed; other .env lines checksum-identical). Migration
  000001 applied to `pebt_dev` (version=1, dirty=false; verified read-only). **Stage A suite
  executed against `pebt_test`: 14 PASS (4 unreliable) · 21 FAIL · 0 skipped** — recorded
  per-row in `test_cases.md` with evidence `docs/evidence/2026-10-30-stage-a-run.log`.
  Three root causes identified (1 API: `store.Category`/`store.Transaction` missing JSON
  tags → wire keys `{"ID":…}` + `category_id=0` cascade; 2 test: TC-A03 uses DELETE for the
  POST-only logout route; 3 test: TC-A25 double `freshEmail`). **Fixes NOT applied —**
  awaiting user approval.
- 2026-10-30 (later, prompt #24): the three approved fixes applied (`json` tags on
  `store.Category`/`store.Transaction`; TC-A03 POST; TC-A25 email source). gofmt/build/vet
  clean. **Stage A run 2: 31 PASS · 4 FAIL · 0 skipped** (evidence
  `docs/evidence/2026-10-30-stage-a-run2.log`). Remaining failures precisely diagnosed and
  recorded, **fixes NOT applied — awaiting approval**: (1) filtered-list SQL bug —
  `"$" + argPlaceholder(n)` yields `$$n` → PG 42601 → 500 on every filtered GET (TC-A14);
  (2) `mapDBError` misses 23001 `restrict_violation` → 500 instead of 409
  `CATEGORY_IN_USE` (TC-A06); (3) TC-A10 code mismatch — service `TYPE_CONFLICT` vs test
  `VALIDATION_ERROR`, design row 14 specifies no code (needs ruling + design envelope list
  omits TYPE_CONFLICT); (4) TC-A23(d)/design.md:174 expect 23503 but RESTRICT deletes raise
  23001. **Additional critical defect (diagnosis evidence:
  `docs/evidence/2026-10-30-server-diagnosis.log`): the server panics at startup whenever
  the SPA route registers (`GET /` vs `/api/` ServeMux conflict) — with `frontend/dist`
  auto-detected, `go run ./cmd/server` always panics → D37 production topology and TC-M17
  are currently unrunnable.**
- 2026-10-30 (later, prompt #25): second approved fix round applied — SQL placeholder
  bug fixed (4 sites in `store/transactions.go`), `mapDBError` maps 23001 → 409
  `CATEGORY_IN_USE`, TC-A10 canonicalized to `TYPE_CONFLICT` (user ruling; design envelope
  list amended), TC-A23(d) corrected to 23001 (test + catalogue + design.md:174), SPA
  startup panic fixed via two-mux router (API mux owns `/api/` with 404-envelope fallback;
  SPA on root for GET/HEAD only), explicit `STATIC_DIR` now existence-checked with a clear
  startup error. gofmt/build/vet clean. **Stage A run 3: 35 PASS · 0 FAIL · exit 0**
  (`docs/evidence/2026-10-30-stage-a-run3.log`). **Production server verified with the
  README command**: SPA root/assets/client-route fallback 200, API 401 without auth,
  unknown API route → 404 envelope (never SPA), invalid STATIC_DIR → clear startup error
  (`docs/evidence/2026-10-30-prod-server-verification.log`); `pebt_dev` not written to.
- Next permitted action: Phase 3 remains open for its remaining work — Stage B manual
  checks (TC-M01–M17, needs the app running in a browser), README fresh-clone validation
  (TC-M17), and the user's decision on git init/GitHub (DEL1) when they choose.

## 11. Approval Log

| Date | Event | Evidence |
|---|---|---|
| 2026-10-27 | **Phase 1 — Planning APPROVED** by user, with Q1–Q8 clarifications ("I have reviewed the SDLC Phase 1 planning report and approve the overall plan…"). Approved in Claude Code session 11078d54; recovered verbatim 2026-10-29 into `prompt.log` entry #4. Timestamp 2026-10-27 23:58 IST (session metadata, approximate). | `prompt.log` #4; session transcript 11078d54 |
| 2026-10-28/29 | Phase 2 drafted and revised in conversation; user rulings on D9 (JWT 60-min + cookie-clearing logout), D10 (HttpOnly cookie + CSRF), D14 (money strings 2 dp) given in session 315305fa. **Phase 2 itself remains unapproved.** | `prompt.log` #15; `docs/decisions.md` |
| 2026-10-29 | User authorized creation of planning/design documentation files (this pack) — explicitly **not** a Phase 2 approval and **not** Phase 3 authorization. | `prompt.log` #17 |
| 2026-10-29 | **Phase 2 — Requirements & Design APPROVED.** The user delegated the decision: "You have my authorization to make the genuine approval decision based on the evidence." Final review performed on the saved files against `druthi 1.docx`: all 5 FRs (with sub-points), **AC1–AC16**, CON1–4, and DEL1–3 map end-to-end in `docs/traceability.md`; DB-level integrity enforces AC2/AC4/AC5/AC6/AC7/AC8/AC12 (composite FKs, CHECKs, UNIQUE, RESTRICT); security design reviewed sound (bcrypt cost 12, HS256 60-min JWT in HttpOnly+Secure+SameSite=Lax cookie, CSRF header guard, 404 existence masking, exact-decimal money with >2 dp rejected 422); 52 planned tests cover every AC (RTM references all resolve — verified programmatically); status policy consistent across design and tests. **One defect found and fixed during review:** stale decision reference `D10-edit` in docs/design.md (budget-edit behavior) — corrected and registered as **D38**. No genuine blockers. Design approval only — **no test has been executed** and **Phase 3 remains BLOCKED**. | This review session; `docs/traceability.md`; `docs/decisions.md` status note; `prompt.log` (delegation prompt, entry #19 to be appended) |
