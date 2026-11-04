package data

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
)

func TestTokenUsageStoreCreateAndList(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:token_usage_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewTokenUsageStore(client)

	older := time.Date(2026, time.November, 1, 8, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	if err := store.Create(ctx, TokenUsageRecord{
		ThreadID: "thread-1", RunID: "run-1",
		PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, JudgeTokens: 30,
		CreatedAt: older,
	}); err != nil {
		t.Fatalf("create older: %v", err)
	}
	if err := store.Create(ctx, TokenUsageRecord{
		ThreadID: "thread-2", RunID: "run-2",
		PromptTokens: 200, CompletionTokens: 80, TotalTokens: 280,
		CreatedAt: newer,
	}); err != nil {
		t.Fatalf("create newer: %v", err)
	}

	records, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ThreadID != "thread-2" || records[1].ThreadID != "thread-1" {
		t.Fatalf("expected newest first, got %+v", records)
	}
	if records[0].TotalTokens != 280 || records[1].JudgeTokens != 30 {
		t.Fatalf("unexpected token fields: %+v", records)
	}
}

func TestTokenUsageStoreListDefaultLimit(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:token_usage_limit_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewTokenUsageStore(client)

	if err := store.Create(ctx, TokenUsageRecord{ThreadID: "t", RunID: "r", TotalTokens: 10}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// limit 0 应回退到默认 50，仍能列出。
	records, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}
