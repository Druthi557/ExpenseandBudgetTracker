# Decision Register, Risks & Open Questions

**Status legend:**
- **Mandatory (doc)** — required by `druthi 1.docx`; not a choice.
- **Ruled** — explicitly decided by the user.
- **Default (proposed)** — recommended default standing unless the user objects; Phase 2
  approval ratifies the table. **Nothing below is silently treated as approved.**

## 1. Decision register

| ID | Decision | Position | Status |
|---|---|---|---|
| D1 | Frontend framework | React.js, functional components + hooks | Mandatory (doc CON1) |
| D2 | Backend | Go REST APIs | Mandatory (doc CON2) |
| D3 | Database | PostgreSQL, relationships + constraints | Mandatory (doc CON3) |
| D4 | Auth mechanism | JWT + hashed passwords | Mandatory (doc CON4) |
| D5 | Monetary arithmetic | Exact decimal, never floating-point | Mandatory (doc AC13) |
| D6 | Hash algorithm | bcrypt, cost 12 (argon2id alternative rejected for simplicity) | Default |
| D7 | Login identity | Email (+password); no display name | Default |
| D8 | Password policy | ≥8 chars, ≥1 letter + 1 digit, ≤72 bytes | Default |
| D9 | JWT profile & logout | HS256, 60-min expiry, claims sub/iat/exp; logout clears the cookie server-side; revocation limitation documented | **Ruled** (2026-10-29) |
| D10 | Token transport | HttpOnly + Secure + SameSite=Lax cookie `pebt_token`; no token in response bodies | **Ruled** (2026-10-29) |
| D11 | Display currency | INR (₹), display-only; storage currency-agnostic | Default (Phase 1 Q2) |
| D12 | Transaction ordering | Newest-first, `tx_date DESC, id DESC` tie-break | Default (Phase 1 Q4) |
| D13 | Dashboard recent-5 scope | Overall (any month); by-category totals = selected month (genuine doc ambiguity resolved as proposed) | Default (Phase 1 Q3) |
| D14 | Money wire format | JSON strings, exactly 2 dp, via explicit `StringFixed(2)` DTOs; >2 dp → 422 | **Ruled** (2026-10-29) |
| D15 | Go money type | shopspring/decimal (int64-paise alternative rejected) | Default |
| D16 | Foreign resources | 404, identical to not-found (existence masking) | Default |
| D17 | Error envelope | `{"error":{"code","message"}}` | Default |
| D18 | Category names | Unique per user, case-insensitive: `UNIQUE (user_id, lower(name))` — optional vs doc | Default |
| D19 | Expense-only budgets | Carrier column + CHECK + composite FK (trigger alternative rejected) | Default |
| D20 | Schema delivery | Versioned SQL migrations, golang-migrate (doc permits schema.sql too) | Default |
| D21 | Go router | stdlib `net/http` (Go 1.22+); chi optional | Default |
| D22 | Automated tests | In scope: httptest + dedicated test DB (Phase 1 Q8) | Default |
| D23 | Pagination | None in v1 | Default |
| D24 | Date storage | `tx_date DATE` (no time/time-of-day) | Default |
| D25 | Dashboard visuals | Table only, no chart | Default |
| D26 | Money column | NUMERIC(12,2) | **Ruled** (2026-10-29, with D14) |
| D27 | Category type changes | Allowed while unused; DB composite FK auto-blocks retyping referenced categories (→409) | Default |
| D28 | HTTP status policy | 400 malformed JSON · 401 auth · 403 CSRF · 404 missing/foreign · 409 conflicts · 422 validation (fields & query params) · 500 unexpected | Default (test A27/A28 aligned to it) |
| D29 | API base path | `/api/v1` | Default |
| D30 | Auth response bodies | `{expires_at}` only; token never in a body (follows from D10) | Default (revised with D10) |
| D31 | Column naming | `tx_date` | Default |
| D32 | `/auth/me` | Omit; login state probed via any authenticated call | Default |
| D33 | AC16 verification | Restart test TC-A24 (not just design inspection) | Default |
| D34 | Test ID scheme | Unified TC-A01–A35 + TC-M01–M17; superseded v2/6-part IDs retired | Default |
| D35 | Registration fields | Email + password only | Default |
| D36 | CSRF strategy | SameSite=Lax + required `X-Requested-With` header on mutations (double-submit token = option B) | Default (chosen under D10 ruling) |
| D37 | Cookie & deployment config | Cookie attrs per design §5; same-origin deployment (Vite proxy in dev, Go serves SPA in prod); split-host = documented option B | Default (chosen under D10 ruling) |
| D38 | Budget edit behavior | `PUT /budgets/{id}` edits **amount only**; changing category/month/year = delete + recreate (US10) | Default |

> **Status update 2026-10-29 — Phase 2 APPROVED.** Per the user's explicit delegation ("You have
> my authorization to make the genuine approval decision based on the evidence"), the final Phase 2
> review of the saved documents against `druthi 1.docx` found no genuine blockers, and Phase 2 was
> **approved** (rationale and evidence: CLAUDE.md §11 Approval Log). Effect: every row marked
> **Default** above is thereby **ratified as approved**; **Ruled** rows reflect the user's explicit
> rulings; **Mandatory (doc)** rows remain assignment requirements. This approval covers design
> only — **all 52 tests remain Not Run**, and **Phase 3 stays BLOCKED** until the user separately
> authorizes implementation.

## 2. Risk register

| Risk | Impact | Likelihood | Mitigation | Status |
|---|---|---|---|---|
| Float leakage into the frontend breaking AC13 | High | Low | String DTOs (D14) + TC-A07/A19/A22 | Mitigated by design |
| Month-boundary aggregation bug (AC10) | Medium | Medium | `DATE` column, half-open range, TC-A12/A21 | Mitigated by design |
| JWT-in-cookie misconfiguration ( Secure/SameSite wrong in an environment) | Medium | Low | D37 config table + cookie-attribute assertions in TC-A01/A04 + README security notes | Open |
| Split-host deployment tempting (breaks SameSite=Lax cookies) | Medium | Low | Same-origin recommended; option B documented with requirements | Open |
| Stateless logout token window (≤60 min) | Low | Accepted | Documented limitation; short expiry; HttpOnly | Accepted |
| Local Go/PostgreSQL/Node setup friction on Windows | Medium | Medium | README + fresh-clone run TC-M17 | Open |
| prompt.log capture reliability (no automatic hook installed) | Low | Certain (manual) | Manual/assisted appending; honest UNRECOVERABLE marks; no automatic capture claimed | Open |
| Scope creep | Medium | Medium | Phase approval gates (CLAUDE.md §9) | Active |

## 3. Assumptions

1. **Same-origin topology** (D37): dev proxy + Go-serving-SPA. If split hosting is later chosen,
   only cookie/CORS *configuration* changes (option B) — not the design.
2. **Localhost is a secure context** in current browsers, so the Secure cookie works in local dev;
   `COOKIE_SECURE=false` exists as a local-only fallback.
3. **CSRF header check is sufficient** for this assignment's scope; double-submit token remains
   option B.
4. **Register auto-logs-in** (sets the session cookie) — a small UX choice made in design.
5. Exact patch versions are pinned at Phase 3 start and recorded in README (D20/D21 etc.).

## 4. Open questions

- None blocking Phase 2 approval. The Default rows above are ratifiable as a block
  ("approve all recommendations") or row-by-row.
- Post-approval open item: scheduling/effort for Phase 3 (no deadline set, Phase 1 Q1).

## 5. Milestone status

| Milestone | Status |
|---|---|
| M0 Phase 1 plan approved | **Done** (2026-10-27) |
| M1 Phase 2 design | Drafted (v2.1) — **awaiting user approval** |
| M2–M7 (auth → CRUD → budgets/dashboard → UI → verification → packaging) | Planned, Phase 3+ — blocked |
