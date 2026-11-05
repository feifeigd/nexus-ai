package auth

import "context"

// OAuthUser 是三方登录拉取到的、用于建立本地账号的最小用户信息。ID 是平台上稳定
// 的用户 id（subject），Login 是用户名，Email 是主邮箱（可能为空）。
type OAuthUser struct {
	ID    string
	Login string
	Email string
}

// OAuthProvider 抽象一个三方登录提供方（如 github、gitee）的授权码流程。每个提供方
// 的端点、鉴权方式、用户/邮箱解析略有差异，但对回调逻辑暴露统一的四步：
// 构造授权地址 → 用 code 换 token → 拉取用户。Name 用于身份存储（oauth_provider）。
type OAuthProvider interface {
	Name() string
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (string, error)
	FetchUser(ctx context.Context, accessToken string) (OAuthUser, error)
}
