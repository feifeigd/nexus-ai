// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type AddTenantMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddTenantMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddTenantMemberLogic {
	return &AddTenantMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AddTenantMember 由租户 owner 把一个用户（按 username）拉进组织，切换其 tenant_id。
func (l *AddTenantMemberLogic) AddTenantMember(req *types.AddTenantMemberInput) (*types.AddTenantMemberResponse, error) {
	uid := currentUserID(l.ctx)
	if uid == "" {
		return nil, errors.New("未登录")
	}
	if l.svcCtx.Tenants == nil || l.svcCtx.Users == nil {
		return nil, errors.New("tenant or user store is not configured")
	}
	tenant, err := l.svcCtx.Tenants.GetByID(l.ctx, req.ID)
	if errors.Is(err, data.ErrTenantNotFound) {
		return nil, errors.New("租户不存在")
	}
	if err != nil {
		return nil, err
	}
	role, err := l.svcCtx.Tenants.MemberRole(l.ctx, tenant.ID, uid)
	if err != nil {
		return nil, err
	}
	if role != "owner" {
		return nil, errors.New("只有租户所有者可以添加成员")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}
	member, err := l.svcCtx.Users.GetByUsername(l.ctx, username)
	if errors.Is(err, data.ErrUserNotFound) {
		return nil, errors.New("用户不存在")
	}
	if err != nil {
		return nil, err
	}
	if existing, err := l.svcCtx.Tenants.MemberRole(l.ctx, tenant.ID, member.ID); err != nil {
		return nil, err
	} else if existing != "" {
		return nil, errors.New("该用户已是成员")
	}
	if err := l.svcCtx.Tenants.AddMember(l.ctx, tenant.ID, member.ID, "member"); err != nil {
		return nil, err
	}
	recordAudit(l.ctx, l.svcCtx, "add_tenant_member", "tenant", tenant.ID, member.Username)
	return &types.AddTenantMemberResponse{UserID: member.ID, Username: member.Username}, nil
}
