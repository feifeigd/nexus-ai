package config

import (
	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Agent  agent.Config
	Data   data.Data
	Auth   AuthConfig
	GitHub GitHubOAuthConfig
}

// AuthConfig 是认证配置，字段对齐 go-zero 的 JWT 中间件命名。AccessSecret 是 JWT
// 签名密钥，AccessExpire 是 token 有效期（秒）。可用环境变量 NEXUS_AUTH_SECRET 覆盖密钥。
type AuthConfig struct {
	AccessSecret string `json:",optional"`
	AccessExpire int64  `json:",optional"`
}

// GitHubOAuthConfig 是 GitHub OAuth（三方登录）配置。ClientSecret 只能经环境变量注入，
// 不写进 yaml 或提交 Git。RedirectURI 必须与 GitHub OAuth App 注册的回调地址一致。
// TokenURL/UserURL/EmailsURL 留空使用 GitHub 默认地址，主要用于测试与 GitHub Enterprise。
type GitHubOAuthConfig struct {
	ClientID     string `json:",optional"`
	ClientSecret string `json:",optional"`
	RedirectURI  string `json:",optional"`
	WebBaseURL   string `json:",optional"`
	TokenURL     string `json:",optional"`
	UserURL      string `json:",optional"`
	EmailsURL    string `json:",optional"`
}
