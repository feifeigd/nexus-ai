// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"context"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/middleware"

	"github.com/zeromicro/go-zero/rest"
)

type KnowledgeService interface {
	IngestDocument(ctx context.Context, title, content, tenantID string) (string, error)
}

type KnowledgeDocumentStore interface {
	ListDocuments(ctx context.Context, tenantID string) ([]data.KnowledgeDocument, error)
	GetDocument(ctx context.Context, id, tenantID string) (data.KnowledgeDocumentDetail, error)
	DeleteDocument(ctx context.Context, id, tenantID string) error
}

type ServiceContext struct {
	Config              *config.Config
	Runner              agent.Runner
	Approvals           approval.Store
	Knowledge           KnowledgeService
	Documents           KnowledgeDocumentStore
	TokenUsage          *data.TokenUsageStore
	Users               *data.UserStore
	Tenants             *data.TenantStore
	StaffOnlyMiddleware rest.Middleware
}

func NewServiceContext(c *config.Config, runner agent.Runner, approvals approval.Store, knowledge KnowledgeService, documents KnowledgeDocumentStore, tokenUsage *data.TokenUsageStore, users *data.UserStore, tenants *data.TenantStore) *ServiceContext {
	return &ServiceContext{
		Config:              c,
		Runner:              runner,
		Approvals:           approvals,
		Knowledge:           knowledge,
		Documents:           documents,
		TokenUsage:          tokenUsage,
		Users:               users,
		Tenants:             tenants,
		StaffOnlyMiddleware: middleware.NewStaffOnlyMiddleware().Handle,
	}
}
