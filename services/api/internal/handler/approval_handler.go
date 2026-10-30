package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"nexusai/services/api/internal/approval"
)

type approvalListResponse struct {
	Approvals []approval.Approval `json:"approvals"`
}

func ListApprovalsHandler(store *approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeApprovalJSON(w, http.StatusOK, approvalListResponse{Approvals: store.List()})
	}
}

func ApproveApprovalHandler(store *approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := approvalID(r)
		if id == "" {
			http.Error(w, "approval ID is required", http.StatusBadRequest)
			return
		}
		result, err := store.Approve(r.Context(), id)
		if err != nil {
			writeApprovalError(w, err)
			return
		}
		writeApprovalJSON(w, http.StatusOK, result)
	}
}

func RejectApprovalHandler(store *approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := approvalID(r)
		if id == "" {
			http.Error(w, "approval ID is required", http.StatusBadRequest)
			return
		}
		result, err := store.Reject(r.Context(), id)
		if err != nil {
			writeApprovalError(w, err)
			return
		}
		writeApprovalJSON(w, http.StatusOK, result)
	}
}

func approvalID(r *http.Request) string {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/approvals/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

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
