// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type DeleteMachineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteMachineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMachineLogic {
	return &DeleteMachineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteMachine 删除一台机器（admin 专用）。
func (l *DeleteMachineLogic) DeleteMachine(req *types.DeleteMachineInput) (*types.DeleteMachineResponse, error) {
	if l.svcCtx.Machines == nil {
		return nil, errors.New("machine store is not configured")
	}
	if err := l.svcCtx.Machines.Delete(l.ctx, req.ID); err != nil {
		if errors.Is(err, data.ErrMachineNotFound) {
			return nil, errors.New("机器不存在")
		}
		return nil, err
	}
	recordAudit(l.ctx, l.svcCtx, "delete_machine", "machine", req.ID, "")
	return &types.DeleteMachineResponse{ID: req.ID}, nil
}
