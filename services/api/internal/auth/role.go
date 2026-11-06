package auth

// 用户角色（细粒度授权）。
const (
	RoleAdmin    = "admin"    // 管理员：读写 + 审批全部
	RoleAuditor  = "auditor"  // 审核员：读 + 审批，不能改工单/文档
	RoleReadonly = "readonly" // 只读：只能查看
	RoleUser     = "user"     // 普通用户：登录 Web，使用聊天与租户
)

// staffRoles 是内部工作人员角色（可登录 Admin）。
var staffRoles = map[string]bool{
	RoleAdmin:    true,
	RoleAuditor:  true,
	RoleReadonly: true,
}

// IsStaffRole 判断 role 是否属于内部工作人员（admin/auditor/readonly）。
func IsStaffRole(role string) bool {
	return staffRoles[role]
}

// CanApprove 判断 role 是否可以审批（admin/auditor）。
func CanApprove(role string) bool {
	return role == RoleAdmin || role == RoleAuditor
}

// IsAdmin 判断 role 是否为管理员。
func IsAdmin(role string) bool {
	return role == RoleAdmin
}
