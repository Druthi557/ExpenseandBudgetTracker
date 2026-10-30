package api

import (
	"net/http"

	"pebt/backend/internal/httpx"
	"pebt/backend/internal/service"
)

// handleTransactionList lists transactions newest-first with optional
// ?type=, ?category_id=, ?from=, ?to= filters (FR3.4/FR3.5).
func (s *Server) handleTransactionList(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	q := r.URL.Query()
	txs, err := s.Svc.ListTransactions(r.Context(), userID,
		q.Get("type"), q.Get("category_id"), q.Get("from"), q.Get("to"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, txs)
}

// handleTransactionCreate adds a transaction (FR3.1).
func (s *Server) handleTransactionCreate(w http.ResponseWriter, r *http.Request) {
	var req service.TxInput
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	tx, err := s.Svc.CreateTransaction(r.Context(), userID, req)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, tx)
}

// handleTransactionGet fetches one transaction (FR3.3).
func (s *Server) handleTransactionGet(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	tx, err := s.Svc.GetTransaction(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tx)
}

// handleTransactionUpdate fully replaces a transaction (FR3.3).
func (s *Server) handleTransactionUpdate(w http.ResponseWriter, r *http.Request) {
	var req service.TxInput
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	userID := userIDFrom(r.Context())
	tx, err := s.Svc.UpdateTransaction(r.Context(), userID, r.PathValue("id"), req)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tx)
}

// handleTransactionDelete removes a transaction (FR3.3).
func (s *Server) handleTransactionDelete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r.Context())
	if err := s.Svc.DeleteTransaction(r.Context(), userID, r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
