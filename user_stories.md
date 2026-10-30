# User Stories — Personal Expense and Budget Tracker

Status legend: **Planned** = derived from confirmed assignment requirements; implementation has
not started (Phase 3 blocked). All stories trace to `druthi 1.docx` via the FR/AC IDs in
`docs/requirements.md` and the design in `docs/design.md`.

---

## US1 — Register an account
- **Story:** As a new user, I want to create an account with my email and a password, so that I can keep my finances private and secure.
- **Acceptance criteria:** valid email + policy-compliant password required; duplicate email rejected with a clear error; on success the user is logged in (session cookie set).
- **Traces:** FR1.1 · AC1, AC3, AC14, AC15
- **Dependencies:** none
- **Status:** Planned (Phase 3)

## US2 — Log in
- **Story:** As a registered user, I want to log in with my email and password, so that I can access my own financial data.
- **Acceptance criteria:** correct credentials issue a session; wrong email or wrong password produce the same generic error (no account enumeration).
- **Traces:** FR1.2 · AC1, AC14, AC15
- **Dependencies:** US1
- **Status:** Planned (Phase 3)

## US3 — Log out
- **Story:** As a logged-in user, I want to log out, so that my session ends on the browser I am using.
- **Acceptance criteria:** logout clears the auth cookie; protected pages become inaccessible until re-login. (Known limitation: token technically valid until its 60-min expiry — see docs/design.md §6.)
- **Traces:** FR1.3 · AC1, AC15
- **Dependencies:** US2
- **Status:** Planned (Phase 3)

## US4 — Manage categories
- **Story:** As a logged-in user, I want to create, view, rename, and delete my income/expense categories and filter them by type, so that I can organize my finances my way.
- **Acceptance criteria:** name (1–100 chars) + type (Income|Expense) required; duplicate names rejected; categories referenced by transactions or budgets cannot be deleted (clear error).
- **Traces:** FR2.1–FR2.6 · AC3, AC6 (indirect), AC12, AC14, AC15
- **Dependencies:** US2
- **Status:** Planned (Phase 3)

## US5 — Add an income or expense transaction
- **Story:** As a logged-in user, I want to record an income or expense with amount, category, date, and an optional description, so that my spending and income are tracked.
- **Acceptance criteria:** amount > 0 with at most 2 decimals; valid calendar date; category must match the transaction type; frontend inline validation + backend re-validation.
- **Traces:** FR3.1–FR3.2 · AC3, AC4, AC5, AC6, AC13, AC14, AC15
- **Dependencies:** US4
- **Status:** Planned (Phase 3)

## US6 — View and filter transactions
- **Story:** As a logged-in user, I want my transactions listed newest-first with filters for type, category, and date range, so that I can find and review entries quickly.
- **Acceptance criteria:** ordering newest-first with stable same-date tie-break; each filter returns exactly the matching rows; invalid filter inputs get clear errors.
- **Traces:** FR3.3–FR3.5 · AC2, AC14
- **Dependencies:** US5
- **Status:** Planned (Phase 3)

## US7 — Edit or delete a transaction
- **Story:** As a logged-in user, I want to update or delete a transaction, so that I can correct mistakes.
- **Acceptance criteria:** every field re-validated on update (including type ↔ category consistency after a change); deletion immediate; other users' transactions are invisible.
- **Traces:** FR3.3 · AC4, AC5, AC6, AC14
- **Dependencies:** US5
- **Status:** Planned (Phase 3)

## US8 — Create monthly budgets
- **Story:** As a logged-in user, I want to set a monthly budget amount for an expense category, so that I can plan my spending.
- **Acceptance criteria:** budgets only on expense categories; one budget per category+month+year (duplicates rejected); amount > 0; month/year valid.
- **Traces:** FR4.1–FR4.2 · AC4, AC7, AC8, AC14, AC15
- **Dependencies:** US4
- **Status:** Planned (Phase 3)

## US9 — See budget vs. actual with overspend warning
- **Story:** As a logged-in user, I want to see each budget's amount, actual spending, and remaining balance — with overspending clearly flagged — so that I know where I stand.
- **Acceptance criteria:** actual = expenses in the calendar month of the budget; remaining may be negative; overspend rows visually highlighted.
- **Traces:** FR4.5–FR4.6 · AC10, AC11
- **Dependencies:** US5, US8
- **Status:** Planned (Phase 3)

## US10 — Edit or delete budgets
- **Story:** As a logged-in user, I want to update a budget's amount or remove it, so that my plans stay current.
- **Acceptance criteria:** amount editable (re-validated); category/month/year changes = delete + recreate; deletion immediate.
- **Traces:** FR4.3–FR4.4 · AC4, AC14
- **Dependencies:** US8
- **Status:** Planned (Phase 3)

## US11 — Dashboard for a selected month
- **Story:** As a logged-in user, I want a dashboard showing total income, total expenses, remaining balance, expense totals by category, and my five most recent transactions, so that I can monitor my month at a glance.
- **Acceptance criteria:** figures recompute live after every transaction add/update/delete; recent-5 are overall (any month) per decision D13; by-category totals cover the selected month.
- **Traces:** FR5.1–FR5.5 · AC9
- **Dependencies:** US5, US8
- **Status:** Planned (Phase 3)

---

## Proposed enhancements — out of scope unless separately approved

| ID | Story | Traces | Status |
|---|---|---|---|
| US-P1 | Pagination controls on long lists | D23 (decision: none in v1) | Not planned (default: no pagination) |
| US-P2 | `/auth/me` profile display | D32 (decision: omit) | Not planned |
