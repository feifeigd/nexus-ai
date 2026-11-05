package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Tenant 是租户（组织或个人空间）。users.tenant_id 指向它，同一租户下的用户共享
// 知识库、审批与工单。owner_id 是创建者/所有者用户 id。
type Tenant struct {
	ent.Schema
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name").NotEmpty().MaxLen(200),
		field.String("owner_id").Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the Tenant.
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_id").Annotations(entsql.DescColumns("created_at")),
	}
}
