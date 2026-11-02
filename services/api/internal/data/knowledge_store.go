package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data/ent"
	"nexusai/services/api/internal/data/ent/chunk"
	entdocument "nexusai/services/api/internal/data/ent/document"
	entschema "nexusai/services/api/internal/data/ent/schema"
)

const (
	DefaultKnowledgeSearchLimit = 10
	MaxKnowledgeSearchLimit     = 50
)

var (
	ErrVectorSearchUnsupported   = errors.New("vector similarity search requires PostgreSQL with pgvector")
	ErrKnowledgeDocumentNotFound = errors.New("knowledge document not found")
)

type KnowledgeMatch struct {
	ChunkID    string
	DocumentID string
	Ordinal    int
	Content    string
	Distance   float64
}

type KnowledgeDocument struct {
	DocumentID string
	Title      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type KnowledgeDocumentDetail struct {
	KnowledgeDocument
	Content string
}

type KnowledgeStore struct {
	client *ent.Client
	driver string
}

func NewKnowledgeStore(client *ent.Client, driver string) *KnowledgeStore {
	return &KnowledgeStore{client: client, driver: driver}
}

func (s *KnowledgeStore) ListDocuments(ctx context.Context) ([]KnowledgeDocument, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("knowledge store client is required")
	}
	rows, err := s.client.Document.Query().
		Order(ent.Desc(entdocument.FieldCreatedAt), ent.Asc(entdocument.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list knowledge documents: %w", err)
	}
	documents := make([]KnowledgeDocument, 0, len(rows))
	for _, row := range rows {
		documents = append(documents, KnowledgeDocument{
			DocumentID: row.ID,
			Title:      row.Title,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return documents, nil
}

func (s *KnowledgeStore) GetDocument(ctx context.Context, id string) (KnowledgeDocumentDetail, error) {
	if s == nil || s.client == nil {
		return KnowledgeDocumentDetail{}, fmt.Errorf("knowledge store client is required")
	}
	row, err := s.client.Document.Get(ctx, id)
	if ent.IsNotFound(err) {
		return KnowledgeDocumentDetail{}, ErrKnowledgeDocumentNotFound
	}
	if err != nil {
		return KnowledgeDocumentDetail{}, fmt.Errorf("get knowledge document: %w", err)
	}
	return KnowledgeDocumentDetail{
		KnowledgeDocument: KnowledgeDocument{
			DocumentID: row.ID,
			Title:      row.Title,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		},
		Content: row.Content,
	}, nil
}

// DeleteDocument removes the document and relies on the schema's foreign-key
// cascade to remove its chunks in the same transaction.
func (s *KnowledgeStore) DeleteDocument(ctx context.Context, id string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("knowledge store client is required")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin knowledge document deletion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.Client().Document.DeleteOneID(id).Exec(ctx); ent.IsNotFound(err) {
		return ErrKnowledgeDocumentNotFound
	} else if err != nil {
		return fmt.Errorf("delete knowledge document: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge document deletion transaction: %w", err)
	}
	return nil
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
