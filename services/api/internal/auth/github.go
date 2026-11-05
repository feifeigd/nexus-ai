package auth

// 本文件实现 GitHub OAuth 的「授权码模式（Authorization Code Flow）」客户端。
//
// 授权码模式是三方登录最常用的流程，好处是：用户的密码永远不经过我们的后端，
// 只在 GitHub 那里输入；我们拿到的是一次性授权码（code），用它换 access_token，
// 再拉取用户信息。流程：
//
//  1. 浏览器跳 GitHub 授权页（带 client_id、redirect_uri、scope、state）
//  2. 用户同意后，GitHub 带 code 跳回我们的回调地址
//  3. 后端用 code + client_id + client_secret 换 access_token
//  4. 后端用 access_token 调 GitHub API 拿用户信息（id / login / email）
//
// state 是防 CSRF 的随机串：发起授权时随机生成、回调时校验是否一致，防止攻击者
// 伪造回调把你登录成他的账号。

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubUserURL      = "https://api.github.com/user"
	githubEmailsURL    = "https://api.github.com/user/emails"
)

// GitHubOAuth 封装 GitHub OAuth 授权码流程：构造授权地址、用 code 换 access_token、
// 拉取 GitHub 用户信息。ClientSecret 只能经配置注入，不落日志。
type GitHubOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTPClient   *http.Client

	// 以下端点可覆盖（测试用），空则用 GitHub 默认地址。
	TokenURL  string
	UserURL   string
	EmailsURL string
}

// RandomState 生成一段随机的 OAuth state，用于防 CSRF。
func RandomState() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}

// AuthorizeURL 构造跳转到 GitHub 授权页的地址。
func (o *GitHubOAuth) AuthorizeURL(state string) string {
	query := url.Values{}
	query.Set("client_id", o.ClientID)
	query.Set("redirect_uri", o.RedirectURI)
	query.Set("scope", "user:email")
	query.Set("state", state)
	return githubAuthorizeURL + "?" + query.Encode()
}

// Name 返回提供方标识，用于 oauth_provider 存储与回调路由。
func (o *GitHubOAuth) Name() string { return "github" }

// ExchangeCode 用一次性授权码换 access_token。
func (o *GitHubOAuth) ExchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", o.ClientID)
	form.Set("client_secret", o.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", o.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build github token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange github code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("github token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("decode github token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("github token response missing access_token")
	}
	return token.AccessToken, nil
}

// FetchUser 用 access_token 拉取 GitHub 用户信息。
func (o *GitHubOAuth) FetchUser(ctx context.Context, accessToken string) (OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.userURL(), nil)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("build github user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := o.client().Do(req)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("fetch github user: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return OAuthUser{}, fmt.Errorf("github user endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var raw struct {
		ID    int64   `json:"id"`
		Login string  `json:"login"`
		Email *string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return OAuthUser{}, fmt.Errorf("decode github user response: %w", err)
	}
	user := OAuthUser{ID: strconv.FormatInt(raw.ID, 10), Login: raw.Login}
	if raw.Email != nil {
		user.Email = *raw.Email
	}
	if user.Email == "" {
		user.Email, err = o.fetchPrimaryEmail(ctx, accessToken)
		if err != nil {
			return OAuthUser{}, err
		}
	}
	return user, nil
}

// fetchPrimaryEmail 从 /user/emails 取主邮箱（公开邮箱可能为空，例如 GitHub 开启了隐私邮箱）。
func (o *GitHubOAuth) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.emailsURL(), nil)
	if err != nil {
		return "", fmt.Errorf("build github emails request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := o.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch github emails: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails endpoint returned %d", resp.StatusCode)
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode github emails response: %w", err)
	}
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}
	return "", nil
}

func (o *GitHubOAuth) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return http.DefaultClient
}

func (o *GitHubOAuth) tokenURL() string {
	if o.TokenURL != "" {
		return o.TokenURL
	}
	return githubTokenURL
}

func (o *GitHubOAuth) userURL() string {
	if o.UserURL != "" {
		return o.UserURL
	}
	return githubUserURL
}

func (o *GitHubOAuth) emailsURL() string {
	if o.EmailsURL != "" {
		return o.EmailsURL
	}
	return githubEmailsURL
}
