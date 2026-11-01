package data

import (
	"context"
	"testing"

	"nexusai/services/api/internal/data/ent/chunk"

	"entgo.io/ent/dialect"
)

func TestKnowledgeBaseSQLiteSchema(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_base_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()

	document, err := client.Document.Create().
		SetID("document-1").
		SetTitle("网络运维手册").
		SetContent("第一章：故障排查。第二章：恢复验证。").
		Save(ctx)
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if document.CreatedAt.IsZero() || document.UpdatedAt.IsZero() {
		t.Fatalf("document timestamps were not populated: %+v", document)
	}
	for _, input := range []struct {
		id      string
		ordinal int
		content string
	}{
		{id: "chunk-2", ordinal: 1, content: "第二章：恢复验证。"},
		{id: "chunk-1", ordinal: 0, content: "第一章：故障排查。"},
	} {
		if err := client.Chunk.Create().
			SetID(input.id).
			SetDocumentID(document.ID).
			SetOrdinal(input.ordinal).
			SetContent(input.content).
			Exec(ctx); err != nil {
			t.Fatalf("create %s: %v", input.id, err)
		}
	}

	if err := client.Chunk.Create().
		SetID("chunk-duplicate").
		SetDocumentID(document.ID).
		SetOrdinal(0).
		SetContent("重复序号").
		Exec(ctx); err == nil {
		t.Fatal("expected duplicate document ordinal to violate the unique index")
	}
	chunks, err := client.Chunk.Query().
		Where(chunk.DocumentID(document.ID)).
		Order(chunk.ByOrdinal()).
		All(ctx)
	if err != nil {
		t.Fatalf("query document chunks: %v", err)
	}
	if len(chunks) != 2 || chunks[0].ID != "chunk-1" || chunks[1].ID != "chunk-2" {
		t.Fatalf("chunks are not ordered by ordinal: %+v", chunks)
	}

	if err := client.Document.DeleteOneID(document.ID).Exec(ctx); err != nil {
		t.Fatalf("delete document: %v", err)
	}
	remaining, err := client.Chunk.Query().Where(chunk.DocumentID(document.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("count remaining chunks: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleting a document must cascade to its chunks, got %d", remaining)
	}
}
