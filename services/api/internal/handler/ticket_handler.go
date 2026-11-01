package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"nexusai/services/api/internal/approval"
)

func writeTicketError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, approval.ErrTicketNotFound):
		http.Error(w, "ticket not found", http.StatusNotFound)
	case errors.Is(err, approval.ErrInvalidTicketStatus):
		http.Error(w, "invalid ticket status", http.StatusBadRequest)
	case errors.Is(err, approval.ErrInvalidTicketTransition):
		http.Error(w, "invalid ticket status transition", http.StatusConflict)
	default:
		http.Error(w, "ticket status update failed", http.StatusInternalServerError)
	}
}

func writeTicketJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
