package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestUpdateTicketStatusHandlerEnforcesWorkflow(t *testing.T) {
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

	invalidRequest := httptest.NewRequest(http.MethodPatch, "/tickets/"+approved.TicketID+"/status", strings.NewReader(`{"status":"resolved"}`))
	invalidResponse := httptest.NewRecorder()
	UpdateTicketStatusHandler(store).ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusConflict {
		t.Fatalf("expected skipped transition to return 409, got %d", invalidResponse.Code)
	}

	request := httptest.NewRequest(http.MethodPatch, "/tickets/"+approved.TicketID+"/status", strings.NewReader(`{"status":"in_progress"}`))
	response := httptest.NewRecorder()
	UpdateTicketStatusHandler(store).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d; body=%s", response.Code, response.Body.String())
	}
	var ticket approval.Ticket
	if err := json.NewDecoder(response.Body).Decode(&ticket); err != nil {
		t.Fatalf("decode updated ticket: %v", err)
	}
	if ticket.Status != approval.TicketStatusInProgress {
		t.Fatalf("unexpected ticket status: %+v", ticket)
	}

	unknownRequest := httptest.NewRequest(http.MethodPatch, "/tickets/"+approved.TicketID+"/status", strings.NewReader(`{"status":"closed"}`))
	unknownResponse := httptest.NewRecorder()
	UpdateTicketStatusHandler(store).ServeHTTP(unknownResponse, unknownRequest)
	if unknownResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown status to return 400, got %d", unknownResponse.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodPatch, "/tickets/missing/status", strings.NewReader(`{"status":"in_progress"}`))
	missingResponse := httptest.NewRecorder()
	UpdateTicketStatusHandler(store).ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing ticket to return 404, got %d", missingResponse.Code)
	}
}
