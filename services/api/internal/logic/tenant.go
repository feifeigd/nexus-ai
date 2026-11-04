package logic

import (
	"context"

	"nexusai/services/api/internal/auth"
)

// managementTenantID 解析审批/工单等后台管理操作的租户作用域：staff 管理全部租户
// （返回空表示不过滤），普通用户只看自己的租户。
func managementTenantID(ctx context.Context) string {
	if role, _ := ctx.Value("role").(string); role == "staff" {
		return ""
	}
	return auth.TenantIDFromContext(ctx)
}
