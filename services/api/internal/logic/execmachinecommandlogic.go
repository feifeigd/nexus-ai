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

type ExecMachineCommandLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExecMachineCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExecMachineCommandLogic {
	return &ExecMachineCommandLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ExecMachineCommand 在机器上执行一条白名单内的 SSH 命令（admin 专用）。
// 安全顺序：取机器 → 校验白名单 → 检查私钥 → 执行 → 审计。私钥只在此处内部读取，
// 绝不返回给调用方。
func (l *ExecMachineCommandLogic) ExecMachineCommand(req *types.ExecMachineCommandInput) (*types.ExecMachineCommandResponse, error) {
	if l.svcCtx.Machines == nil {
		return nil, errors.New("machine store is not configured")
	}
	if l.svcCtx.SSHRunner == nil {
		return nil, errors.New("ssh runner is not configured")
	}

	machine, err := l.svcCtx.Machines.GetByID(l.ctx, req.ID)
	if err != nil {
		if errors.Is(err, data.ErrMachineNotFound) {
			return nil, errors.New("机器不存在")
		}
		return nil, err
	}

	command := strings.TrimSpace(req.Command)
	if l.svcCtx.SSHWhitelist == nil || !l.svcCtx.SSHWhitelist.Allowed(command) {
		return nil, errors.New("命令不在白名单内")
	}
	if machine.SSHPrivateKey == "" {
		return nil, errors.New("该机器未配置 SSH 私钥")
	}

	result, err := l.svcCtx.SSHRunner.Exec(l.ctx, machine.PrivateAddress, []byte(machine.SSHPrivateKey), command)
	if err != nil {
		// 连接/解析失败：不回传内部错误细节。
		return nil, errors.New("SSH 执行失败")
	}

	recordAudit(l.ctx, l.svcCtx, "run_ssh_command", "machine", machine.ID, command)
	return &types.ExecMachineCommandResponse{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}, nil
}
