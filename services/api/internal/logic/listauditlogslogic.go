// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type ListAuditLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAuditLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAuditLogsLogic {
	return &ListAuditLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListAuditLogs 返回最近的操作审计记录。staff（本路由仅 staff 可访问）看全部租户，
// 按时间倒序、最多 100 条。
func (l *ListAuditLogsLogic) ListAuditLogs() (*types.ListAuditLogsResponse, error) {
	if l.svcCtx.Audit == nil {
		return nil, errors.New("audit store is not configured")
	}
	entries, err := l.svcCtx.Audit.List(l.ctx, "", 100)
	if err != nil {
		return nil, err
	}
	items := make([]types.AuditLogItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, types.AuditLogItem{
			ID:            entry.ID,
			ActorUsername: entry.ActorUsername,
			Action:        entry.Action,
			ResourceType:  entry.ResourceType,
			ResourceID:    entry.ResourceID,
			Detail:        entry.Detail,
			CreatedAt:     entry.CreatedAt.Format(time.RFC3339),
		})
	}
	return &types.ListAuditLogsResponse{Items: items}, nil
}
