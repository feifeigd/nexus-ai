package approval

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/testpg"

	"entgo.io/ent/dialect"
	_ "github.com/lib/pq"
)

func TestEntStorePersistsApprovalAndTicketAcrossReopen(t *testing.T) {
	baseDSN := os.Getenv("NEXUS_TEST_POSTGRES_DSN")
	if baseDSN == "" {
		t.Skip("NEXUS_TEST_POSTGRES_DSN is not configured")
	}
	dsn, dropDB, err := testpg.NewDatabase(baseDSN)
	if err != nil {
		t.Fatalf("create disposable postgres database: %v", err)
	}
	defer dropDB()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &data.Data{Database: data.Database{
		Driver:       dialect.Postgres,
		Source:       dsn,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		AutoMigrate:  true,
	}}
	client, closeData, err := data.NewData(config)
	if err != nil {
		t.Fatalf("create postgres data source: %v", err)
	}
	store := NewEntStore(client, dialect.Postgres)
	rawDB, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		closeData()
		t.Fatalf("open integration cleanup connection: %v", err)
	}
	defer func() {
		_, _ = rawDB.ExecContext(context.Background(), `TRUNCATE tickets, approvals`)
		_ = rawDB.Close()
		closeData()
	}()
	if _, err := rawDB.ExecContext(ctx, `TRUNCATE tickets, approvals`); err != nil {
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
	if err := rawDB.QueryRowContext(ctx, `SELECT title, status FROM tickets WHERE id = $1`, approved.TicketID).Scan(&title, &status); err != nil {
		t.Fatalf("query created ticket: %v", err)
	}
	if title != "持久化验证" || status != "open" {
		t.Fatalf("unexpected ticket: title=%q status=%q", title, status)
	}

	reopenedClient, closeReopened, err := data.NewData(config)
	if err != nil {
		t.Fatalf("reopen postgres data source: %v", err)
	}
	defer closeReopened()
	reopened := NewEntStore(reopenedClient, dialect.Postgres)
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
	if err := rawDB.QueryRowContext(ctx, `SELECT status FROM tickets WHERE id = $1`, approved.TicketID).Scan(&persistedStatus); err != nil {
		t.Fatalf("query updated ticket status: %v", err)
	}
	if persistedStatus != TicketStatusInProgress {
		t.Fatalf("ticket status was not persisted: %q", persistedStatus)
	}
	if _, err := reopened.Approve(ctx, approved.ID); !errors.Is(err, ErrAlreadyDecided) {
		t.Fatalf("expected persisted decision to remain final, got %v", err)
	}

	if err := reopened.Tool().Propose(ctx, agent.ToolProposal{
		ThreadID:   "thread-rejected",
		RunID:      "run-rejected",
		ToolCallID: "call-rejected",
		Name:       createTicketToolName,
		Arguments:  json.RawMessage(`{"title":"拒绝验证","description":"拒绝后不应创建工单"}`),
	}); err != nil {
		t.Fatalf("propose rejected ticket: %v", err)
	}
	approvals, err = reopened.List(ctx)
	if err != nil {
		t.Fatalf("list approvals before rejection: %v", err)
	}
	var rejectedID string
	for _, candidate := range approvals {
		if candidate.ToolCallID == "call-rejected" {
			rejectedID = candidate.ID
		}
	}
	if rejectedID == "" {
		t.Fatal("rejected proposal was not persisted")
	}
	rejected, err := reopened.Reject(ctx, rejectedID)
	if err != nil || rejected.Status != StatusRejected || rejected.DecidedAt == nil {
		t.Fatalf("reject persisted proposal: %+v, %v", rejected, err)
	}
	tickets, err = reopened.ListTickets(ctx)
	if err != nil || len(tickets) != 1 {
		t.Fatalf("rejection must not create another ticket: %+v, %v", tickets, err)
	}
}
