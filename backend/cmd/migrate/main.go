// Command migrate applies or rolls back the embedded SQL migrations against
// the database given via DATABASE_URL (or -url). Usage:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
//	go run ./cmd/migrate reset   (drop schema, then apply all)
package main

import (
	"fmt"
	"io/fs"
	"os"

	"pebt/backend"
	"pebt/backend/internal/config"
	"pebt/backend/internal/dbmigrate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate up|down|reset")
		os.Exit(2)
	}
	config.LoadDotEnv(".env") // same .env fallbacks as the server
	config.LoadDotEnv("../.env")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is not set (copy .env.example to .env and fill it in)")
		os.Exit(2)
	}
	fsys, err := fs.Sub(backend.MigrationsFS, "migrations")
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "up":
		err = dbmigrate.Up(url, fsys)
		if err == nil {
			fmt.Println("migrations applied")
		}
	case "down":
		err = dbmigrate.Down(url, fsys)
		if err == nil {
			fmt.Println("migrations rolled back")
		}
	case "reset":
		err = dbmigrate.DropSchema(url)
		if err == nil {
			err = dbmigrate.Up(url, fsys)
			if err == nil {
				fmt.Println("schema reset and migrations applied")
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (want up|down|reset)\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
	os.Exit(1)
}
