// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type ListTenantsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTenantsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTenantsLogic {
	return &ListTenantsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListTenants 列出当前用户拥有的租户（owner 是自己）。
func (l *ListTenantsLogic) ListTenants() (*types.ListTenantsResponse, error) {
	uid := currentUserID(l.ctx)
	if uid == "" {
		return nil, errors.New("未登录")
	}
	if l.svcCtx.Tenants == nil {
		return nil, errors.New("tenant store is not configured")
	}
	tenants, err := l.svcCtx.Tenants.ListByOwner(l.ctx, uid)
	if err != nil {
		return nil, err
	}
	items := make([]types.TenantItem, 0, len(tenants))
	for _, tenant := range tenants {
		items = append(items, types.TenantItem{ID: tenant.ID, Name: tenant.Name})
	}
	return &types.ListTenantsResponse{Tenants: items}, nil
}
