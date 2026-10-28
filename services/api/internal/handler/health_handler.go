package handler

import (
	"net/http"

	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, types.HealthResponse{
			Status:  "ok",
			Service: "nexus-api",
		})
	}
}
