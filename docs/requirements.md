# Requirements Baseline

**Source of truth:** `druthi 1.docx` (read and verified in full on 2026-10-27, re-verified
2026-10-28 and 2026-10-29; 3,422 extracted text characters; no headers/footers/images in the file).
Everything in §1–§3 is **[Confirmed]** — quoted/derived directly from the document.
§4 lists what the document deliberately does not say. Nothing here is invented.

## 1. Functional requirements

**FR1 — User Registration and Authentication**
- FR1.1 Registration page/flow · FR1.2 Login · FR1.3 Logout
- FR1.4 Every user must have a separate account
- FR1.5 Users must only access their own financial records

**FR2 — Category Management**
- FR2.1 Create · FR2.2 View all · FR2.3 Update · FR2.4 Delete categories
- FR2.5 Each category has a **name** and a **type** (Income or Expense)
  (document examples: Income — Salary, Freelance; Expense — Food, Travel, Shopping)
- FR2.6 Users can view all categories, filtered by type

**FR3 — Income and Expense Management**
- FR3.1 Add income and expense transactions
- FR3.2 Each transaction contains: amount, category, date (required); description (optional)
- FR3.3 View, update, and delete transactions
- FR3.4 Display transactions in a list ordered by date
- FR3.5 Filter by transaction type, category, and date range

**FR4 — Monthly Budget Management**
- FR4.1 Create monthly budgets for **expense** categories
- FR4.2 Each budget contains: category, month, year, budget amount
- FR4.3 Update · FR4.4 Delete budgets
- FR4.5 Display budget amount, actual expense amount, and remaining balance
- FR4.6 Highlight categories where spending exceeds the allocated budget

**FR5 — Dashboard (selected month)**
- FR5.1 Total income · FR5.2 Total expenses
- FR5.3 Remaining balance (Income − Expenses)
- FR5.4 Expense totals grouped by category
- FR5.5 Five most recent transactions

## 2. Acceptance criteria (16, in document order)

| ID | Criterion |
|---|---|
| AC1 | Registration and login use secure password storage and token-based authentication |
| AC2 | Users must not be able to access another user's transactions or budgets |
| AC3 | Frontend validation on all relevant forms |
| AC4 | Transaction and budget amounts must be greater than zero |
| AC5 | All transaction dates must be valid |
| AC6 | A transaction must use a category matching its transaction type |
| AC7 | Only expense categories can be assigned monthly budgets |
| AC8 | Duplicate budgets for the same category, month, and year are not allowed |
| AC9 | Dashboard calculations reflect newly added, updated, and deleted transactions |
| AC10 | Monthly budgets are calculated from expenses recorded within the corresponding calendar month |
| AC11 | Remaining budget may become negative; the UI must clearly indicate overspending |
| AC12 | Categories associated with existing transactions or budgets cannot be deleted |
| AC13 | Monetary calculations use an appropriate decimal representation instead of floating-point |
| AC14 | All APIs implement backend validation and meaningful HTTP error responses |
| AC15 | Users receive clear success and error notifications |
| AC16 | All application data is stored persistently in PostgreSQL |

*(Note: an earlier conversational summary once counted "15 items" — a miscount of its own list.
The document contains 16 acceptance criteria; the enumeration above matches the document verbatim.)*

## 3. Constraints and deliverables

**Technology constraints (mandatory):**
| ID | Constraint |
|---|---|
| CON1 | React.js frontend with functional components and hooks |
| CON2 | Go backend with REST APIs |
| CON3 | PostgreSQL database with appropriate relationships and constraints |
| CON4 | JWT-based authentication and hashed passwords |

**Submission deliverables:**
| ID | Deliverable |
|---|---|
| DEL1 | GitHub URL for source code containing the React frontend and Go backend |
| DEL2 | PostgreSQL schema **or** database migrations |
| DEL3 | README with installation and execution instructions |

**Expected outcome (document's words):** a working financial tracking application demonstrating
secure authentication, CRUD APIs, database relationships, date-based filtering, aggregation
queries, and budget calculation logic.

## 4. Non-functional requirements

The document states **no separate non-functional requirements** (no performance, scalability,
browser-support, or test-coverage targets). Security, validation, numeric correctness,
persistence, and UX feedback are embedded in the ACs above. Phase 1 recorded this as an open
question; the working default is: correctness and security first, no formal performance targets,
verification by AC walkthrough. No NFR has been invented.

## 5. Documented silences & ambiguities (surfaced, not silently resolved)

| # | The document does not say | Resolved (as proposal/ruling) in |
|---|---|---|
| 1 | Login identity field (email vs username); password policy | `docs/decisions.md` D7, D8 |
| 2 | Whether category names must be unique (per user), and scope | D18 |
| 3 | Whether a category's Income/Expense type can change later | D27 |
| 4 | Whether transactions store their own type explicitly | docs/design.md §2 (explicit `type` column) |
| 5 | Transaction ordering direction and same-date tie-break | D12 |
| 6 | Whether dashboard recent-5 are within the selected month or overall (bullets 1–3 say "selected month"; bullets 4–5 don't) | D13 — recent-5 overall; by-category = selected month |
| 7 | Date-range filter input format | docs/design.md §4 (ISO `YYYY-MM-DD` params) |
| 8 | JWT expiry, refresh, storage mechanism | D9, D10 (user-ruled), docs/design.md §6 |
| 9 | Currency and display format | D11 (INR display proposal) |
| 10 | Go framework/router, ORM, migration tool; test framework | D20–D22 |

## 6. Out of scope

Per the approved Phase 1 plan: recurring transactions, multi-currency, export/reporting,
email verification, mobile apps, cloud deployment. Deadline: none set (Phase 1 Q1).
