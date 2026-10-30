package api

import (
	"net/http"

	"pebt/backend/internal/httpx"
	"pebt/backend/internal/service"
)

// handleBudgetList lists budgets for ?year=&month= with computed actual,
// remaining and over-budget figures (FR4.5/FR4.6).
func (s *Server) handleBudgetList(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	q := r.URL.Query()
	budgets, err := s.Svc.ListBudgets(r.Context(), userID, q.Get("year"), q.Get("month"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, budgets)
}

// handleBudgetCreate creates a monthly budget (FR4.1).
func (s *Server) handleBudgetCreate(w http.ResponseWriter, r *http.Request) {
	var req service.BudgetInput
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	budget, err := s.Svc.CreateBudget(r.Context(), userID, req)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, budget)
}

// handleBudgetUpdate edits the budget amount only (D38).
func (s *Server) handleBudgetUpdate(w http.ResponseWriter, r *http.Request) {
	var req service.BudgetAmountInput
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	budget, err := s.Svc.UpdateBudgetAmount(r.Context(), userID, r.PathValue("id"), req)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, budget)
}

// handleBudgetDelete removes a budget (FR4.4).
func (s *Server) handleBudgetDelete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	if err := s.Svc.DeleteBudget(r.Context(), userID, r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
