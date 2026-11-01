// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func updateTicketStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := types.UpdateTicketStatusInput{ID: strings.TrimSpace(pathvar.Vars(r)["id"])}
		if req.ID == "" {
			http.Error(w, "ticket ID is required", http.StatusBadRequest)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid ticket status request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			http.Error(w, "invalid ticket status request", http.StatusBadRequest)
			return
		}

		l := logic.NewUpdateTicketStatusLogic(r.Context(), svcCtx)
		resp, err := l.UpdateTicketStatus(&req)
		if err != nil {
			writeTicketError(w, err)
			return
		}
		writeTicketJSON(w, http.StatusOK, resp)
	}
}
