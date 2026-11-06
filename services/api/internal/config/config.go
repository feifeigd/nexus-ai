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
	Gitee  GiteeOAuthConfig
	SSH    SSHConfig

	// WebBaseURL 是 Web 用户端地址（如 http://localhost:3000），三方登录成功后跳回
	// Web 用它拼接回调地址。所有提供方共享。
	WebBaseURL string `json:",optional"`
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
	TokenURL     string `json:",optional"`
	UserURL      string `json:",optional"`
	EmailsURL    string `json:",optional"`
}

// GiteeOAuthConfig 是码云（Gitee）OAuth 配置，字段语义同 GitHubOAuthConfig。
type GiteeOAuthConfig struct {
	ClientID     string `json:",optional"`
	ClientSecret string `json:",optional"`
	RedirectURI  string `json:",optional"`
	TokenURL     string `json:",optional"`
	UserURL      string `json:",optional"`
	EmailsURL    string `json:",optional"`
}

// SSHConfig 是远程命令执行的配置。User/Port/Timeout 未配置时使用默认值（root/22/30s）。
// AllowedCommands 是命令白名单前缀，可用环境变量 NEXUS_SSH_ALLOWED_COMMANDS（逗号分隔）覆盖。
type SSHConfig struct {
	User            string   `json:",optional"`
	Port            int      `json:",optional"`
	Timeout         int      `json:",optional"` // 命令超时（秒），默认 30
	AllowedCommands []string `json:",optional"`
}
