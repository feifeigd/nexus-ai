// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func deleteKnowledgeDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteKnowledgeDocumentInput
		if err := httpx.Parse(r, &req); err != nil {
			http.Error(w, "invalid knowledge document deletion request", http.StatusBadRequest)
			return
		}

		l := logic.NewDeleteKnowledgeDocumentLogic(r.Context(), svcCtx)
		resp, err := l.DeleteKnowledgeDocument(&req)
		if err != nil {
			if errors.Is(err, data.ErrKnowledgeDocumentNotFound) {
				http.Error(w, "knowledge document not found", http.StatusNotFound)
				return
			}
			logx.WithContext(r.Context()).Errorw("knowledge document deletion failed", logx.Field("error", err))
			http.Error(w, "knowledge document deletion failed", http.StatusInternalServerError)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
