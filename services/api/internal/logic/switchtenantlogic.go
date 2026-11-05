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

type SwitchTenantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSwitchTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SwitchTenantLogic {
	return &SwitchTenantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SwitchTenant 把当前用户的「激活租户」切到指定组织。只能切到自己所属的租户；
// 切换后需重新登录（或刷新 token）才会按新租户过滤数据。
func (l *SwitchTenantLogic) SwitchTenant(req *types.SwitchTenantInput) (*types.SwitchTenantResponse, error) {
	uid := currentUserID(l.ctx)
	if uid == "" {
		return nil, errors.New("未登录")
	}
	if l.svcCtx.Tenants == nil || l.svcCtx.Users == nil {
		return nil, errors.New("tenant or user store is not configured")
	}
	role, err := l.svcCtx.Tenants.MemberRole(l.ctx, req.ID, uid)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, errors.New("你不是该组织成员")
	}
	if _, err := l.svcCtx.Users.UpdateTenantID(l.ctx, uid, req.ID); err != nil {
		return nil, err
	}
	return &types.SwitchTenantResponse{TenantID: req.ID}, nil
}
