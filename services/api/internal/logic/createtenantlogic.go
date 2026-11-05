// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type CreateTenantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTenantLogic {
	return &CreateTenantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateTenant 创建一个组织租户，并把当前用户（创建者）的 tenant_id 切到该组织。
func (l *CreateTenantLogic) CreateTenant(req *types.CreateTenantInput) (*types.CreateTenantResponse, error) {
	uid := currentUserID(l.ctx)
	if uid == "" {
		return nil, errors.New("未登录")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return nil, errors.New("租户名称需在 1-200 字之间")
	}
	if l.svcCtx.Tenants == nil || l.svcCtx.Users == nil {
		return nil, errors.New("tenant or user store is not configured")
	}
	tenant, err := l.svcCtx.Tenants.Create(l.ctx, name, uid)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Tenants.AddMember(l.ctx, tenant.ID, uid, "owner"); err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.Users.UpdateTenantID(l.ctx, uid, tenant.ID); err != nil {
		return nil, err
	}
	recordAudit(l.ctx, l.svcCtx, "create_tenant", "tenant", tenant.ID, tenant.Name)
	return &types.CreateTenantResponse{TenantID: tenant.ID, Name: tenant.Name}, nil
}
