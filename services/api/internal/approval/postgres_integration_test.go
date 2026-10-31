package approval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"nexusai/services/api/internal/agent"
)

func TestPostgresStorePersistsApprovalAndTicketAcrossReopen(t *testing.T) {
	dsn := os.Getenv("NEXUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NEXUS_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("create postgres store: %v", err)
	}
	defer func() {
		_, _ = store.db.ExecContext(context.Background(), `TRUNCATE tickets, approvals`)
		_ = store.Close()
	}()
	if _, err := store.db.ExecContext(ctx, `TRUNCATE tickets, approvals`); err != nil {
		t.Fatalf("reset integration tables: %v", err)
	}

	if err := store.Tool().Propose(ctx, agent.ToolProposal{
		ThreadID:   "thread-postgres",
		RunID:      "run-postgres",
		ToolCallID: "call-postgres",
		Name:       createTicketToolName,
		Arguments:  json.RawMessage(`{"title":"持久化验证","description":"重启后审批与工单仍需存在"}`),
	}); err != nil {
		t.Fatalf("propose ticket: %v", err)
	}
	approvals, err := store.List(ctx)
	if err != nil || len(approvals) != 1 {
		t.Fatalf("list pending approvals: %+v, %v", approvals, err)
	}
	approved, err := store.Approve(ctx, approvals[0].ID)
	if err != nil {
		t.Fatalf("approve proposal: %v", err)
	}

	var title, status string
	if err := store.db.QueryRowContext(ctx, `SELECT title, status FROM tickets WHERE id = $1`, approved.TicketID).Scan(&title, &status); err != nil {
		t.Fatalf("query created ticket: %v", err)
	}
	if title != "持久化验证" || status != "open" {
		t.Fatalf("unexpected ticket: title=%q status=%q", title, status)
	}

	reopened, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen postgres store: %v", err)
	}
	defer reopened.Close()
	persisted, err := reopened.List(ctx)
	if err != nil || len(persisted) != 1 {
		t.Fatalf("list persisted approvals: %+v, %v", persisted, err)
	}
	if persisted[0].Status != StatusApproved || persisted[0].TicketID != approved.TicketID || persisted[0].DecidedAt == nil {
		t.Fatalf("approval was not persisted: %+v", persisted[0])
	}
	tickets, err := reopened.ListTickets(ctx)
	if err != nil || len(tickets) != 1 {
		t.Fatalf("list persisted tickets: %+v, %v", tickets, err)
	}
	if tickets[0].ID != approved.TicketID || tickets[0].Title != "持久化验证" || tickets[0].Status != "open" {
		t.Fatalf("unexpected persisted ticket: %+v", tickets[0])
	}
	updated, err := reopened.UpdateTicketStatus(ctx, approved.TicketID, TicketStatusInProgress)
	if err != nil || updated.Status != TicketStatusInProgress {
		t.Fatalf("update persisted ticket status: %+v, %v", updated, err)
	}
	var persistedStatus TicketStatus
	if err := store.db.QueryRowContext(ctx, `SELECT status FROM tickets WHERE id = $1`, approved.TicketID).Scan(&persistedStatus); err != nil {
		t.Fatalf("query updated ticket status: %v", err)
	}
	if persistedStatus != TicketStatusInProgress {
		t.Fatalf("ticket status was not persisted: %q", persistedStatus)
	}
	if _, err := reopened.Approve(ctx, approved.ID); !errors.Is(err, ErrAlreadyDecided) {
		t.Fatalf("expected persisted decision to remain final, got %v", err)
	}
}
