package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/pgvector/pgvector-go"
)

const EmbeddingDimensions = 1536

// Chunk stores an ordered text segment and its optional embedding. PostgreSQL
// uses pgvector; SQLite persists the value only for development and tests.
type Chunk struct {
	ent.Schema
}

func (Chunk) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("document_id").NotEmpty().Immutable(),
		field.Int("ordinal").NonNegative().Immutable(),
		field.String("content").NotEmpty(),
		field.Other("embedding", pgvector.Vector{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "vector(1536)",
				dialect.SQLite:   "BLOB",
			}),
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
