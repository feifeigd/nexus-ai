// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"nexusai/services/api/internal/logic"
	"nexusai/services/api/internal/svc"
)

func giteeLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGiteeLoginLogic(r.Context(), svcCtx)
		resp, err := l.GiteeLogin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    resp.State,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   600,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, resp.RedirectURL, http.StatusFound)
	}
}
