package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
	return nil
}
