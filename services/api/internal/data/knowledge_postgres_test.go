package data

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/pgvector/pgvector-go"

	entschema "nexusai/services/api/internal/data/ent/schema"
)

func TestKnowledgeStorePostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NEXUS_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set NEXUS_TEST_POSTGRES_DSN to a disposable PostgreSQL database or schema")
	}
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:      dialect.Postgres,
		Source:      dsn,
		AutoMigrate: true,
	}})
	if err != nil {
		t.Fatalf("create postgres data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	id := fmt.Sprintf("knowledge-test-%d", time.Now().UnixNano())
	if err := client.Document.Create().
		SetID(id).
		SetTitle("pgvector 检索测试").
		SetContent("向量检索集成测试").
		Exec(ctx); err != nil {
		t.Fatalf("create postgres document: %v", err)
	}
	defer func() {
		if err := client.Document.DeleteOneID(id).Exec(ctx); err != nil {
			t.Errorf("clean up postgres document: %v", err)
		}
	}()

	for _, input := range []struct {
		suffix    string
		ordinal   int
		embedding pgvector.Vector
	}{
		{suffix: "nearest", ordinal: 0, embedding: knowledgeTestVector(1, 0)},
		{suffix: "second", ordinal: 1, embedding: knowledgeTestVector(0.9, 0.1)},
		{suffix: "farthest", ordinal: 2, embedding: knowledgeTestVector(0, 1)},
	} {
		if err := client.Chunk.Create().
			SetID(id + "-" + input.suffix).
			SetDocumentID(id).
			SetOrdinal(input.ordinal).
			SetContent(input.suffix).
			SetEmbedding(input.embedding).
			Exec(ctx); err != nil {
			t.Fatalf("create postgres chunk %s: %v", input.suffix, err)
		}
	}
	if err := client.Chunk.Create().
		SetID(id + "-without-embedding").
		SetDocumentID(id).
		SetOrdinal(3).
		SetContent("not searchable").
		Exec(ctx); err != nil {
		t.Fatalf("create postgres chunk without embedding: %v", err)
	}

	store := NewKnowledgeStore(client, dialect.Postgres)
	matches, err := store.SearchSimilar(ctx, knowledgeTestVector(1, 0), 2)
	if err != nil {
		t.Fatalf("search postgres embeddings: %v", err)
	}
	if len(matches) != 2 || matches[0].ChunkID != id+"-nearest" || matches[1].ChunkID != id+"-second" {
		t.Fatalf("unexpected cosine distance ordering: %+v", matches)
	}
	if matches[0].Distance < 0 || matches[0].Distance > matches[1].Distance {
		t.Fatalf("invalid cosine distances: %+v", matches)
	}
	if _, err := store.SearchSimilar(ctx, pgvector.NewVector([]float32{1}), 1); err == nil {
		t.Fatal("expected an embedding dimension validation error")
	}
	if _, err := store.SearchSimilar(ctx, knowledgeTestVector(1, 0), -1); err == nil {
		t.Fatal("expected a search limit validation error")
	}
}

func knowledgeTestVector(first, second float32) pgvector.Vector {
	values := make([]float32, entschema.EmbeddingDimensions)
	values[0] = first
	values[1] = second
	return pgvector.NewVector(values)
}
