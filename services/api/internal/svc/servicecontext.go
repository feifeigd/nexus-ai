// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"context"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
)

type KnowledgeService interface {
	IngestDocument(ctx context.Context, title, content string) (string, error)
}

type KnowledgeDocumentStore interface {
	ListDocuments(ctx context.Context) ([]data.KnowledgeDocument, error)
	GetDocument(ctx context.Context, id string) (data.KnowledgeDocumentDetail, error)
	DeleteDocument(ctx context.Context, id string) error
}

type ServiceContext struct {
	Config    *config.Config
	Runner    agent.Runner
	Approvals approval.Store
	Knowledge KnowledgeService
	Documents KnowledgeDocumentStore
}

func NewServiceContext(c *config.Config, runner agent.Runner, approvals approval.Store, knowledge KnowledgeService, documents KnowledgeDocumentStore) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Runner:    runner,
		Approvals: approvals,
		Knowledge: knowledge,
		Documents: documents,
	}
}
