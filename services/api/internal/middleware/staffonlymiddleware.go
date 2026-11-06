// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/auth"
)

type StaffOnlyMiddleware struct {
}

func NewStaffOnlyMiddleware() *StaffOnlyMiddleware {
	return &StaffOnlyMiddleware{}
}

// Handle 校验当前登录用户的 role 是否属于内部工作人员（admin/auditor/readonly）。
// 依赖 JWT 中间件已把 token 的 claims（含 role）注入 request context。
func (m *StaffOnlyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		if !auth.IsStaffRole(role) {
			logx.WithContext(r.Context()).Errorw(
				"staff-only access denied",
				logx.Field("role", role),
				logx.Field("path", r.URL.Path),
			)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
