package approval

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
)

func TestProposalRecordsPendingApprovalWithoutExecuting(t *testing.T) {
	store := NewStore()
	approval := proposeTicket(t, store, `{"title":"  网络故障  ","description":"  无法访问办公系统  "}`)

	if approval.ID == "" || approval.Status != StatusPending {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if approval.ToolCallID != "call-1" || approval.ToolName != createTicketToolName {
		t.Fatalf("unexpected tool correlation: %+v", approval)
	}
	if approval.Arguments.Title != "网络故障" || approval.Arguments.Description != "无法访问办公系统" {
		t.Fatalf("arguments were not validated and trimmed: %+v", approval.Arguments)
	}
	if approval.TicketID != "" || len(store.tickets) != 0 {
		t.Fatal("a proposal must not execute the ticket operation")
	}
}

func TestApproveCreatesTicketExactlyOnce(t *testing.T) {
	store := NewStore()
	proposal := proposeTicket(t, store, `{"title":"网络故障","description":"无法访问办公系统"}`)

	approved, err := store.Approve(context.Background(), proposal.ID)
	if err != nil {
		t.Fatalf("approve proposal: %v", err)
	}
	if approved.Status != StatusApproved || approved.TicketID == "" || approved.DecidedAt == nil {
		t.Fatalf("unexpected approval result: %+v", approved)
	}
	ticket, ok := store.tickets[approved.TicketID]
	if !ok || ticket.Title != proposal.Arguments.Title || ticket.Status != "open" {
		t.Fatalf("ticket was not created from validated arguments: %+v", ticket)
	}

	if _, err := store.Approve(context.Background(), proposal.ID); !errors.Is(err, ErrAlreadyDecided) {
		t.Fatalf("expected duplicate approval to fail, got %v", err)
	}
	if len(store.tickets) != 1 {
		t.Fatal("duplicate approval must not create another ticket")
	}
}

func TestRejectDoesNotExecuteAndCannotBeApproved(t *testing.T) {
	store := NewStore()
	proposal := proposeTicket(t, store, `{"title":"网络故障","description":"无法访问办公系统"}`)

	rejected, err := store.Reject(context.Background(), proposal.ID)
	if err != nil {
		t.Fatalf("reject proposal: %v", err)
	}
	if rejected.Status != StatusRejected || rejected.TicketID != "" || len(store.tickets) != 0 {
		t.Fatalf("rejection must not create a ticket: %+v", rejected)
	}
	if _, err := store.Approve(context.Background(), proposal.ID); !errors.Is(err, ErrAlreadyDecided) {
		t.Fatalf("expected rejected approval to stay final, got %v", err)
	}
}

func TestProposalRejectsInvalidArguments(t *testing.T) {
	tests := map[string]string{
		"malformed JSON":       `{"title":`,
		"unknown field":        `{"title":"标题","description":"描述","admin":true}`,
		"missing title":        `{"description":"描述"}`,
		"missing description":  `{"title":"标题"}`,
		"null description":     `{"title":"标题","description":null}`,
		"blank title":          `{"title":"  ","description":"描述"}`,
		"multiple JSON values": `{"title":"标题","description":"描述"} {}`,
		"long title":           `{"title":"` + strings.Repeat("界", 201) + `","description":"描述"}`,
	}
	for name, arguments := range tests {
		t.Run(name, func(t *testing.T) {
			store := NewStore()
			err := store.Tool().Propose(context.Background(), agent.ToolProposal{
				ThreadID:   "thread-1",
				RunID:      "run-1",
				ToolCallID: "call-1",
				Name:       createTicketToolName,
				Arguments:  json.RawMessage(arguments),
			})
			if err == nil {
				t.Fatal("expected invalid arguments to be rejected")
			}
			if len(store.List()) != 0 || len(store.tickets) != 0 {
				t.Fatal("invalid arguments must not create approvals or tickets")
			}
		})
	}
}

func proposeTicket(t *testing.T, store *Store, arguments string) Approval {
	t.Helper()
	if err := store.Tool().Propose(context.Background(), agent.ToolProposal{
		ThreadID:   "thread-1",
		RunID:      "run-1",
		ToolCallID: "call-1",
		Name:       createTicketToolName,
		Arguments:  json.RawMessage(arguments),
	}); err != nil {
		t.Fatalf("propose ticket: %v", err)
	}
	approvals := store.List()
	if len(approvals) != 1 {
		t.Fatalf("expected one approval, got %d", len(approvals))
	}
	return approvals[0]
}
