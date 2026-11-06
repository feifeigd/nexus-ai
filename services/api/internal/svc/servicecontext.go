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
	"nexusai/services/api/internal/sshx"

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
	Audit               *data.AuditStore
	Machines            *data.MachineStore
	SSHRunner           sshx.Runner
	SSHWhitelist        *sshx.Whitelist
	StaffOnlyMiddleware rest.Middleware
	AdminOnlyMiddleware rest.Middleware
	ApproverMiddleware  rest.Middleware
}

// ServiceContextDeps 聚合 ServiceContext 的所有业务依赖。
//
// 为什么用结构体而不是一长串位置参数：位置参数构造器每新增一个依赖
// （例如机器列表的 Machines），所有按位置传参的测试都必须同步补一个 nil，
// 否则无法编译。改成结构体后，新增依赖只需在结构体里加一个字段，
// 已有测试只设置自己关心的字段、其余保持零值即可，不再受新依赖牵连。
type ServiceContextDeps struct {
	Runner     agent.Runner
	Approvals  approval.Store
	Knowledge  KnowledgeService
	Documents  KnowledgeDocumentStore
	TokenUsage *data.TokenUsageStore
	Users      *data.UserStore
	Tenants    *data.TenantStore
	Audit      *data.AuditStore
	Machines   *data.MachineStore
	SSHRunner  sshx.Runner
}

func NewServiceContext(c *config.Config, deps ServiceContextDeps) *ServiceContext {
	return &ServiceContext{
		Config:              c,
		Runner:              deps.Runner,
		Approvals:           deps.Approvals,
		Knowledge:           deps.Knowledge,
		Documents:           deps.Documents,
		TokenUsage:          deps.TokenUsage,
		Users:               deps.Users,
		Tenants:             deps.Tenants,
		Audit:               deps.Audit,
		Machines:            deps.Machines,
		SSHRunner:           deps.SSHRunner,
		SSHWhitelist:        sshx.NewWhitelist(c.SSH.AllowedCommands),
		StaffOnlyMiddleware: middleware.NewStaffOnlyMiddleware().Handle,
		AdminOnlyMiddleware: middleware.NewAdminOnlyMiddleware().Handle,
		ApproverMiddleware:  middleware.NewApproverMiddleware().Handle,
	}
}
