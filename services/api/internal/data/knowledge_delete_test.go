package data

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/data/ent/chunk"
)

func TestKnowledgeStoreDeleteDocument(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_delete_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()

	if err := client.Document.Create().
		SetID("document-delete").
		SetTitle("待删除运维手册").
		SetContent("第一章：故障排查。第二章：恢复验证。").
		SetTenantID("tenant-a").
		Exec(ctx); err != nil {
		t.Fatalf("create document: %v", err)
	}
	for _, input := range []struct {
		id      string
		ordinal int
		content string
	}{
		{id: "chunk-delete-1", ordinal: 0, content: "第一章：故障排查。"},
		{id: "chunk-delete-2", ordinal: 1, content: "第二章：恢复验证。"},
	} {
		if err := client.Chunk.Create().
			SetID(input.id).
			SetDocumentID("document-delete").
			SetTenantID("tenant-a").
			SetOrdinal(input.ordinal).
			SetContent(input.content).
			Exec(ctx); err != nil {
			t.Fatalf("create %s: %v", input.id, err)
		}
	}

	store := NewKnowledgeStore(client, dialect.SQLite)
	if err := store.DeleteDocument(ctx, "document-delete", "tenant-a"); err != nil {
		t.Fatalf("delete document: %v", err)
	}
	if _, err := store.GetDocument(ctx, "document-delete", "tenant-a"); !errors.Is(err, ErrKnowledgeDocumentNotFound) {
		t.Fatalf("expected deleted document to be absent, got %v", err)
	}
	remaining, err := client.Chunk.Query().Where(chunk.DocumentID("document-delete")).Count(ctx)
	if err != nil {
		t.Fatalf("count remaining chunks: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected document chunks to be cascade deleted, got %d", remaining)
	}
	if err := store.DeleteDocument(ctx, "document-delete", "tenant-a"); !errors.Is(err, ErrKnowledgeDocumentNotFound) {
		t.Fatalf("expected repeated deletion to return not found, got %v", err)
	}
}
