package approval

import (
	"testing"
	"time"

	dataent "nexusai/services/api/internal/data/ent"
)

func TestApprovalFromEntMapsDomainFields(t *testing.T) {
	ticketID := "ticket-1"
	decidedAt := time.Date(2026, 11, 1, 10, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	row := &dataent.Approval{
		ID:         "approval-1",
		ThreadID:   "thread-1",
		RunID:      "run-1",
		ToolCallID: "call-1",
		ToolName:   createTicketToolName,
		Arguments: map[string]any{
			"title":       "网络故障",
			"description": "无法访问办公系统",
		},
		Status:    string(StatusApproved),
		TicketID:  &ticketID,
		CreatedAt: decidedAt,
		DecidedAt: &decidedAt,
	}

	approval, err := approvalFromEnt(row)
	if err != nil {
		t.Fatalf("map ent approval: %v", err)
	}
	if approval.ID != row.ID || approval.ThreadID != row.ThreadID || approval.RunID != row.RunID {
		t.Fatalf("unexpected correlation fields: %+v", approval)
	}
	if approval.Arguments.Title != "网络故障" || approval.Arguments.Description != "无法访问办公系统" {
		t.Fatalf("unexpected arguments: %+v", approval.Arguments)
	}
	if approval.Status != StatusApproved || approval.TicketID != ticketID {
		t.Fatalf("unexpected decision fields: %+v", approval)
	}
	if !approval.CreatedAt.Equal(decidedAt) || approval.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at was not converted to UTC: %v", approval.CreatedAt)
	}
	if approval.DecidedAt == nil || !approval.DecidedAt.Equal(decidedAt) || approval.DecidedAt.Location() != time.UTC {
		t.Fatalf("decided_at was not converted to UTC: %v", approval.DecidedAt)
	}
}

func TestApprovalFromEntMapsEmptyDecisionFields(t *testing.T) {
	approval, err := approvalFromEnt(&dataent.Approval{
		Arguments: map[string]any{"title": "标题", "description": "描述"},
		Status:    string(StatusPending),
	})
	if err != nil {
		t.Fatalf("map pending ent approval: %v", err)
	}
	if approval.TicketID != "" || approval.DecidedAt != nil {
		t.Fatalf("pending approval must not contain decision fields: %+v", approval)
	}
}
