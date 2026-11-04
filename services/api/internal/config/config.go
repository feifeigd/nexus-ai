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

// AuthConfig 是认证配置。Secret 是 JWT 签名密钥，可用环境变量 NEXUS_AUTH_SECRET 覆盖。
type AuthConfig struct {
	Secret string `json:",optional"`
}
