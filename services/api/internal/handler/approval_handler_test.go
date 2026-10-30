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

func TestApprovalHandlersListAndDecideProposal(t *testing.T) {
	store := approval.NewStore()
	if err := store.Tool().Propose(context.Background(), agent.ToolProposal{
		ThreadID:   "thread-1",
		RunID:      "run-1",
		ToolCallID: "call-1",
		Name:       "create_ticket",
		Arguments:  json.RawMessage(`{"title":"网络故障","description":"无法访问"}`),
	}); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	proposal := mustList(t, store)[0]

	listRequest := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	listResponse := httptest.NewRecorder()
	ListApprovalsHandler(store).ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResponse.Code)
	}
	var list approvalListResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Approvals) != 1 || list.Approvals[0].Status != approval.StatusPending {
		t.Fatalf("unexpected approvals: %+v", list.Approvals)
	}

	approveRequest := httptest.NewRequest(http.MethodPost, "/approvals/"+proposal.ID+"/approve", nil)
	approveRequest.SetPathValue("id", proposal.ID)
	approveResponse := httptest.NewRecorder()
	ApproveApprovalHandler(store).ServeHTTP(approveResponse, approveRequest)
	if approveResponse.Code != http.StatusOK {
		t.Fatalf("unexpected approve status: %d; body=%s", approveResponse.Code, approveResponse.Body.String())
	}
	var approved approval.Approval
	if err := json.NewDecoder(approveResponse.Body).Decode(&approved); err != nil {
		t.Fatalf("decode approval: %v", err)
	}
	if approved.Status != approval.StatusApproved || approved.TicketID == "" {
		t.Fatalf("unexpected approval: %+v", approved)
	}

	duplicateResponse := httptest.NewRecorder()
	ApproveApprovalHandler(store).ServeHTTP(duplicateResponse, approveRequest)
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate approval conflict, got %d", duplicateResponse.Code)
	}
}

func TestApprovalHandlersRejectAndReturnNotFound(t *testing.T) {
	store := approval.NewStore()
	if err := store.Tool().Propose(context.Background(), agent.ToolProposal{
		ThreadID:   "thread-1",
		RunID:      "run-1",
		ToolCallID: "call-1",
		Name:       "create_ticket",
		Arguments:  json.RawMessage(`{"title":"网络故障","description":"无法访问"}`),
	}); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	proposal := mustList(t, store)[0]

	rejectRequest := httptest.NewRequest(http.MethodPost, "/approvals/"+proposal.ID+"/reject", nil)
	rejectRequest.SetPathValue("id", proposal.ID)
	rejectResponse := httptest.NewRecorder()
	RejectApprovalHandler(store).ServeHTTP(rejectResponse, rejectRequest)
	if rejectResponse.Code != http.StatusOK {
		t.Fatalf("unexpected reject status: %d", rejectResponse.Code)
	}
	var rejected approval.Approval
	if err := json.NewDecoder(rejectResponse.Body).Decode(&rejected); err != nil {
		t.Fatalf("decode rejection: %v", err)
	}
	if rejected.Status != approval.StatusRejected || rejected.TicketID != "" {
		t.Fatalf("unexpected rejection: %+v", rejected)
	}

	missingRequest := httptest.NewRequest(http.MethodPost, "/approvals/missing/approve", nil)
	missingRequest.SetPathValue("id", "missing")
	missingResponse := httptest.NewRecorder()
	ApproveApprovalHandler(store).ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing approval to return 404, got %d", missingResponse.Code)
	}
}

func mustList(t *testing.T, store approval.Store) []approval.Approval {
	t.Helper()
	approvals, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	return approvals
}
