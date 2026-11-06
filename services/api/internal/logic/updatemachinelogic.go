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

type UpdateMachineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateMachineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMachineLogic {
	return &UpdateMachineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateMachine 全量更新一台机器（admin 专用）。
func (l *UpdateMachineLogic) UpdateMachine(req *types.UpdateMachineInput) (*types.Machine, error) {
	if l.svcCtx.Machines == nil {
		return nil, errors.New("machine store is not configured")
	}
	privateAddress := strings.TrimSpace(req.PrivateAddress)
	if privateAddress == "" {
		return nil, errors.New("私有地址不能为空")
	}
	m, err := l.svcCtx.Machines.Update(l.ctx, data.MachineRecord{
		ID:             req.ID,
		Name:           strings.TrimSpace(req.Name),
		PrivateAddress: privateAddress,
		PublicAddress:  strings.TrimSpace(req.PublicAddress),
		Region:         strings.TrimSpace(req.Region),
		RootPublicKey:  strings.TrimSpace(req.RootPublicKey),
		JoinedK8s:      req.JoinedK8s,
	})
	if err != nil {
		return nil, err
	}
	recordAudit(l.ctx, l.svcCtx, "update_machine", "machine", m.ID, m.PrivateAddress)
	resp := machineResponse(m)
	return &resp, nil
}
