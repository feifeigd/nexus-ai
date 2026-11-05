// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type GiteeLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGiteeLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GiteeLoginLogic {
	return &GiteeLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GiteeLogin 生成一次性 OAuth state 并返回码云授权跳转地址。state 由 handler 写入
// cookie 并在回调时校验，用于防 CSRF。
func (l *GiteeLoginLogic) GiteeLogin() (resp *types.GiteeLoginResponse, err error) {
	cfg := l.svcCtx.Config.Gitee
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURI == "" {
		return nil, errors.New("gitee oauth is not configured")
	}
	state, err := auth.RandomState()
	if err != nil {
		return nil, err
	}
	oauth := auth.GiteeOAuth{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
	}
	return &types.GiteeLoginResponse{RedirectURL: oauth.AuthorizeURL(state), State: state}, nil
}
