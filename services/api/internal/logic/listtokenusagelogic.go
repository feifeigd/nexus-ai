// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"time"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTokenUsageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTokenUsageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTokenUsageLogic {
	return &ListTokenUsageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTokenUsageLogic) ListTokenUsage(scope string) (resp *types.ListTokenUsageResponse, err error) {
	if l.svcCtx.TokenUsage == nil {
		return nil, errors.New("token usage store is not configured")
	}
	role, _ := l.ctx.Value("role").(string)
	tenantID := auth.TenantIDFromContext(l.ctx)
	// 内部工作人员（admin/auditor/readonly）且 scope=all 时看全部租户；其余只看自己租户。
	if auth.IsStaffRole(role) && scope == "all" {
		tenantID = ""
	} else if tenantID == "" {
		return nil, errors.New("未登录或缺少租户信息")
	}
	records, err := l.svcCtx.TokenUsage.List(l.ctx, tenantID, 50)
	if err != nil {
		return nil, err
	}
	items := make([]types.TokenUsageItem, 0, len(records))
	for _, r := range records {
		items = append(items, types.TokenUsageItem{
			ThreadID:         r.ThreadID,
			RunID:            r.RunID,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			TotalTokens:      r.TotalTokens,
			JudgeTokens:      r.JudgeTokens,
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &types.ListTokenUsageResponse{Items: items}, nil
}
