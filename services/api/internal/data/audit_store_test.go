package data

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
)

func TestAuditStoreCreateAndList(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:audit_store_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewAuditStore(client)

	if err := store.Create(ctx, AuditRecord{
		TenantID:      "tenant-1",
		ActorID:       "user-1",
		ActorUsername: "alice",
		Action:        "approve_approval",
		ResourceType:  "approval",
		ResourceID:    "approval-1",
		Detail:        "open",
	}); err != nil {
		t.Fatalf("create audit log: %v", err)
	}

	entries, err := store.List(ctx, "tenant-1", 10)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(entries) != 1 || entries[0].ActorUsername != "alice" || entries[0].Action != "approve_approval" || entries[0].ResourceID != "approval-1" {
		t.Fatalf("unexpected audit entries: %+v", entries)
	}

	// 其他租户不可见。
	others, err := store.List(ctx, "tenant-2", 10)
	if err != nil {
		t.Fatalf("list other tenant logs: %v", err)
	}
	if len(others) != 0 {
		t.Fatalf("expected no cross-tenant audit entries, got %+v", others)
	}
}
