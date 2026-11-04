// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveApprovalLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApproveApprovalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveApprovalLogic {
	return &ApproveApprovalLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApproveApprovalLogic) ApproveApproval(req *types.ApprovalDecisionInput) (*types.Approval, error) {
	result, err := l.svcCtx.Approvals.Approve(l.ctx, req.ID, managementTenantID(l.ctx))
	if err != nil {
		return nil, err
	}
	response := approvalResponse(result)
	return &response, nil
}
