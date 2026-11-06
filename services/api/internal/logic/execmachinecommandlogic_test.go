package logic

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/sshx"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

type fakeSSHRunner struct {
	result  sshx.Result
	err     error
	gotHost string
	gotKey  []byte
	gotCmd  string
}

func (f *fakeSSHRunner) Exec(_ context.Context, host string, privateKey []byte, command string) (sshx.Result, error) {
	f.gotHost, f.gotKey, f.gotCmd = host, privateKey, command
	return f.result, f.err
}

func newExecMachineTestSvc(t *testing.T, runner sshx.Runner) (*svc.ServiceContext, *data.MachineStore, *data.AuditStore) {
	t.Helper()
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:exec_machine_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite: %v", err)
	}
	t.Cleanup(closeData)
	svcCtx := svc.NewServiceContext(&config.Config{
		Auth: config.AuthConfig{AccessSecret: "test-secret-123"},
		SSH:  config.SSHConfig{AllowedCommands: []string{"uptime", "df", "tail"}},
	}, svc.ServiceContextDeps{
		Machines:  data.NewMachineStore(client),
		Audit:     data.NewAuditStore(client),
		SSHRunner: runner,
	})
	return svcCtx, data.NewMachineStore(client), data.NewAuditStore(client)
}

func TestExecMachineCommand(t *testing.T) {
	fakeRunner := &fakeSSHRunner{result: sshx.Result{Stdout: "ok\n", ExitCode: 0}}
	svcCtx, machines, audit := newExecMachineTestSvc(t, fakeRunner)
	ctx := context.Background()

	withKey, err := machines.Create(ctx, data.MachineRecord{
		PrivateAddress: "10.0.0.1",
		SSHPrivateKey:  "test-private-key",
	})
	if err != nil {
		t.Fatalf("create machine with key: %v", err)
	}
	noKey, err := machines.Create(ctx, data.MachineRecord{PrivateAddress: "10.0.0.2"})
	if err != nil {
		t.Fatalf("create machine without key: %v", err)
	}

	authCtx := context.WithValue(ctx, "uid", "user-1")
	authCtx = context.WithValue(authCtx, "username", "admin")
	authCtx = context.WithValue(authCtx, "tenant_id", "")

	l := NewExecMachineCommandLogic(authCtx, svcCtx)

	t.Run("success", func(t *testing.T) {
		resp, err := l.ExecMachineCommand(&types.ExecMachineCommandInput{ID: withKey.ID, Command: "uptime"})
		if err != nil {
			t.Fatalf("exec: %v", err)
		}
		if resp.Stdout != "ok\n" || resp.Stderr != "" || resp.ExitCode != 0 {
			t.Fatalf("unexpected response: %+v", resp)
		}
		if fakeRunner.gotHost != "10.0.0.1" || string(fakeRunner.gotKey) != "test-private-key" || fakeRunner.gotCmd != "uptime" {
			t.Fatalf("runner got host=%q key=%q cmd=%q", fakeRunner.gotHost, fakeRunner.gotKey, fakeRunner.gotCmd)
		}
		entries, err := audit.List(ctx, "", 10)
		if err != nil || len(entries) != 1 || entries[0].Action != "run_ssh_command" || entries[0].ResourceID != withKey.ID {
			t.Fatalf("unexpected audit entries: %+v err=%v", entries, err)
		}
	})

	t.Run("command not allowed", func(t *testing.T) {
		if _, err := l.ExecMachineCommand(&types.ExecMachineCommandInput{ID: withKey.ID, Command: "rm -rf /"}); err == nil {
			t.Fatal("expected whitelist rejection")
		}
	})

	t.Run("no private key", func(t *testing.T) {
		if _, err := l.ExecMachineCommand(&types.ExecMachineCommandInput{ID: noKey.ID, Command: "uptime"}); err == nil {
			t.Fatal("expected missing private key error")
		}
	})

	t.Run("machine not found", func(t *testing.T) {
		if _, err := l.ExecMachineCommand(&types.ExecMachineCommandInput{ID: "machine_nope", Command: "uptime"}); err == nil {
			t.Fatal("expected not found error")
		}
	})
}
