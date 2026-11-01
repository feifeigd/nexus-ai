// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectApprovalLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRejectApprovalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectApprovalLogic {
	return &RejectApprovalLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejectApprovalLogic) RejectApproval(req *types.ApprovalDecisionInput) (*types.Approval, error) {
	result, err := l.svcCtx.Approvals.Reject(l.ctx, req.ID)
	if err != nil {
		return nil, err
	}
	response := approvalResponse(result)
	return &response, nil
}
