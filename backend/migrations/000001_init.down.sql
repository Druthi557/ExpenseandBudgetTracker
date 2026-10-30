-- 000001_init.down.sql — reverse of 000001_init.up.sql
BEGIN;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
COMMIT;
