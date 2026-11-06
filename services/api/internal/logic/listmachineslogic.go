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

type ListMachinesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMachinesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMachinesLogic {
	return &ListMachinesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListMachines 列出所有机器（平台级基础设施，staff 可读）。
func (l *ListMachinesLogic) ListMachines() (*types.ListMachinesResponse, error) {
	if l.svcCtx.Machines == nil {
		return nil, errors.New("machine store is not configured")
	}
	machines, err := l.svcCtx.Machines.List(l.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]types.Machine, 0, len(machines))
	for _, m := range machines {
		items = append(items, machineResponse(m))
	}
	return &types.ListMachinesResponse{Machines: items}, nil
}
