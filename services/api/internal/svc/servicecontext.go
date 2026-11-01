// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"context"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
)

type KnowledgeService interface {
	IngestDocument(ctx context.Context, title, content string) (string, error)
}

type ServiceContext struct {
	Config    config.Config
	Runner    agent.Runner
	Approvals approval.Store
	Knowledge KnowledgeService
}

func NewServiceContext(c config.Config, runner agent.Runner, approvals approval.Store, knowledge KnowledgeService) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Runner:    runner,
		Approvals: approvals,
		Knowledge: knowledge,
	}
}
