// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func githubCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GithubCallbackInput
		if err := httpx.Parse(r, &req); err != nil {
			http.Error(w, "invalid github callback", http.StatusBadRequest)
			return
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

		l := logic.NewGithubCallbackLogic(r.Context(), svcCtx)
		resp, err := l.GithubCallback(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, resp.RedirectURL, http.StatusFound)
	}
}
