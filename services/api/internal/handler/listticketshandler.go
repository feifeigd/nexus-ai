// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
)

func listTicketsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListTicketsLogic(r.Context(), svcCtx)
		resp, err := l.ListTickets()
		if err != nil {
			http.Error(w, "ticket list failed", http.StatusInternalServerError)
			return
		}
		writeTicketJSON(w, http.StatusOK, resp)
	}
}
