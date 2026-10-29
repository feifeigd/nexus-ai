package config

import (
	"nexusai/services/api/internal/agent"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Agent agent.Config
}
