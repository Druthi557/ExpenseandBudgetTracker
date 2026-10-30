// Package backend embeds the SQL migration files so the migrate command and
// the test suite can apply them without external tooling (decision D20 — the
// files themselves stay golang-migrate-compatible on disk).
package backend

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS

// MigrationsDir is the sub-filesystem root containing the .up.sql/.down.sql files.
func MigrationsDir() (embed.FS, string) {
	return MigrationsFS, "migrations"
}
