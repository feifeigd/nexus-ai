package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuditLog 是一条不可变的操作审计记录：谁（actor）在哪个租户下、对什么资源、做了什么。
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("tenant_id").Immutable(),
		field.String("actor_id").Immutable(),
		field.String("actor_username").Immutable(),
		field.String("action").Immutable(),
		field.String("resource_type").Immutable(),
		field.String("resource_id").Immutable(),
		field.String("detail").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the AuditLog.
func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "created_at").Annotations(entsql.DescColumns("created_at")),
		index.Fields("actor_id"),
	}
}
