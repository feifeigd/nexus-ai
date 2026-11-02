package knowledge

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/data/ent/chunk"
	entschema "nexusai/services/api/internal/data/ent/schema"
)

type embedderFunc func(ctx context.Context, texts []string) ([]pgvector.Vector, error)

func (f embedderFunc) Embed(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	return f(ctx, texts)
}

func TestServiceIngestsChunkedDocument(t *testing.T) {
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_service_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	store := data.NewKnowledgeStore(client, dialect.SQLite)
	var embeddedTexts []string
	embedder := embedderFunc(func(_ context.Context, texts []string) ([]pgvector.Vector, error) {
		embeddedTexts = append([]string(nil), texts...)
		vectors := make([]pgvector.Vector, len(texts))
		for index := range texts {
			vectors[index] = knowledgeServiceTestVector(float32(index + 1))
		}
		return vectors, nil
	})
	service, err := NewService(embedder, store, 5, 1)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}
	documentID, err := service.IngestDocument(context.Background(), " 运维手册 ", "abcdefghij")
	if err != nil {
		t.Fatalf("ingest document: %v", err)
	}
	if !reflect.DeepEqual(embeddedTexts, []string{"abcde", "efghi", "ij"}) {
		t.Fatalf("unexpected embedded chunks: %v", embeddedTexts)
	}
	document, err := client.Document.Get(context.Background(), documentID)
	if err != nil {
		t.Fatalf("load ingested document: %v", err)
	}
	if document.Title != "运维手册" || document.Content != "abcdefghij" {
		t.Fatalf("unexpected document: %+v", document)
	}
	chunks, err := client.Chunk.Query().Where(chunk.DocumentID(documentID)).Order(chunk.ByOrdinal()).All(context.Background())
	if err != nil {
		t.Fatalf("load ingested chunks: %v", err)
	}
	if len(chunks) != 3 || chunks[0].Content != "abcde" || chunks[1].Content != "efghi" || chunks[2].Content != "ij" {
		t.Fatalf("unexpected persisted chunks: %+v", chunks)
	}
	for index, chunk := range chunks {
		if chunk.Embedding == nil || chunk.Embedding.Slice()[0] != float32(index+1) {
			t.Fatalf("chunk %d has the wrong embedding", index)
		}
	}

	failingService, err := NewService(embedderFunc(func(context.Context, []string) ([]pgvector.Vector, error) {
		return nil, errors.New("provider unavailable")
	}), store, 5, 1)
	if err != nil {
		t.Fatalf("create failing knowledge service: %v", err)
	}
	if _, err := failingService.IngestDocument(context.Background(), "失败文档", "abcdefghij"); err == nil {
		t.Fatal("expected embedding failure")
	}
	count, err := client.Document.Query().Count(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("failed ingestion must not create a document, count=%d err=%v", count, err)
	}
}

func TestSplitTextUsesRuneBoundaries(t *testing.T) {
	got := splitText("一二三四五六", 3, 1)
	want := []string{"一二三", "三四五", "五六"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected rune-aware chunks: got %v want %v", got, want)
	}
}

func knowledgeServiceTestVector(value float32) pgvector.Vector {
	values := make([]float32, entschema.EmbeddingDimensions)
	values[0] = value
	return pgvector.NewVector(values)
}
