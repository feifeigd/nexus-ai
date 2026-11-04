package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Ticket holds the schema definition for the Ticket entity.
type Ticket struct {
	ent.Schema
}

// Fields of the Ticket.
func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(), // 覆盖 Ent 默认的自增整数 ID，成为字符串主键。
		field.String("approval_id").Unique().Immutable(),
		field.String("title"),
		field.String("description"),
		field.String("status"),
		field.String("tenant_id").Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Ticket.
func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("approval", Approval.Type).Ref("ticket_relation").Immutable().
			Field("approval_id").Required().Unique(),
	}
}

// Annotations of the Ticket.
func (Ticket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		&entsql.Annotation{
			Checks: map[string]string{
				"tickets_status_check": "status IN ('open', 'in_progress', 'resolved')",
			},
		},
	}
}
