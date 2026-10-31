package schema

import "entgo.io/ent"

// Approval holds the schema definition for the Approval entity.
type Approval struct {
	ent.Schema
}

// Fields of the Approval.
func (Approval) Fields() []ent.Field {
	return nil
}

// Edges of the Approval.
func (Approval) Edges() []ent.Edge {
	return nil
}
