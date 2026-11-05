package logic

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func newTenantTestSvc(t *testing.T) (*svc.ServiceContext, *data.UserStore) {
	t.Helper()
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:tenant_logic_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite: %v", err)
	}
	t.Cleanup(closeData)
	users := data.NewUserStore(client)
	tenants := data.NewTenantStore(client)
	svcCtx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, nil, nil, nil, users, tenants)
	return svcCtx, users
}

func TestCreateTenantSwitchesOwner(t *testing.T) {
	svcCtx, users := newTenantTestSvc(t)
	ctx := context.Background()

	owner, err := users.Create(ctx, "owner", "hash", "user", "")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	personalTenantID := owner.TenantID

	l := NewCreateTenantLogic(context.WithValue(ctx, "uid", owner.ID), svcCtx)
	resp, err := l.CreateTenant(&types.CreateTenantInput{Name: "  蓝雀科技  "})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if resp.TenantID == "" || resp.Name != "蓝雀科技" {
		t.Fatalf("unexpected tenant: %+v", resp)
	}
	if resp.TenantID == personalTenantID {
		t.Fatalf("expected a new org tenant id, got the personal tenant id")
	}
	got, err := users.GetByID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("get owner: %v", err)
	}
	if got.TenantID != resp.TenantID {
		t.Fatalf("owner tenant not switched: got %q want %q", got.TenantID, resp.TenantID)
	}
}

func TestAddTenantMember(t *testing.T) {
	svcCtx, users := newTenantTestSvc(t)
	ctx := context.Background()

	owner, _ := users.Create(ctx, "owner", "hash", "user", "")
	member, _ := users.Create(ctx, "member", "hash", "user", "")

	l := NewCreateTenantLogic(context.WithValue(ctx, "uid", owner.ID), svcCtx)
	tenant, err := l.CreateTenant(&types.CreateTenantInput{Name: "蓝雀科技"})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	am := NewAddTenantMemberLogic(context.WithValue(ctx, "uid", owner.ID), svcCtx)
	if _, err := am.AddTenantMember(&types.AddTenantMemberInput{ID: tenant.TenantID, Username: "member"}); err != nil {
		t.Fatalf("add member: %v", err)
	}
	got, _ := users.GetByID(ctx, member.ID)
	if got.TenantID != tenant.TenantID {
		t.Fatalf("member tenant not switched: got %q want %q", got.TenantID, tenant.TenantID)
	}

	// 非 owner 不能添加成员。
	other, _ := users.Create(ctx, "other", "hash", "user", "")
	am2 := NewAddTenantMemberLogic(context.WithValue(ctx, "uid", other.ID), svcCtx)
	if _, err := am2.AddTenantMember(&types.AddTenantMemberInput{ID: tenant.TenantID, Username: "member"}); err == nil {
		t.Fatal("expected non-owner add to fail")
	}
}
