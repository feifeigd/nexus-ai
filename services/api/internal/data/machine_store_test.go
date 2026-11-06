package data

import (
	"context"
	"errors"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
)

func TestMachineStoreCRUD(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:machine_store_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	ctx := context.Background()
	store := NewMachineStore(client)

	created, err := store.Create(ctx, MachineRecord{
		Name:           "web-1",
		PrivateAddress: "10.0.0.1",
		PublicAddress:  "1.2.3.4",
		Region:         "cn-east",
		RootPublicKey:  "ssh-rsa AAAA",
		SSHPrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		JoinedK8s:      false,
	})
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	if created.ID == "" || created.PrivateAddress != "10.0.0.1" || created.PublicAddress != "1.2.3.4" || created.JoinedK8s || created.SSHPrivateKey == "" {
		t.Fatalf("unexpected created machine: %+v", created)
	}

	list, err := store.List(ctx)
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected machine list: %+v err=%v", list, err)
	}

	// 更新：加入集群 + 清空公网地址。私钥留空应保持原值不变（只写不读）。
	updated, err := store.Update(ctx, MachineRecord{
		ID:             created.ID,
		Name:           "web-1",
		PrivateAddress: "10.0.0.1",
		JoinedK8s:      true,
	})
	if err != nil {
		t.Fatalf("update machine: %v", err)
	}
	if !updated.JoinedK8s || updated.PublicAddress != "" || updated.SSHPrivateKey == "" {
		t.Fatalf("unexpected updated machine: %+v", updated)
	}

	// 换新私钥。
	rotated, err := store.Update(ctx, MachineRecord{
		ID:             created.ID,
		Name:           "web-1",
		PrivateAddress: "10.0.0.1",
		JoinedK8s:      true,
		SSHPrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\nrotated\n-----END OPENSSH PRIVATE KEY-----",
	})
	if err != nil {
		t.Fatalf("rotate machine key: %v", err)
	}
	if !strings.Contains(rotated.SSHPrivateKey, "rotated") {
		t.Fatalf("expected rotated private key, got %+v", rotated.SSHPrivateKey)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete machine: %v", err)
	}
	if _, err := store.GetByID(ctx, created.ID); !errors.Is(err, ErrMachineNotFound) {
		t.Fatalf("expected deleted machine to be absent, got %v", err)
	}
}
