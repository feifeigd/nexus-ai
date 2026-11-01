// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
)

func listApprovalsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListApprovalsLogic(r.Context(), svcCtx)
		resp, err := l.ListApprovals()
		if err != nil {
			writeApprovalError(w, err)
			return
		}
		writeApprovalJSON(w, http.StatusOK, resp)
	}
}
