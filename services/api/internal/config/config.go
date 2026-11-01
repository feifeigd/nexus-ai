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
}
