package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data"
	entschema "nexusai/services/api/internal/data/ent/schema"
)

const (
	DefaultChunkSize     = 800
	DefaultChunkOverlap  = 120
	DefaultSearchLimit   = 5
	MaxSearchLimit       = 10
	MaxSearchQueryLength = 1000

	// SearchDistanceThreshold drops chunks whose cosine distance exceeds it
	// (range [0,2], 0 identical / 2 opposite), filtering low-relevance matches.
	SearchDistanceThreshold = 0.5
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([]pgvector.Vector, error)
}

type Store interface {
	CreateDocument(ctx context.Context, title, content string, chunks []data.KnowledgeChunkInput) (string, error)
	SearchSimilar(ctx context.Context, embedding pgvector.Vector, limit int) ([]data.KnowledgeMatch, error)
}

type Service struct {
	embedder     Embedder
	store        Store
	chunkSize    int
	chunkOverlap int
}

func NewService(embedder Embedder, store Store, chunkSize, chunkOverlap int) (*Service, error) {
	if embedder == nil {
		return nil, fmt.Errorf("knowledge embedder is required")
	}
	if store == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
		if chunkOverlap == 0 {
			chunkOverlap = DefaultChunkOverlap
		}
	}
	if chunkSize < 1 || chunkOverlap < 0 || chunkOverlap >= chunkSize {
		return nil, fmt.Errorf("chunk size must be positive and overlap must be smaller than chunk size")
	}
	return &Service{embedder: embedder, store: store, chunkSize: chunkSize, chunkOverlap: chunkOverlap}, nil
}

func (s *Service) IngestDocument(ctx context.Context, title, content string) (string, error) {
	chunks := splitText(content, s.chunkSize, s.chunkOverlap)
	if len(chunks) == 0 {
		return "", fmt.Errorf("document content is required")
	}
	embeddings, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		return "", fmt.Errorf("embed knowledge document: %w", err)
	}
	if len(embeddings) != len(chunks) {
		return "", fmt.Errorf("embedder returned %d vectors for %d chunks", len(embeddings), len(chunks))
	}
	inputs := make([]data.KnowledgeChunkInput, len(chunks))
	for index, chunk := range chunks {
		if len(embeddings[index].Slice()) != entschema.EmbeddingDimensions {
			return "", fmt.Errorf("embedding must contain %d dimensions, got %d", entschema.EmbeddingDimensions, len(embeddings[index].Slice()))
		}
		inputs[index] = data.KnowledgeChunkInput{Content: chunk, Embedding: embeddings[index]}
	}
	return s.store.CreateDocument(ctx, title, content, inputs)
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]data.KnowledgeMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("knowledge search query is required")
	}
	if len([]rune(query)) > MaxSearchQueryLength {
		return nil, fmt.Errorf("knowledge search query must contain at most %d characters", MaxSearchQueryLength)
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	if limit < 1 || limit > MaxSearchLimit {
		return nil, fmt.Errorf("knowledge search limit must be between 1 and %d", MaxSearchLimit)
	}
	embeddings, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed knowledge search query: %w", err)
	}
	if len(embeddings) != 1 {
		return nil, fmt.Errorf("embedder returned %d vectors for one search query", len(embeddings))
	}
	if len(embeddings[0].Slice()) != entschema.EmbeddingDimensions {
		return nil, fmt.Errorf("embedding must contain %d dimensions, got %d", entschema.EmbeddingDimensions, len(embeddings[0].Slice()))
	}
	matches, err := s.store.SearchSimilar(ctx, embeddings[0], limit)
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	filtered := make([]data.KnowledgeMatch, 0, len(matches))
	for _, match := range matches {
		if match.Distance <= SearchDistanceThreshold {
			filtered = append(filtered, match)
		}
	}
	return filtered, nil
}

func splitText(content string, size, overlap int) []string {
	runes := []rune(strings.TrimSpace(content))
	chunks := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); {
		end := min(start+size, len(runes))
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return chunks
}
