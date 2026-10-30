package api

import (
	"net/http"

	"pebt/backend/internal/httpx"
	"pebt/backend/internal/service"
)

type categoryRequest struct {
	Name service.Text `json:"name"`
	Type service.Text `json:"type"`
}

// handleCategoryList lists the user's categories with an optional ?type=
// filter (FR2.2/FR2.6).
func (s *Server) handleCategoryList(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	cats, err := s.Svc.ListCategories(r.Context(), userID, r.URL.Query().Get("type"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cats)
}

// handleCategoryCreate creates a category (FR2.1).
func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	cat, err := s.Svc.CreateCategory(r.Context(), userID, req.Name, req.Type)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, cat)
}

// handleCategoryUpdate renames / retypes a category (FR2.3).
func (s *Server) handleCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	cat, err := s.Svc.UpdateCategory(r.Context(), userID, r.PathValue("id"), req.Name, req.Type)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cat)
}

// handleCategoryDelete deletes an unused category (FR2.4).
func (s *Server) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	if err := s.Svc.DeleteCategory(r.Context(), userID, r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
