package data

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
)

func TestUserStoreCreateAndGet(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:user_store_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewUserStore(client)

	created, err := store.Create(ctx, "alice", "hashed-password", "staff", "")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID == "" || created.Username != "alice" || created.Role != "staff" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	got, err := store.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if got.ID != created.ID || got.PasswordHash != "hashed-password" {
		t.Fatalf("unexpected user: %+v", got)
	}

	byID, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Username != "alice" {
		t.Fatalf("unexpected user by id: %+v", byID)
	}
}

func TestUserStoreGetByUsernameNotFound(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:user_store_notfound_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewUserStore(client)

	if _, err := store.GetByUsername(ctx, "nobody"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserStoreOAuth(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:user_store_oauth_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewUserStore(client)

	if _, err := store.GetByGithubID(ctx, "12345"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for missing github id, got %v", err)
	}

	created, err := store.CreateOAuth(ctx, "alice", "user", "12345")
	if err != nil {
		t.Fatalf("create oauth user: %v", err)
	}
	if created.ID == "" || created.Username != "alice" || created.Role != "user" || created.TenantID != created.ID {
		t.Fatalf("unexpected oauth user: %+v", created)
	}

	got, err := store.GetByGithubID(ctx, "12345")
	if err != nil {
		t.Fatalf("get by github id: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("unexpected user by github id: %+v", got)
	}
}
