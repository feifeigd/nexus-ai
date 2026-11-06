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

type CreateMachineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMachineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMachineLogic {
	return &CreateMachineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateMachine 创建一台机器（admin 专用）。私有地址必填。
func (l *CreateMachineLogic) CreateMachine(req *types.CreateMachineInput) (*types.Machine, error) {
	if l.svcCtx.Machines == nil {
		return nil, errors.New("machine store is not configured")
	}
	privateAddress := strings.TrimSpace(req.PrivateAddress)
	if privateAddress == "" {
		return nil, errors.New("私有地址不能为空")
	}
	m, err := l.svcCtx.Machines.Create(l.ctx, data.MachineRecord{
		Name:           strings.TrimSpace(req.Name),
		PrivateAddress: privateAddress,
		PublicAddress:  strings.TrimSpace(req.PublicAddress),
		Region:         strings.TrimSpace(req.Region),
		RootPublicKey:  strings.TrimSpace(req.RootPublicKey),
		SSHPrivateKey:  strings.TrimSpace(req.SSHPrivateKey),
		JoinedK8s:      req.JoinedK8s,
	})
	if err != nil {
		return nil, err
	}
	recordAudit(l.ctx, l.svcCtx, "create_machine", "machine", m.ID, m.PrivateAddress)
	resp := machineResponse(m)
	return &resp, nil
}
