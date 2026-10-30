package api

import (
	"context"
	"net/http"
	"time"

	"pebt/backend/internal/auth"
	"pebt/backend/internal/httpx"
)

func httpxError(w http.ResponseWriter, status int, code, message string) {
	httpx.Error(w, status, code, message)
}

// withUserID stores the authenticated user id in the request context.
func withUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

// statusRecorder captures the response status for request logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// withLogging logs one line per request: method, path, status, duration.
// Request bodies (passwords, tokens) are never logged.
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if p := recover(); p != nil {
				s.Log.Printf("panic serving %s %s: %v", r.Method, r.URL.Path, p)
				if rec.status == http.StatusOK { // nothing written yet
					httpxError(rec, 500, "INTERNAL", "Something went wrong. Please try again.")
				}
			}
			s.Log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
		}()
		next.ServeHTTP(rec, r)
	})
}

// requireAuth authenticates the session cookie and puts the user id in the
// request context (AC1/AC2, D10).
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil || cookie.Value == "" {
			httpxError(w, 401, "TOKEN_MISSING", "Authentication required. Please log in.")
			return
		}
		userID, verr := auth.VerifyToken(cookie.Value, s.Cfg.JWTSecret)
		if verr != nil {
			if verr == auth.ErrExpired {
				httpxError(w, 401, "TOKEN_EXPIRED", "Your session has expired. Please log in again.")
			} else {
				httpxError(w, 401, "TOKEN_INVALID", "Invalid session. Please log in again.")
			}
			return
		}
		next(w, r.WithContext(withUserID(r.Context(), userID)))
	}
}

// mutatingMethods are the request methods the CSRF guard covers (D36).
var mutatingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// requireCSRF rejects mutating requests without the X-Requested-With header
// (D36; SameSite=Lax is the first layer, this header the second).
func (s *Server) requireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mutatingMethods[r.Method] && r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			httpxError(w, 403, "CSRF_CHECK_FAILED", "Missing X-Requested-With header")
			return
		}
		next(w, r)
	}
}
