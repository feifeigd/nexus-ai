package logic

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
)

// oauthCallbackRedirect 是三方登录回调的通用流程：换 token → 拉用户 → 按
// (provider, subject) 建号/复用 → 签 JWT，返回跳回 Web 的回调地址。任何失败都通过
// redirect 的 error 参数反馈给前端（浏览器已在 OAuth 跳转流程中，不宜返回 5xx）。
func oauthCallbackRedirect(ctx context.Context, svcCtx *svc.ServiceContext, provider auth.OAuthProvider, code string) (string, error) {
	webBase := strings.TrimRight(strings.TrimSpace(svcCtx.Config.WebBaseURL), "/")
	if webBase == "" {
		webBase = "http://localhost:3000"
	}
	toWeb := func(values url.Values) string {
		return webBase + "/auth/callback?" + values.Encode()
	}
	fail := func(message string) string {
		return toWeb(url.Values{"error": {message}})
	}

	if code == "" {
		return fail("缺少授权码"), nil
	}
	accessToken, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		logx.WithContext(ctx).Errorf("exchange %s code: %v", provider.Name(), err)
		return fail("授权失败，请重试"), nil
	}
	oauthUser, err := provider.FetchUser(ctx, accessToken)
	if err != nil {
		logx.WithContext(ctx).Errorf("fetch %s user: %v", provider.Name(), err)
		return fail("获取用户信息失败，请重试"), nil
	}
	if oauthUser.ID == "" {
		return fail("用户信息不完整"), nil
	}

	user, err := svcCtx.Users.GetByOAuthSubject(ctx, provider.Name(), oauthUser.ID)
	if errors.Is(err, data.ErrUserNotFound) {
		user, err = createOAuthUser(ctx, svcCtx, provider.Name(), oauthUser.ID, oauthUser.Login)
		if err != nil {
			logx.WithContext(ctx).Errorf("create oauth user: %v", err)
			return fail("创建账号失败，请重试"), nil
		}
	} else if err != nil {
		logx.WithContext(ctx).Errorf("get user by oauth subject: %v", err)
		return fail("查询账号失败，请重试"), nil
	}

	secret := authSecret(svcCtx.Config.Auth)
	if secret == "" {
		return fail("认证密钥未配置"), nil
	}
	token, err := auth.SignToken(secret, user.ID, user.Username, user.Role, user.TenantID, time.Now(), authTTL(svcCtx.Config.Auth))
	if err != nil {
		logx.WithContext(ctx).Errorf("sign token: %v", err)
		return fail("登录失败，请重试"), nil
	}
	return toWeb(url.Values{
		"token":    {token},
		"username": {user.Username},
		"role":     {user.Role},
	}), nil
}

// createOAuthUser 用三方身份新建普通用户：用户名取登录名，冲突时追加 subject 前缀。
func createOAuthUser(ctx context.Context, svcCtx *svc.ServiceContext, provider, subject, login string) (data.UserRecord, error) {
	username := strings.TrimSpace(login)
	if username == "" {
		username = provider + "_user"
	}
	if _, err := svcCtx.Users.GetByUsername(ctx, username); err == nil {
		suffix := subject
		if len(suffix) > 8 {
			suffix = suffix[:8]
		}
		username = username + "_" + suffix
	}
	return svcCtx.Users.CreateOAuth(ctx, provider, subject, username, "user")
}
