package data

import (
	"context"
	"errors"
	"strings"
	"testing"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/machinekey"
)

const initialMachinePrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"

func TestMachineStoreCRUDEncryptsPrivateKeys(t *testing.T) {
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
	keys, err := machinekey.NewBox("test-secret-123")
	if err != nil {
		t.Fatalf("create machine key box: %v", err)
	}
	ctx := context.Background()
	store := NewMachineStore(client, keys)

	created, err := store.Create(ctx, MachineRecord{
		Name:           "web-1",
		PrivateAddress: "10.0.0.1",
		PublicAddress:  "1.2.3.4",
		Region:         "cn-east",
		RootPublicKey:  "ssh-rsa AAAA",
		SSHPrivateKey:  initialMachinePrivateKey,
		JoinedK8s:      false,
	})
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	if created.ID == "" || created.PrivateAddress != "10.0.0.1" || created.PublicAddress != "1.2.3.4" || created.JoinedK8s || created.SSHPrivateKey != initialMachinePrivateKey {
		t.Fatalf("unexpected created machine: %+v", created)
	}

	// 直接读取 Ent 原始行，验证落库值不是业务层返回的明文。
	raw, err := client.Machine.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("read raw machine: %v", err)
	}
	initialCiphertext := derefOrEmpty(raw.SSHPrivateKey)
	if !machinekey.IsEncrypted(initialCiphertext) || strings.Contains(initialCiphertext, "BEGIN OPENSSH") {
		t.Fatalf("expected encrypted private key at rest, got %q", initialCiphertext)
	}

	list, err := store.List(ctx)
	if err != nil || len(list) != 1 || list[0].ID != created.ID || list[0].SSHPrivateKey != "" {
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
	if !updated.JoinedK8s || updated.PublicAddress != "" || updated.SSHPrivateKey != initialMachinePrivateKey {
		t.Fatalf("unexpected updated machine: %+v", updated)
	}

	// 换新私钥时重新加密，业务层仍只拿到新明文。
	const rotatedPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nrotated\n-----END OPENSSH PRIVATE KEY-----"
	rotated, err := store.Update(ctx, MachineRecord{
		ID:             created.ID,
		Name:           "web-1",
		PrivateAddress: "10.0.0.1",
		JoinedK8s:      true,
		SSHPrivateKey:  rotatedPrivateKey,
	})
	if err != nil {
		t.Fatalf("rotate machine key: %v", err)
	}
	if rotated.SSHPrivateKey != rotatedPrivateKey {
		t.Fatalf("expected rotated private key, got %+v", rotated.SSHPrivateKey)
	}
	raw, err = client.Machine.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("read rotated raw machine: %v", err)
	}
	rotatedCiphertext := derefOrEmpty(raw.SSHPrivateKey)
	if !machinekey.IsEncrypted(rotatedCiphertext) || rotatedCiphertext == initialCiphertext || strings.Contains(rotatedCiphertext, "rotated") {
		t.Fatalf("expected re-encrypted private key, got %q", rotatedCiphertext)
	}

	// 只有显式清除标志可以删除私钥，机器本身和其它字段保持存在。
	cleared, err := store.Update(ctx, MachineRecord{
		ID:                 created.ID,
		Name:               "web-1",
		PrivateAddress:     "10.0.0.1",
		JoinedK8s:          true,
		ClearSSHPrivateKey: true,
	})
	if err != nil || cleared.SSHPrivateKey != "" {
		t.Fatalf("clear machine private key: cleared=%+v err=%v", cleared, err)
	}
	raw, err = client.Machine.Get(ctx, created.ID)
	if err != nil || derefOrEmpty(raw.SSHPrivateKey) != "" {
		t.Fatalf("expected raw private key to be cleared: raw=%+v err=%v", raw, err)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete machine: %v", err)
	}
	if _, err := store.GetByID(ctx, created.ID); !errors.Is(err, ErrMachineNotFound) {
		t.Fatalf("expected deleted machine to be absent, got %v", err)
	}
}

func TestMachineStoreReadsLegacyPlaintextPrivateKey(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:machine_store_legacy_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	keys, err := machinekey.NewBox("test-secret-123")
	if err != nil {
		t.Fatalf("create machine key box: %v", err)
	}
	ctx := context.Background()
	if _, err := client.Machine.Create().
		SetID("machine_legacy").
		SetPrivateAddress("10.0.0.9").
		SetJoinedK8s(false).
		SetSSHPrivateKey(initialMachinePrivateKey).
		Save(ctx); err != nil {
		t.Fatalf("insert legacy machine: %v", err)
	}

	record, err := NewMachineStore(client, keys).GetByID(ctx, "machine_legacy")
	if err != nil || record.SSHPrivateKey != initialMachinePrivateKey {
		t.Fatalf("read legacy plaintext private key: record=%+v err=%v", record, err)
	}
}

func TestMachineStoreReplacesPrivateKeyAfterSecretChange(t *testing.T) {
	client, closeData, err := NewData(&Data{Database: Database{
		Driver:       dialect.SQLite,
		Source:       "file:machine_store_secret_change_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	oldKeys, err := machinekey.NewBox("old-secret")
	if err != nil {
		t.Fatalf("create old machine key box: %v", err)
	}
	newKeys, err := machinekey.NewBox("new-secret")
	if err != nil {
		t.Fatalf("create new machine key box: %v", err)
	}
	ctx := context.Background()
	created, err := NewMachineStore(client, oldKeys).Create(ctx, MachineRecord{
		PrivateAddress: "10.0.0.10",
		SSHPrivateKey:  initialMachinePrivateKey,
	})
	if err != nil {
		t.Fatalf("create machine with old secret: %v", err)
	}

	store := NewMachineStore(client, newKeys)
	list, err := store.List(ctx)
	if err != nil || len(list) != 1 || list[0].SSHPrivateKey != "" {
		t.Fatalf("old ciphertext must not block machine list: list=%+v err=%v", list, err)
	}
	if _, err := store.GetByID(ctx, created.ID); err == nil {
		t.Fatal("expected old ciphertext to fail authentication with the new secret")
	}

	const replacement = "-----BEGIN OPENSSH PRIVATE KEY-----\nreplacement\n-----END OPENSSH PRIVATE KEY-----"
	updated, err := store.Update(ctx, MachineRecord{
		ID:             created.ID,
		PrivateAddress: "10.0.0.10",
		SSHPrivateKey:  replacement,
	})
	if err != nil || updated.SSHPrivateKey != replacement {
		t.Fatalf("replace private key after secret change: updated=%+v err=%v", updated, err)
	}
}
