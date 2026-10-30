// Package api is the HTTP layer: routing, middleware (session, CSRF, logging,
// panic recovery), request/response DTOs and cookie handling. It owns the JWT
// secret and never logs passwords or tokens (CLAUDE.md §5).
package api

import (
	"context"
	"log"
	"net/http"

	"pebt/backend/internal/config"
	"pebt/backend/internal/service"
)

// Server carries the shared dependencies for all handlers. The router field
// is set by NewRouter and used by withLogging.
type Server struct {
	Svc    *service.Service
	Cfg    *config.Config
	Log    *log.Logger
	router http.Handler
}

// New builds a Server.
func New(svc *service.Service, cfg *config.Config, lg *log.Logger) *Server {
	if lg == nil {
		lg = log.Default()
	}
	return &Server{Svc: svc, Cfg: cfg, Log: lg}
}

type userIDKey struct{}

// userIDFrom returns the authenticated user id placed in the context by the
// auth middleware. The user id never comes from a request body or URL.
func userIDFrom(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey{}).(int64)
	return id
}

// writeServiceError renders a domain error with its pinned status/code, or a
// generic 500 if an untyped error ever escapes the service layer.
func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	if e, ok := err.(*service.Error); ok {
		httpxError(w, e.Status, e.Code, e.Message)
		return
	}
	s.Log.Printf("unhandled service error: %v", err)
	httpxError(w, 500, "INTERNAL", "Something went wrong. Please try again.")
}
