package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
)

// recordAudit 追加一条操作审计记录。失败只记日志、不影响主操作——审计是旁路，
// 不能因为审计失败而让业务操作失败。
func recordAudit(ctx context.Context, svcCtx *svc.ServiceContext, action, resourceType, resourceID, detail string) {
	if svcCtx == nil || svcCtx.Audit == nil {
		return
	}
	actorUsername, _ := ctx.Value("username").(string)
	if err := svcCtx.Audit.Create(ctx, data.AuditRecord{
		TenantID:      auth.TenantIDFromContext(ctx),
		ActorID:       currentUserID(ctx),
		ActorUsername: actorUsername,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Detail:        detail,
	}); err != nil {
		logx.WithContext(ctx).Errorw("record audit", logx.Field("action", action), logx.Field("error", err))
	}
}
