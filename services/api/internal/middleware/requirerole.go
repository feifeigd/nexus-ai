package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

// RequireRole 返回一个校验当前用户 role 是否满足 check 的中间件。
func RequireRole(check func(string) bool) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(string)
			if !check(role) {
				logx.WithContext(r.Context()).Errorw(
					"role access denied",
					logx.Field("role", role),
					logx.Field("path", r.URL.Path),
				)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}
