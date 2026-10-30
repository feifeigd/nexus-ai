package handler

import (
	"encoding/json"
	"net/http"

	"nexusai/services/api/internal/approval"
)

type ticketListResponse struct {
	Tickets []approval.Ticket `json:"tickets"`
}

func ListTicketsHandler(store approval.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tickets, err := store.ListTickets(r.Context())
		if err != nil {
			http.Error(w, "ticket list failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ticketListResponse{Tickets: tickets})
	}
}
