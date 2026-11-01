// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTicketStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTicketStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTicketStatusLogic {
	return &UpdateTicketStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTicketStatusLogic) UpdateTicketStatus(req *types.UpdateTicketStatusInput) (*types.Ticket, error) {
	ticket, err := l.svcCtx.Approvals.UpdateTicketStatus(l.ctx, req.ID, approval.TicketStatus(req.Status))
	if err != nil {
		return nil, err
	}
	response := ticketResponse(ticket)
	return &response, nil
}
