package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"nexusai/services/api/internal/approval"
)

type ticketListResponse struct {
	Tickets []approval.Ticket `json:"tickets"`
}

type updateTicketStatusRequest struct {
	Status approval.TicketStatus `json:"status"`
}

func ListTicketsHandler(store approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tickets, err := store.ListTickets(r.Context())
		if err != nil {
			http.Error(w, "ticket list failed", http.StatusInternalServerError)
			return
		}
		writeTicketJSON(w, http.StatusOK, ticketListResponse{Tickets: tickets})
	}
}

func UpdateTicketStatusHandler(store approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/tickets/"))
		id = strings.TrimSpace(strings.TrimSuffix(id, "/status"))
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "ticket ID is required", http.StatusBadRequest)
			return
		}
		var input updateTicketStatusRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "invalid ticket status request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			http.Error(w, "invalid ticket status request", http.StatusBadRequest)
			return
		}
		ticket, err := store.UpdateTicketStatus(r.Context(), id, input.Status)
		if err != nil {
			writeTicketError(w, err)
			return
		}
		writeTicketJSON(w, http.StatusOK, ticket)
	}
}

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
