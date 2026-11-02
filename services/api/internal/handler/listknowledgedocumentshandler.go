// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
)

func listKnowledgeDocumentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListKnowledgeDocumentsLogic(r.Context(), svcCtx)
		resp, err := l.ListKnowledgeDocuments()
		if err != nil {
			logx.WithContext(r.Context()).Errorw("knowledge document list failed", logx.Field("error", err))
			http.Error(w, "knowledge document list failed", http.StatusInternalServerError)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
