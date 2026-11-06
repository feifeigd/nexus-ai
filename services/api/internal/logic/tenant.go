package logic

import (
	"context"

	"nexusai/services/api/internal/auth"
)

// managementTenantID 解析审批/工单等后台管理操作的租户作用域：内部工作人员（admin/
// auditor/readonly）管理全部租户（返回空表示不过滤），普通用户只看自己的租户。
func managementTenantID(ctx context.Context) string {
	if role, _ := ctx.Value("role").(string); auth.IsStaffRole(role) {
		return ""
	}
	return auth.TenantIDFromContext(ctx)
}

// currentUserID 从请求 context 取 JWT 注入的当前用户 id（uid）。
func currentUserID(ctx context.Context) string {
	userID, _ := ctx.Value("uid").(string)
	return userID
}
