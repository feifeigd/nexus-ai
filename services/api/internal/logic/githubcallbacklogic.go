// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type GithubCallbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGithubCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GithubCallbackLogic {
	return &GithubCallbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GithubCallback 完成 GitHub OAuth 回调：构造 GitHub 客户端后走通用授权码流程
// （换 token → 拉用户 → 建号/复用 → 签 JWT），返回跳回 Web 的地址。
func (l *GithubCallbackLogic) GithubCallback(req *types.GithubCallbackInput) (*types.GithubCallbackResponse, error) {
	cfg := l.svcCtx.Config.GitHub
	oauth := &auth.GitHubOAuth{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
		TokenURL:     cfg.TokenURL,
		UserURL:      cfg.UserURL,
		EmailsURL:    cfg.EmailsURL,
	}
	redirectURL, err := oauthCallbackRedirect(l.ctx, l.svcCtx, oauth, req.Code)
	if err != nil {
		return nil, err
	}
	return &types.GithubCallbackResponse{RedirectURL: redirectURL}, nil
}
