// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
)

type ServiceContext struct {
	Config    config.Config
	Runner    agent.Runner
	Approvals approval.Store
}

func NewServiceContext(c config.Config, runner agent.Runner, approvals approval.Store) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Runner:    runner,
		Approvals: approvals,
	}
}
