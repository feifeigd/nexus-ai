package data

import (
	"context"
	"errors"
	"strings"
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

	if _, err := store.GetByOAuthSubject(ctx, "github", "12345"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for missing oauth subject, got %v", err)
	}

	created, err := store.CreateOAuth(ctx, "github", "12345", "alice", "user")
	if err != nil {
		t.Fatalf("create oauth user: %v", err)
	}
	if created.ID == "" || created.Username != "alice" || created.Role != "user" || created.TenantID == "" {
		t.Fatalf("unexpected oauth user: %+v", created)
	}

	// 个人租户应已创建，且 owner 是自己、名称含用户名。
	tenant, err := client.Tenant.Get(ctx, created.TenantID)
	if err != nil {
		t.Fatalf("load personal tenant: %v", err)
	}
	if tenant.OwnerID != created.ID || !strings.Contains(tenant.Name, "alice") {
		t.Fatalf("unexpected personal tenant: %+v", tenant)
	}

	got, err := store.GetByOAuthSubject(ctx, "github", "12345")
	if err != nil {
		t.Fatalf("get by oauth subject: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("unexpected user by oauth subject: %+v", got)
	}

	// 跨平台同 subject 应是不同用户。
	if _, err := store.GetByOAuthSubject(ctx, "gitee", "12345"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected gitee subject 12345 to be a separate identity, got %v", err)
	}
}
