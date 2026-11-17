package logic

import (
	"context"
	"testing"

	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/types"
)

func TestUpdateMachineClearsSSHPrivateKey(t *testing.T) {
	svcCtx, machines, audit := newExecMachineTestSvc(t, nil)
	ctx := context.Background()
	created, err := machines.Create(ctx, data.MachineRecord{
		PrivateAddress: "10.0.0.1",
		SSHPrivateKey:  "test-private-key",
	})
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	authCtx := context.WithValue(ctx, "uid", "user-1")
	authCtx = context.WithValue(authCtx, "username", "admin")
	authCtx = context.WithValue(authCtx, "tenant_id", "")
	logic := NewUpdateMachineLogic(authCtx, svcCtx)

	// 清除和替换是互斥操作，避免调用方对最终状态产生歧义。
	if _, err := logic.UpdateMachine(&types.UpdateMachineInput{
		ID:                 created.ID,
		PrivateAddress:     "10.0.0.1",
		SSHPrivateKey:      "new-private-key",
		ClearSSHPrivateKey: true,
	}); err == nil {
		t.Fatal("expected clear plus replacement to be rejected")
	}

	response, err := logic.UpdateMachine(&types.UpdateMachineInput{
		ID:                 created.ID,
		PrivateAddress:     "10.0.0.1",
		ClearSSHPrivateKey: true,
	})
	if err != nil || response.ID != created.ID {
		t.Fatalf("clear machine key: response=%+v err=%v", response, err)
	}
	stored, err := machines.GetByID(ctx, created.ID)
	if err != nil || stored.SSHPrivateKey != "" {
		t.Fatalf("expected cleared private key: stored=%+v err=%v", stored, err)
	}
	entries, err := audit.List(ctx, "", 10)
	if err != nil || len(entries) != 1 || entries[0].Action != "clear_machine_ssh_key" || entries[0].ResourceID != created.ID {
		t.Fatalf("unexpected clear-key audit entries: %+v err=%v", entries, err)
	}
}
