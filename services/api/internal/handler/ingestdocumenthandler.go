// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func ingestDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IngestDocumentInput
		if err := httpx.Parse(r, &req); err != nil {
			http.Error(w, "invalid knowledge document request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" || utf8.RuneCountInString(strings.TrimSpace(req.Title)) > 200 || strings.TrimSpace(req.Content) == "" {
			http.Error(w, "title must contain 1-200 characters and content is required", http.StatusBadRequest)
			return
		}

		l := logic.NewIngestDocumentLogic(r.Context(), svcCtx)
		resp, err := l.IngestDocument(&req)
		if err != nil {
			if errors.Is(err, logic.ErrKnowledgeNotConfigured) {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			logx.WithContext(r.Context()).Errorw("knowledge ingestion failed", logx.Field("error", err))
			http.Error(w, "knowledge ingestion failed", http.StatusInternalServerError)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
