package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pgvector/pgvector-go"

	entschema "nexusai/services/api/internal/data/ent/schema"
)

type KnowledgeChunkInput struct {
	Content   string
	Embedding pgvector.Vector
}

// CreateDocument stores a document and all embedded chunks atomically.
func (s *KnowledgeStore) CreateDocument(ctx context.Context, title, content string, chunks []KnowledgeChunkInput) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("knowledge store client is required")
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || utf8.RuneCountInString(title) > 200 {
		return "", fmt.Errorf("document title must contain between 1 and 200 characters")
	}
	if content == "" {
		return "", fmt.Errorf("document content is required")
	}
	if len(chunks) == 0 {
		return "", fmt.Errorf("document chunks are required")
	}
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk.Content) == "" {
			return "", fmt.Errorf("document chunk content is required")
		}
		if len(chunk.Embedding.Slice()) != entschema.EmbeddingDimensions {
			return "", fmt.Errorf("embedding must contain %d dimensions, got %d", entschema.EmbeddingDimensions, len(chunk.Embedding.Slice()))
		}
	}

	documentID, err := newKnowledgeID("document")
	if err != nil {
		return "", err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return "", fmt.Errorf("begin knowledge ingestion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.Client().Document.Create().
		SetID(documentID).
		SetTitle(title).
		SetContent(content).
		Exec(ctx); err != nil {
		return "", fmt.Errorf("insert knowledge document: %w", err)
	}
	for ordinal, chunk := range chunks {
		chunkID, err := newKnowledgeID("chunk")
		if err != nil {
			return "", err
		}
		if err := tx.Client().Chunk.Create().
			SetID(chunkID).
			SetDocumentID(documentID).
			SetOrdinal(ordinal).
			SetContent(chunk.Content).
			SetEmbedding(chunk.Embedding).
			Exec(ctx); err != nil {
			return "", fmt.Errorf("insert knowledge chunk %d: %w", ordinal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit knowledge ingestion transaction: %w", err)
	}
	return documentID, nil
}

func newKnowledgeID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate %s ID: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
