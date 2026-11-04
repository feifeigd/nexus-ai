package data

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
)

func TestKnowledgeStoreTenantIsolation(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_tenant_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()

	for _, input := range []struct {
		id       string
		tenantID string
		title    string
	}{
		{id: "doc-tenant-a", tenantID: "tenant-a", title: "A 租户文档"},
		{id: "doc-tenant-b", tenantID: "tenant-b", title: "B 租户文档"},
	} {
		if err := client.Document.Create().
			SetID(input.id).
			SetTitle(input.title).
			SetContent("正文").
			SetTenantID(input.tenantID).
			Exec(ctx); err != nil {
			t.Fatalf("create %s: %v", input.id, err)
		}
	}

	store := NewKnowledgeStore(client, dialect.SQLite)

	documents, err := store.ListDocuments(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list tenant-a documents: %v", err)
	}
	if len(documents) != 1 || documents[0].DocumentID != "doc-tenant-a" {
		t.Fatalf("expected only tenant-a documents, got %+v", documents)
	}

	if _, err := store.GetDocument(ctx, "doc-tenant-b", "tenant-a"); !errors.Is(err, ErrKnowledgeDocumentNotFound) {
		t.Fatalf("expected cross-tenant read to return not found, got %v", err)
	}

	if err := store.DeleteDocument(ctx, "doc-tenant-b", "tenant-a"); !errors.Is(err, ErrKnowledgeDocumentNotFound) {
		t.Fatalf("expected cross-tenant delete to return not found, got %v", err)
	}
	if _, err := client.Document.Get(ctx, "doc-tenant-b"); err != nil {
		t.Fatalf("cross-tenant delete must not remove the other tenant's document: %v", err)
	}
}
