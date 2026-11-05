package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TenantMember 记录用户与租户的成员关系。一个用户可属于多个租户；role 区分 owner
// （可管理成员）与 member。users.tenant_id 仍是「当前激活租户」，与本表无关。
type TenantMember struct {
	ent.Schema
}

// Fields of the TenantMember.
func (TenantMember) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("tenant_id").Immutable(),
		field.String("user_id").Immutable(),
		field.String("role").Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the TenantMember.
func (TenantMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}
