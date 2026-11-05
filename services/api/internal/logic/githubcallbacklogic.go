// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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

// GithubCallback 完成 GitHub 授权码换 token、拉取用户、按 github_id 建立/复用本地账号、
// 签发 JWT，并返回跳回 Web 的回调地址。任何失败都通过 redirect 的 error 参数反馈给前端，
// 而不是返回 5xx，因为浏览器已在 OAuth 跳转流程中。
func (l *GithubCallbackLogic) GithubCallback(req *types.GithubCallbackInput) (resp *types.GithubCallbackResponse, err error) {
	webBase := strings.TrimRight(strings.TrimSpace(l.svcCtx.Config.GitHub.WebBaseURL), "/")
	if webBase == "" {
		webBase = "http://localhost:3000"
	}
	toWeb := func(values url.Values) *types.GithubCallbackResponse {
		return &types.GithubCallbackResponse{RedirectURL: webBase + "/auth/callback?" + values.Encode()}
	}
	fail := func(message string) *types.GithubCallbackResponse {
		return toWeb(url.Values{"error": {message}})
	}

	if req.Code == "" {
		return fail("缺少授权码"), nil
	}
	cfg := l.svcCtx.Config.GitHub
	oauth := auth.GitHubOAuth{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
		TokenURL:     cfg.TokenURL,
		UserURL:      cfg.UserURL,
		EmailsURL:    cfg.EmailsURL,
	}
	accessToken, err := oauth.ExchangeCode(l.ctx, req.Code)
	if err != nil {
		l.Errorf("exchange github code: %v", err)
		return fail("GitHub 授权失败，请重试"), nil
	}
	ghUser, err := oauth.FetchUser(l.ctx, accessToken)
	if err != nil {
		l.Errorf("fetch github user: %v", err)
		return fail("获取 GitHub 用户信息失败，请重试"), nil
	}
	if ghUser.ID == "" {
		return fail("GitHub 用户信息不完整"), nil
	}

	user, err := l.svcCtx.Users.GetByGithubID(l.ctx, ghUser.ID)
	if errors.Is(err, data.ErrUserNotFound) {
		user, err = l.createOAuthUser(ghUser.ID, ghUser.Login)
		if err != nil {
			l.Errorf("create oauth user: %v", err)
			return fail("创建账号失败，请重试"), nil
		}
	} else if err != nil {
		l.Errorf("get user by github id: %v", err)
		return fail("查询账号失败，请重试"), nil
	}

	secret := authSecret(l.svcCtx.Config.Auth)
	if secret == "" {
		return fail("认证密钥未配置"), nil
	}
	token, err := auth.SignToken(secret, user.ID, user.Username, user.Role, user.TenantID, time.Now(), authTTL(l.svcCtx.Config.Auth))
	if err != nil {
		l.Errorf("sign token: %v", err)
		return fail("登录失败，请重试"), nil
	}
	return toWeb(url.Values{
		"token":    {token},
		"username": {user.Username},
		"role":     {user.Role},
	}), nil
}

// createOAuthUser 用 GitHub 身份新建普通用户，用户名取登录名，冲突时追加 id 前缀。
func (l *GithubCallbackLogic) createOAuthUser(githubID, login string) (data.UserRecord, error) {
	username := strings.TrimSpace(login)
	if username == "" {
		username = "github_user"
	}
	if _, err := l.svcCtx.Users.GetByUsername(l.ctx, username); err == nil {
		suffix := githubID
		if len(suffix) > 8 {
			suffix = suffix[:8]
		}
		username = username + "_" + suffix
	}
	return l.svcCtx.Users.CreateOAuth(l.ctx, username, "user", githubID)
}
