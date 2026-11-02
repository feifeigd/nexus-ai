package data

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
)

func TestKnowledgeStoreListDocuments(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_list_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	older := time.Date(2026, time.November, 1, 8, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	for _, input := range []struct {
		id        string
		title     string
		createdAt time.Time
	}{
		{id: "document-old", title: "旧运维手册", createdAt: older},
		{id: "document-new", title: "新运维手册", createdAt: newer},
	} {
		if err := client.Document.Create().
			SetID(input.id).
			SetTitle(input.title).
			SetContent("正文不会通过列表元数据返回").
			SetCreatedAt(input.createdAt).
			SetUpdatedAt(input.createdAt).
			Exec(ctx); err != nil {
			t.Fatalf("create %s: %v", input.id, err)
		}
	}

	documents, err := NewKnowledgeStore(client, dialect.SQLite).ListDocuments(ctx)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(documents) != 2 || documents[0].DocumentID != "document-new" || documents[1].DocumentID != "document-old" {
		t.Fatalf("documents are not ordered newest first: %+v", documents)
	}
	if documents[0].Title != "新运维手册" || !documents[0].CreatedAt.Equal(newer) || !documents[0].UpdatedAt.Equal(newer) {
		t.Fatalf("unexpected document metadata: %+v", documents[0])
	}
}
