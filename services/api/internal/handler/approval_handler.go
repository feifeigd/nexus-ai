package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"nexusai/services/api/internal/approval"
)

func writeApprovalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, approval.ErrNotFound):
		http.Error(w, "approval not found", http.StatusNotFound)
	case errors.Is(err, approval.ErrAlreadyDecided):
		http.Error(w, "approval has already been decided", http.StatusConflict)
	default:
		http.Error(w, "approval decision failed", http.StatusInternalServerError)
	}
}

func writeApprovalJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
