// Package dbmigrate applies the versioned SQL migrations (D20) using
// golang-migrate against the embedded migration files. The migration scripts
// contain multiple statements, so the underlying connection runs in PostgreSQL
// simple protocol mode (extended protocol forbids multi-statement scripts).
package dbmigrate

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func open(databaseURL string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return sql.OpenDB(stdlib.GetConnector(*cfg)), nil
}

// newMigrate wires golang-migrate: embedded file source + postgres database
// driver over a pgx-backed database/sql handle.
func newMigrate(databaseURL string, fsys fs.FS) (*migrate.Migrate, *sql.DB, error) {
	db, err := open(databaseURL)
	if err != nil {
		return nil, nil, err
	}
	src, err := iofs.New(fsys, ".")
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	m, err := migrate.NewWithInstance("iofs", src, "", driver)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	return m, db, nil
}

// Up applies all pending migrations from fsys (a filesystem rooted at the
// migrations directory). Applying an already up-to-date database is a no-op.
func Up(databaseURL string, fsys fs.FS) error {
	m, db, err := newMigrate(databaseURL, fsys)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}

// Down rolls back every applied migration.
func Down(databaseURL string, fsys fs.FS) error {
	m, db, err := newMigrate(databaseURL, fsys)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("rolling back migrations: %w", err)
	}
	return nil
}

// DropSchema drops and recreates the public schema. Used only by the test
// suite to start every run from a clean database.
func DropSchema(databaseURL string) error {
	db, err := open(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	for _, stmt := range []string{
		`DROP SCHEMA IF EXISTS public CASCADE`,
		`CREATE SCHEMA public`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("dropping schema: %w", err)
		}
	}
	return nil
}
