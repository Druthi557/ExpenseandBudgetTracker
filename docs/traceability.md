# Requirements Traceability Matrix (RTM)

Maps every assignment item through: **requirement → user story → design element →
implementation component → test cases**. IDs are stable across all documents.
Design references point into `docs/design.md`; tests into `test_cases.md`.
Notation: within a cell, the `TC-` prefix applies to the whole list — e.g.
"TC-A01–A04, A30–A32" means TC-A01…TC-A04 and TC-A30…TC-A32.

| Assignment item | User story | Design element | Implementation component | Test cases |
|---|---|---|---|---|
| FR1.1–1.3 register/login/logout | US1–US3 | §5 auth; API 1–3 | Go auth handlers + middleware; React auth pages | TC-A01–A04, A30–A32, M01–M04 |
| FR1.4 separate accounts | US1 | §2 users; user_id keying | users table + scoping | TC-A01, A33, M14 |
| FR1.5 own records only | US2–US11 | §2 composite FKs; §5 scoping | middleware + repository `WHERE user_id` | TC-A05, A17, A23, A33, M14 |
| FR2.1–2.4 category CRUD | US4 | §2 categories; API 4–7 | category handler/repo; categories UI | TC-A13, A06, M05, M06 |
| FR2.5 name + type | US4 | §2 columns + validation | validators | TC-A13, A26, M05 |
| FR2.6 filter by type | US4 | API 4 | `?type=` filter | TC-A27, M05 |
| FR3.1–3.3 transaction CRUD | US5–US7 | §2 transactions; API 8–12 | tx handler/repo; transactions UI | TC-A15, A16, A17, A05, M07, M08 |
| FR3.4 ordered by date | US6 | §2 index + ordering (D12) | list query | TC-A14, A21, M09 |
| FR3.5 type/category/date-range filters | US6 | API 8 | filter params | TC-A14, A28, M09 |
| FR4.1–4.2 create budgets (expense cats) | US8 | §2 budgets + composite FK | budget handler/repo | TC-A18, A10, A19, M10 |
| FR4.3–4.4 update/delete budgets | US10 | API 15–16 | budget service | TC-A18, M11 |
| FR4.5 budget/actual/remaining | US9 | §2 calculations | GET /budgets computed fields | TC-A12, A20, M12 |
| FR4.6 highlight overspending | US9 | §2 `over_budget` + UI | budgets UI | TC-A20, M12 |
| FR5.1–5.3 dashboard totals | US11 | §2 dashboard | aggregation query; dashboard UI | TC-A21, A34, A12, M13 |
| FR5.4 by-category totals | US11 | §2 dashboard | GROUP BY query | TC-A21, M13 |
| FR5.5 five most recent | US11 | §2 recent-5 (D13) | LIMIT 5 query | TC-A21, M13 |
| AC1 secure passwords + token auth | US1–US2 | §5 | bcrypt + JWT cookie | TC-A01, A04, A25, A30, M03 |
| AC2 cross-user denial | all | §2 composite FKs + §5 | middleware + FKs | TC-A05, A17, A23, A33, M14 |
| AC3 frontend validation | US1,4,5,8 | §4 frontend column | React forms | TC-M01, M07, M10, M16 |
| AC4 amounts > 0 | US5, US8 | CHECK + 422 | validators | TC-A07, A19, M07 |
| AC5 valid dates | US5 | DATE + strict parse | validators | TC-A08, M16 |
| AC6 type match | US5, US7 | composite FK + 422 | tx validation | TC-A09, A16, M08 |
| AC7 expense-only budgets | US8 | composite FK (`cat_type`) + 422 | budget validation | TC-A10, A23, M10 |
| AC8 no duplicate budgets | US8 | UNIQUE + 409 | budget service | TC-A11, A23, M11 |
| AC9 live dashboard | US11 | no-cache + refetch | dashboard query + frontend | TC-A12, A21, M13 |
| AC10 calendar-month calc | US9 | half-open range on DATE | budget SQL | TC-A12, A21 |
| AC11 negative remaining + indication | US9 | remaining/flag + UI | budget response + UI | TC-A20, M12 |
| AC12 category delete guard | US4 | FK RESTRICT + 409 | error mapping | TC-A06, A23, M06 |
| AC13 decimal money | US5/9/11 | NUMERIC(12,2) + decimal lib + string wire | money plumbing | TC-A07, A19, A22, M07 |
| AC14 backend validation + HTTP errors | all API stories | §3 error policy | error middleware | TC-A29, A30, A31, A35 (+ statuses throughout) |
| AC15 notifications | all UI stories | toasts + inline errors | React notifications | TC-M02, M06, M11, M15 |
| AC16 PostgreSQL persistence | all | sole datastore | storage layer | TC-A24 |
| CON1 React functional + hooks | — | §1/§8 | frontend codebase | verified by implementation + pins in README (no automated UI tests — stated) |
| CON2 Go REST | — | §1/§8 | backend codebase | verified by implementation |
| CON3 PostgreSQL constraints | — | §2 | migrations | TC-A23, A24 |
| CON4 JWT + hashed passwords | — | §5 | auth package | TC-A04, A25, A30 |
| DEL1 GitHub repo (frontend+backend) | — | §7 layout | Phase 3 packaging | repo check at Phase 5 |
| DEL2 schema/migrations | — | §7 migrations | `/migrations` | TC-A23/A24 run on migrated DB; artifact check at Phase 5 |
| DEL3 README install/run | — | README | Phase 3/5 | TC-M17 fresh-clone |

## Coverage statement

- All 5 FRs (with sub-points), all **16 ACs**, all 4 technology constraints, and all
  3 submission deliverables map to at least one user story, design element, and test case
  (or an explicit Phase 3/5 artifact check).
- Every test ID referenced here is defined in `test_cases.md`; no dangling references.
- Honest scope notes: AC3/AC15 frontend behavior is manual-only (no frontend test framework
  proposed); CON1–2 are verified by the implementation itself and version pins, not by tests.
- Documented silences (currency, ordering direction, recent-5 scope, password policy, etc.)
  are tracked in `docs/requirements.md` §5 and decided in `docs/decisions.md` — none silently filled.
