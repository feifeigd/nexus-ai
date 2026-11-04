package approval

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"nexusai/services/api/internal/agent"
)

func TestEntStoreTenantIsolation(t *testing.T) {
	store := newTestStore(t)
	ctxA := context.WithValue(context.Background(), "tenant_id", "tenant-a")
	ctxB := context.WithValue(context.Background(), "tenant_id", "tenant-b")

	propose := func(ctx context.Context, toolCallID string) {
		t.Helper()
		if err := store.Tool().Propose(ctx, agent.ToolProposal{
			ThreadID:   "thread-" + toolCallID,
			RunID:      "run-" + toolCallID,
			ToolCallID: toolCallID,
			Name:       createTicketToolName,
			Arguments:  json.RawMessage(`{"title":"工单","description":"描述"}`),
		}); err != nil {
			t.Fatalf("propose %s: %v", toolCallID, err)
		}
	}
	propose(ctxA, "call-a")
	propose(ctxB, "call-b")

	// 列表按租户隔离。
	approvalsA, err := store.List(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("list tenant-a approvals: %v", err)
	}
	if len(approvalsA) != 1 || approvalsA[0].ToolCallID != "call-a" {
		t.Fatalf("expected only tenant-a approvals, got %+v", approvalsA)
	}
	approvalsB, err := store.List(context.Background(), "tenant-b")
	if err != nil {
		t.Fatalf("list tenant-b approvals: %v", err)
	}
	if len(approvalsB) != 1 || approvalsB[0].ToolCallID != "call-b" {
		t.Fatalf("expected only tenant-b approvals, got %+v", approvalsB)
	}

	// 跨租户批准返回 not found，不创建工单。
	if _, err := store.Approve(context.Background(), approvalsB[0].ID, "tenant-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected cross-tenant approve to return not found, got %v", err)
	}

	// 本租户批准成功，且工单落在提出者租户。
	approved, err := store.Approve(context.Background(), approvalsA[0].ID, "tenant-a")
	if err != nil {
		t.Fatalf("approve tenant-a proposal: %v", err)
	}
	if approved.TenantID != "tenant-a" {
		t.Fatalf("expected approval to keep tenant-a, got %q", approved.TenantID)
	}
	ticketsA, err := store.ListTickets(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("list tenant-a tickets: %v", err)
	}
	if len(ticketsA) != 1 || ticketsA[0].ID != approved.TicketID {
		t.Fatalf("expected the approved ticket in tenant-a, got %+v", ticketsA)
	}
	ticketsB, err := store.ListTickets(context.Background(), "tenant-b")
	if err != nil {
		t.Fatalf("list tenant-b tickets: %v", err)
	}
	if len(ticketsB) != 0 {
		t.Fatalf("tenant-b must not see tenant-a's ticket, got %+v", ticketsB)
	}
}
