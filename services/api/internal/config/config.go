package config

import (
	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Agent agent.Config
	Data  data.Data
	Auth  AuthConfig
}

// AuthConfig 是认证配置，字段对齐 go-zero 的 JWT 中间件命名。AccessSecret 是 JWT
// 签名密钥，AccessExpire 是 token 有效期（秒）。可用环境变量 NEXUS_AUTH_SECRET 覆盖密钥。
type AuthConfig struct {
	AccessSecret string `json:",optional"`
	AccessExpire int64  `json:",optional"`
}
