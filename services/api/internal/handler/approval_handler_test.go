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
	"nexusai/services/api/internal/types"
)

func TestApprovalHandlersListAndDecideProposal(t *testing.T) {
	store := newTestApprovalStore(t)
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
	listApprovalsHandler(newTestServiceContext(store, nil)).ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResponse.Code)
	}
	var list types.ApprovalListResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Approvals) != 1 || list.Approvals[0].Status != string(approval.StatusPending) {
		t.Fatalf("unexpected approvals: %+v", list.Approvals)
	}

	approveRequest := httptest.NewRequest(http.MethodPost, "/approvals/"+proposal.ID+"/approve", nil)
	approveRequest = withPathID(approveRequest, proposal.ID)
	approveResponse := httptest.NewRecorder()
	approveApprovalHandler(newTestServiceContext(store, nil)).ServeHTTP(approveResponse, approveRequest)
	if approveResponse.Code != http.StatusOK {
		t.Fatalf("unexpected approve status: %d; body=%s", approveResponse.Code, approveResponse.Body.String())
	}
	var approved types.Approval
	if err := json.NewDecoder(approveResponse.Body).Decode(&approved); err != nil {
		t.Fatalf("decode approval: %v", err)
	}
	if approved.Status != string(approval.StatusApproved) || approved.TicketID == "" {
		t.Fatalf("unexpected approval: %+v", approved)
	}

	duplicateResponse := httptest.NewRecorder()
	approveApprovalHandler(newTestServiceContext(store, nil)).ServeHTTP(duplicateResponse, approveRequest)
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate approval conflict, got %d", duplicateResponse.Code)
	}
}

func TestApprovalHandlersRejectAndReturnNotFound(t *testing.T) {
	store := newTestApprovalStore(t)
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
	rejectRequest = withPathID(rejectRequest, proposal.ID)
	rejectResponse := httptest.NewRecorder()
	rejectApprovalHandler(newTestServiceContext(store, nil)).ServeHTTP(rejectResponse, rejectRequest)
	if rejectResponse.Code != http.StatusOK {
		t.Fatalf("unexpected reject status: %d", rejectResponse.Code)
	}
	var rejected types.Approval
	if err := json.NewDecoder(rejectResponse.Body).Decode(&rejected); err != nil {
		t.Fatalf("decode rejection: %v", err)
	}
	if rejected.Status != string(approval.StatusRejected) || rejected.TicketID != "" {
		t.Fatalf("unexpected rejection: %+v", rejected)
	}

	missingRequest := httptest.NewRequest(http.MethodPost, "/approvals/missing/approve", nil)
	missingRequest = withPathID(missingRequest, "missing")
	missingResponse := httptest.NewRecorder()
	approveApprovalHandler(newTestServiceContext(store, nil)).ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing approval to return 404, got %d", missingResponse.Code)
	}
}

func TestApprovalAndTicketRoutesReadGoZeroPathID(t *testing.T) {
	store := newTestApprovalStore(t)
	if err := store.Tool().Propose(context.Background(), agent.ToolProposal{
		ThreadID:   "thread-route",
		RunID:      "run-route",
		ToolCallID: "call-route",
		Name:       "create_ticket",
		Arguments:  json.RawMessage(`{"title":"路由工单","description":"验证真实 go-zero 路径参数"}`),
	}); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	proposal := mustList(t, store)[0]
	baseURL := startTestServer(t, store)

	response, err := http.Post(baseURL+"/approvals/"+proposal.ID+"/approve", "", nil)
	if err != nil {
		t.Fatalf("approve through real route: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected route status: %d", response.StatusCode)
	}
	var approved types.Approval
	if err := json.NewDecoder(response.Body).Decode(&approved); err != nil {
		t.Fatalf("decode route approval: %v", err)
	}
	if approved.Status != string(approval.StatusApproved) || approved.TicketID == "" {
		t.Fatalf("unexpected route approval: %+v", approved)
	}

	ticketRequest, err := http.NewRequest(
		http.MethodPatch,
		baseURL+"/tickets/"+approved.TicketID+"/status",
		strings.NewReader(`{"status":"in_progress"}`),
	)
	if err != nil {
		t.Fatalf("create ticket route request: %v", err)
	}
	ticketResponse, err := http.DefaultClient.Do(ticketRequest)
	if err != nil {
		t.Fatalf("update ticket through real route: %v", err)
	}
	defer ticketResponse.Body.Close()
	if ticketResponse.StatusCode != http.StatusOK {
		t.Fatalf("unexpected ticket route status: %d", ticketResponse.StatusCode)
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
