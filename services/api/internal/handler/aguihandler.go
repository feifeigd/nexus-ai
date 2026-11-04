// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"nexusai/services/api/internal/svc"
)

func agUIHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return AGUIHandler(svcCtx.Runner, svcCtx.TokenUsage)
}
