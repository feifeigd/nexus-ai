// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GithubLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGithubLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GithubLoginLogic {
	return &GithubLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GithubLogin 生成一次性 OAuth state 并返回 GitHub 授权跳转地址。state 由 handler 写入
// cookie 并在回调时校验，用于防 CSRF。
func (l *GithubLoginLogic) GithubLogin() (resp *types.GithubLoginResponse, err error) {
	cfg := l.svcCtx.Config.GitHub
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURI == "" {
		return nil, errors.New("github oauth is not configured")
	}
	state, err := auth.RandomState()
	if err != nil {
		return nil, err
	}
	oauth := auth.GitHubOAuth{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
	}
	return &types.GithubLoginResponse{RedirectURL: oauth.AuthorizeURL(state), State: state}, nil
}
