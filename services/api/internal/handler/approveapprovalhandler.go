// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"
	"strings"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func approveApprovalHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := types.ApprovalDecisionInput{ID: strings.TrimSpace(pathvar.Vars(r)["id"])}
		if req.ID == "" {
			http.Error(w, "approval ID is required", http.StatusBadRequest)
			return
		}

		l := logic.NewApproveApprovalLogic(r.Context(), svcCtx)
		resp, err := l.ApproveApproval(&req)
		if err != nil {
			writeApprovalError(w, err)
			return
		}
		writeApprovalJSON(w, http.StatusOK, resp)
	}
}
