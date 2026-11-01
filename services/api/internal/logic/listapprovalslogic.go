// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListApprovalsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListApprovalsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListApprovalsLogic {
	return &ListApprovalsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListApprovalsLogic) ListApprovals() (*types.ApprovalListResponse, error) {
	approvals, err := l.svcCtx.Approvals.List(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.ApprovalListResponse{Approvals: approvalResponses(approvals)}, nil
}
