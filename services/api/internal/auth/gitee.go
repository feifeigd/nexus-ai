package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	giteeAuthorizeURL = "https://gitee.com/oauth/authorize"
	giteeTokenURL     = "https://gitee.com/oauth/token"
	giteeUserURL      = "https://gitee.com/api/v5/user"
	giteeEmailsURL    = "https://gitee.com/api/v5/emails"
)

// GiteeOAuth 封装码云（Gitee）OAuth 授权码流程。与 GitHub 的差异：
//   - 换 token 时必须带 User-Agent，否则 Gitee 会返回 403；
//   - 用户接口用 access_token 查询参数鉴权（而非 Bearer 头）；
//   - /user 的 email 可能是字面量「未公开邮箱」，需回落到 /emails 取主邮箱。
type GiteeOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTPClient   *http.Client

	// 以下端点可覆盖（测试用），空则用 Gitee 默认地址。
	TokenURL  string
	UserURL   string
	EmailsURL string
}

// Name 返回提供方标识，用于 oauth_provider 存储与回调路由。
func (o *GiteeOAuth) Name() string { return "gitee" }

// AuthorizeURL 构造跳转到 Gitee 授权页的地址。
func (o *GiteeOAuth) AuthorizeURL(state string) string {
	query := url.Values{}
	query.Set("client_id", o.ClientID)
	query.Set("redirect_uri", o.RedirectURI)
	query.Set("response_type", "code")
	query.Set("scope", "user_info emails")
	query.Set("state", state)
	return giteeAuthorizeURL + "?" + query.Encode()
}

// ExchangeCode 用一次性授权码换 access_token。
func (o *GiteeOAuth) ExchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", o.ClientID)
	form.Set("client_secret", o.ClientSecret)
	form.Set("redirect_uri", o.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build gitee token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "nexus-ai") // Gitee 缺 User-Agent 会 403

	resp, err := o.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange gitee code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("gitee token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("decode gitee token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("gitee token response missing access_token")
	}
	return token.AccessToken, nil
}

// FetchUser 用 access_token 拉取 Gitee 用户信息。
func (o *GiteeOAuth) FetchUser(ctx context.Context, accessToken string) (OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.userURLWithToken(accessToken), nil)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("build gitee user request: %w", err)
	}
	req.Header.Set("User-Agent", "nexus-ai")

	resp, err := o.client().Do(req)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("fetch gitee user: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return OAuthUser{}, fmt.Errorf("gitee user endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var raw struct {
		ID    json.RawMessage `json:"id"` // 数字或字符串，按字符串兼容
		Login string          `json:"login"`
		Email *string         `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return OAuthUser{}, fmt.Errorf("decode gitee user response: %w", err)
	}
	// id 可能是数字也可能是字符串：json.RawMessage 里数字无引号、字符串带引号，
	// 去掉首尾引号后两种都得到纯数字串。
	id := strings.Trim(string(raw.ID), `"`)
	if id == "" || id == "0" {
		return OAuthUser{}, fmt.Errorf("gitee user response missing id")
	}
	user := OAuthUser{ID: id, Login: raw.Login}
	if raw.Email != nil {
		user.Email = *raw.Email
	}
	// Gitee 用户隐藏邮箱时返回字面量「未公开邮箱」，视为空并回落到 /emails。
	if user.Email == "" || user.Email == "未公开邮箱" {
		user.Email, err = o.fetchPrimaryEmail(ctx, accessToken)
		if err != nil {
			return OAuthUser{}, err
		}
	}
	return user, nil
}

// fetchPrimaryEmail 从 /emails 取主邮箱（state=confirmed 且 scope 含 primary）。
func (o *GiteeOAuth) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.emailsURLWithToken(accessToken), nil)
	if err != nil {
		return "", fmt.Errorf("build gitee emails request: %w", err)
	}
	req.Header.Set("User-Agent", "nexus-ai")

	resp, err := o.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch gitee emails: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitee emails endpoint returned %d", resp.StatusCode)
	}
	var emails []struct {
		Email string          `json:"email"`
		State string          `json:"state"`
		Scope json.RawMessage `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode gitee emails response: %w", err)
	}
	for _, email := range emails {
		if email.State == "confirmed" && strings.Contains(string(email.Scope), "primary") {
			return email.Email, nil
		}
	}
	for _, email := range emails {
		if email.State == "confirmed" {
			return email.Email, nil
		}
	}
	return "", nil
}

func (o *GiteeOAuth) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return http.DefaultClient
}

func (o *GiteeOAuth) tokenURL() string {
	if o.TokenURL != "" {
		return o.TokenURL
	}
	return giteeTokenURL
}

func (o *GiteeOAuth) userURLWithToken(accessToken string) string {
	base := giteeUserURL
	if o.UserURL != "" {
		base = o.UserURL
	}
	return base + "?access_token=" + url.QueryEscape(accessToken)
}

func (o *GiteeOAuth) emailsURLWithToken(accessToken string) string {
	base := giteeEmailsURL
	if o.EmailsURL != "" {
		base = o.EmailsURL
	}
	return base + "?access_token=" + url.QueryEscape(accessToken)
}
