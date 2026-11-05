// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func giteeCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 回调是 GET + query 参数（无 body），直接读 query 更可靠，避免 httpx.Parse
		// 对空 body 的处理差异。
		req := &types.GiteeCallbackInput{
			Code:  r.URL.Query().Get("code"),
			State: r.URL.Query().Get("state"),
		}
		cookie, err := r.Cookie("oauth_state")
		if err != nil || cookie.Value == "" || cookie.Value != req.State {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			return
		}
		// state 一次性使用，校验后立即失效。
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		l := logic.NewGiteeCallbackLogic(r.Context(), svcCtx)
		resp, err := l.GiteeCallback(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, resp.RedirectURL, http.StatusFound)
	}
}
