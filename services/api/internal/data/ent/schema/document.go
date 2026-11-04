package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Document stores source content before it is split into searchable chunks.
type Document struct {
	ent.Schema
}

func (Document) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("title").NotEmpty().MaxLen(200),
		field.String("content").NotEmpty(),
		field.String("tenant_id").Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Document) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("chunks", Chunk.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Document) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at", "id").Annotations(entsql.DescColumns("created_at")).StorageKey("documents_created_at_idx"),
	}
}
