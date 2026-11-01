package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Chunk stores an ordered text segment of a document. Embeddings are added by
// the retrieval increment rather than making this base model dialect-specific.
type Chunk struct {
	ent.Schema
}

func (Chunk) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("document_id").NotEmpty().Immutable(),
		field.Int("ordinal").NonNegative().Immutable(),
		field.String("content").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Chunk) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("document", Document.Type).Ref("chunks").Field("document_id").Required().Unique().Immutable(),
	}
}

func (Chunk) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("document_id", "ordinal").Unique(),
	}
}
