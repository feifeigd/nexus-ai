package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Approval holds the schema definition for the Approval entity.
type Approval struct {
	ent.Schema
}

// Fields of the Approval.
func (Approval) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("thread_id").Immutable(),
		field.String("run_id").Immutable(),
		field.String("tool_call_id").Unique().Immutable(),
		field.String("tool_name").Immutable(),
		field.JSON("arguments", map[string]any{}).Immutable(),
		field.String("status"),
		field.String("ticket_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("decided_at").Optional().Nillable(),
	}
}

// Edges of the Approval.
func (Approval) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ticket", Ticket.Type).Unique(), // 一个 Approval 可以拥有零个或一个 Ticket
	}
}

// 索引
func (Approval) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at", "id").Annotations(entsql.DescColumns("created_at")).StorageKey("approvals_created_at_idx"),
	}
}

func (Approval) Annotations() []schema.Annotation {
	return []schema.Annotation{
		&entsql.Annotation{
			Checks: map[string]string{
				"approvals_status_check": "status IN ('pending', 'approved', 'rejected')",
			},
		},
	}
}
