// Command server runs the Expense & Budget Tracker REST API. Configuration
// comes from the environment (or a local .env file); secrets are never
// hardcoded (CLAUDE.md §5).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pebt/backend/internal/api"
	"pebt/backend/internal/config"
	"pebt/backend/internal/service"
	"pebt/backend/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	if !cfg.CookieSecure {
		log.Print("COOKIE_SECURE=false — non-Secure cookies, local development only")
	}
	if cfg.StaticDir == "" {
		log.Print("no built frontend found (frontend/dist) — API only; run the Vite dev server")
	} else {
		log.Printf("serving SPA from %s", cfg.StaticDir)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database unreachable (did you run the migrations?): %v", err)
	}

	srv := api.New(service.New(store.New(pool), log.Default()), cfg, log.Default())
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.NewRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("shutting down…")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
	}
}
