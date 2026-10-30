package api

import (
	"net/http"

	"pebt/backend/internal/httpx"
)

// handleDashboard serves the FR5 aggregates for ?year=&month=.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	q := r.URL.Query()
	view, err := s.Svc.Dashboard(r.Context(), userID, q.Get("year"), q.Get("month"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}
