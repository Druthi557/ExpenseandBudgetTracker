package api

import (
	"net/http"
	"time"

	"pebt/backend/internal/auth"
	"pebt/backend/internal/httpx"
	"pebt/backend/internal/service"
)

type credentialsRequest struct {
	Email    service.Text `json:"email"`
	Password service.Text `json:"password"`
}

type sessionResponse struct {
	ExpiresAt time.Time `json:"expires_at"` // token never in a body (D30)
}

// setSessionCookie delivers the JWT in an HttpOnly, SameSite=Lax cookie
// (D10); Secure follows configuration (true in production).
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.TokenTTL.Seconds()),
		Expires:  exp,
		HttpOnly: true,
		Secure:   s.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie removes the cookie server-side (logout, FR1.3).
func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // renders as Max-Age=0: delete now
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   s.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// handleRegister creates an account and logs the user in (FR1.1).
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID, err := s.Svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	token, exp, tokErr := auth.IssueToken(userID, s.Cfg.JWTSecret)
	if tokErr != nil {
		s.writeServiceError(w, tokErr)
		return
	}
	s.setSessionCookie(w, token, exp)
	httpx.WriteJSON(w, http.StatusCreated, sessionResponse{ExpiresAt: exp})
}

// handleLogin verifies credentials and sets the session cookie (FR1.2).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID, err := s.Svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	token, exp, tokErr := auth.IssueToken(userID, s.Cfg.JWTSecret)
	if tokErr != nil {
		s.writeServiceError(w, tokErr)
		return
	}
	s.setSessionCookie(w, token, exp)
	httpx.WriteJSON(w, http.StatusOK, sessionResponse{ExpiresAt: exp})
}

// handleLogout clears the session cookie (FR1.3). The auth middleware has
// already rejected requests without a valid session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
