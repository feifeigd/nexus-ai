package data

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data/ent"
	"nexusai/services/api/internal/data/ent/chunk"
	entschema "nexusai/services/api/internal/data/ent/schema"
)

const (
	DefaultKnowledgeSearchLimit = 10
	MaxKnowledgeSearchLimit     = 50
)

var ErrVectorSearchUnsupported = errors.New("vector similarity search requires PostgreSQL with pgvector")

type KnowledgeMatch struct {
	ChunkID    string
	DocumentID string
	Ordinal    int
	Content    string
	Distance   float64
}

type KnowledgeStore struct {
	client *ent.Client
	driver string
}

func NewKnowledgeStore(client *ent.Client, driver string) *KnowledgeStore {
	return &KnowledgeStore{client: client, driver: driver}
}

// SearchSimilar returns chunks ordered by pgvector cosine distance. SQLite can
// persist embeddings for development tests, but deliberately has no fallback
// search implementation that would differ from production retrieval behavior.
func (s *KnowledgeStore) SearchSimilar(ctx context.Context, embedding pgvector.Vector, limit int) ([]KnowledgeMatch, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("knowledge store client is required")
	}
	if s.driver != dialect.Postgres {
		return nil, ErrVectorSearchUnsupported
	}
	if len(embedding.Slice()) != entschema.EmbeddingDimensions {
		return nil, fmt.Errorf("embedding must contain %d dimensions, got %d", entschema.EmbeddingDimensions, len(embedding.Slice()))
	}
	if limit == 0 {
		limit = DefaultKnowledgeSearchLimit
	}
	if limit < 0 || limit > MaxKnowledgeSearchLimit {
		return nil, fmt.Errorf("knowledge search limit must be between 1 and %d", MaxKnowledgeSearchLimit)
	}

	distance := entsql.P(func(b *entsql.Builder) {
		b.Ident(chunk.FieldEmbedding).WriteString(" <=> ").Arg(embedding)
	})
	var rows []struct {
		ent.Chunk
		Distance float64 `json:"distance"`
	}
	if err := s.client.Chunk.Query().
		Where(chunk.EmbeddingNotNil()).
		Limit(limit).
		Modify(func(selector *entsql.Selector) {
			selector.AppendSelectExprAs(distance, "distance")
			// Keep the vector expression as the sole ordering key so PostgreSQL
			// can satisfy the scan with the cosine HNSW index.
			selector.OrderBy("distance")
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("search knowledge chunks: %w", err)
	}

	matches := make([]KnowledgeMatch, 0, len(rows))
	for _, row := range rows {
		matches = append(matches, KnowledgeMatch{
			ChunkID:    row.ID,
			DocumentID: row.DocumentID,
			Ordinal:    row.Ordinal,
			Content:    row.Content,
			Distance:   row.Distance,
		})
	}
	return matches, nil
}
