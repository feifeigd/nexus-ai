package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
)

func TestListTicketsHandlerReturnsCreatedTickets(t *testing.T) {
	store := approval.NewStore()
	if err := store.Tool().Propose(context.Background(), agent.ToolProposal{
		ThreadID:   "thread-1",
		RunID:      "run-1",
		ToolCallID: "call-1",
		Name:       "create_ticket",
		Arguments:  json.RawMessage(`{"title":"网络故障","description":"无法访问办公系统"}`),
	}); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	proposals, err := store.List(context.Background())
	if err != nil || len(proposals) != 1 {
		t.Fatalf("list proposals: %+v, %v", proposals, err)
	}
	approved, err := store.Approve(context.Background(), proposals[0].ID)
	if err != nil {
		t.Fatalf("approve proposal: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/tickets", nil)
	response := httptest.NewRecorder()
	ListTicketsHandler(store).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", response.Code)
	}
	var list ticketListResponse
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		t.Fatalf("decode ticket list: %v", err)
	}
	if len(list.Tickets) != 1 {
		t.Fatalf("expected one ticket, got %+v", list.Tickets)
	}
	ticket := list.Tickets[0]
	if ticket.ID != approved.TicketID || ticket.Title != "网络故障" || ticket.Description != "无法访问办公系统" || ticket.Status != "open" {
		t.Fatalf("unexpected ticket: %+v", ticket)
	}
}
