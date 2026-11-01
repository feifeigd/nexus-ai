package handler

import (
	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/svc"
)

func newTestServiceContext(store approval.Store, runner agent.Runner) *svc.ServiceContext {
	if runner == nil {
		runner = agent.MockRunner{}
	}
	return svc.NewServiceContext(config.Config{}, runner, store)
}
