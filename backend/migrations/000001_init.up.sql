-- 000001_init.up.sql — Personal Expense and Budget Tracker
-- Phase 3 M2 foundation schema — implements docs/design.md §2 exactly (approved design).
-- Naming is golang-migrate-compatible (NNNNNN_name.up.sql / .down.sql) per decision D20.
-- Applies cleanly to an empty database. Status: WRITTEN, NOT YET APPLIED (needs DB credentials).

BEGIN;

-- ============ users ============
CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Case-insensitive unique email (D7; plain expression index, no CITEXT — U3)
CREATE UNIQUE INDEX users_email_lower_unique_idx ON users (lower(email));

-- ============ categories ============
CREATE TABLE categories (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name    VARCHAR(100) NOT NULL,
    type    VARCHAR(10)  NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),

    -- FK target that makes ownership+type composite keys expressible (design §2, C1/C2)
    CONSTRAINT categories_id_user_type_unique UNIQUE (id, user_id, type)
);

-- Case-insensitive per-user duplicate-name guard (D18; queries must use identical expression)
CREATE UNIQUE INDEX categories_user_lower_name_unique_idx ON categories (user_id, lower(name));
CREATE INDEX categories_user_id_idx ON categories (user_id);

-- ============ transactions ============
CREATE TABLE transactions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type        VARCHAR(10)   NOT NULL CHECK (type IN ('INCOME', 'EXPENSE')),
    category_id BIGINT        NOT NULL,
    amount      NUMERIC(12,2) NOT NULL CHECK (amount > 0),   -- AC4
    tx_date     DATE          NOT NULL,                      -- AC5 (real calendar date, no TZ)
    description VARCHAR(500),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),        -- app-maintained (U5)

    -- Ownership + type integrity at the DB level (AC2/AC6) — cross-user or wrong-type
    -- references are impossible even if an API check were missed.
    -- ON DELETE RESTRICT implements AC12.
    CONSTRAINT transactions_category_fk FOREIGN KEY (category_id, user_id, type)
        REFERENCES categories (id, user_id, type) ON DELETE RESTRICT
);

CREATE INDEX transactions_user_date_idx     ON transactions (user_id, tx_date DESC);
CREATE INDEX transactions_user_type_idx     ON transactions (user_id, type);
CREATE INDEX transactions_user_cat_date_idx ON transactions (user_id, category_id, tx_date);

-- ============ budgets ============
CREATE TABLE budgets (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id BIGINT        NOT NULL,
    cat_type    VARCHAR(10)   NOT NULL DEFAULT 'EXPENSE' CHECK (cat_type = 'EXPENSE'),  -- AC7 carrier column
    year        SMALLINT      NOT NULL CHECK (year BETWEEN 2000 AND 2100),
    month       SMALLINT      NOT NULL CHECK (month BETWEEN 1 AND 12),
    amount      NUMERIC(12,2) NOT NULL CHECK (amount > 0),   -- AC4
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),

    -- Expense-only + same-user at the DB level (AC2/AC7): the CHECK pins cat_type to
    -- 'EXPENSE', so this FK can only ever match expense categories of the same user.
    -- ON DELETE RESTRICT implements AC12.
    CONSTRAINT budgets_category_fk FOREIGN KEY (category_id, user_id, cat_type)
        REFERENCES categories (id, user_id, type) ON DELETE RESTRICT,

    -- One budget per user/category/month/year (AC8)
    CONSTRAINT budgets_user_cat_year_month_unique UNIQUE (user_id, category_id, year, month)
);

CREATE INDEX budgets_user_year_month_idx ON budgets (user_id, year, month);

COMMIT;
