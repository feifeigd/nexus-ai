package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"nexusai/services/api/internal/types"
)

func TestOpenAIRunnerStreamsToolCallAndRecordsProposal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(request.Tools) != 1 || request.Tools[0].Function.Name != "create_ticket" {
			t.Fatalf("model did not receive the server tool definition: %+v", request.Tools)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"type\":\"function\",\"function\":{\"name\":\"create_ticket\",\"arguments\":\"{\\\"title\\\":\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"网络故障\\\",\\\"description\\\":\\\"无法访问\\\"}\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
	}))
	defer server.Close()

	var proposal ToolProposal
	runner, err := NewOpenAIRunnerWithTools(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client(), []Tool{{
		Name:        "create_ticket",
		Description: "创建工单",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Propose: func(_ context.Context, value ToolProposal) error {
			proposal = value
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	var events []types.AGUIEvent
	err = runner.Run(context.Background(), toolRunInput(), func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("run tool stream: %v", err)
	}

	var eventTypes []string
	for _, event := range events {
		eventTypes = append(eventTypes, event.Type)
	}
	wantTypes := []string{
		"RUN_STARTED",
		"TOOL_CALL_START",
		"TOOL_CALL_ARGS",
		"TOOL_CALL_ARGS",
		"TOOL_CALL_END",
		"RUN_FINISHED",
	}
	if !reflect.DeepEqual(eventTypes, wantTypes) {
		t.Fatalf("expected events %v, got %v", wantTypes, eventTypes)
	}
	if proposal.ToolCallID != "call-1" || proposal.Name != "create_ticket" {
		t.Fatalf("unexpected proposal: %+v", proposal)
	}
	if string(proposal.Arguments) != `{"title":"网络故障","description":"无法访问"}` {
		t.Fatalf("unexpected tool arguments: %s", proposal.Arguments)
	}
}

func TestOpenAIRunnerDoesNotProposeIncompleteToolStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"function\":{\"name\":\"create_ticket\",\"arguments\":\"{\\\"title\\\":\\\"不完整\\\"}\"}}]}}]}\n\n")
	}))
	defer server.Close()

	proposed := false
	runner, err := NewOpenAIRunnerWithTools(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client(), []Tool{{
		Name:       "create_ticket",
		Parameters: json.RawMessage(`{"type":"object"}`),
		Propose: func(context.Context, ToolProposal) error {
			proposed = true
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	var events []types.AGUIEvent
	err = runner.Run(context.Background(), toolRunInput(), func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "before completion") {
		t.Fatalf("expected incomplete stream error, got %v", err)
	}
	if proposed {
		t.Fatal("an incomplete model stream must not create an approval")
	}
	for _, event := range events {
		if event.Type == "TOOL_CALL_END" || event.Type == "RUN_FINISHED" {
			t.Fatalf("incomplete tool stream emitted successful completion: %+v", events)
		}
	}
}

func TestOpenAIRunnerRejectsUnknownTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"function\":{\"name\":\"delete_everything\",\"arguments\":\"{}\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	proposed := false
	runner, err := NewOpenAIRunnerWithTools(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client(), []Tool{{
		Name:       "create_ticket",
		Parameters: json.RawMessage(`{"type":"object"}`),
		Propose: func(context.Context, ToolProposal) error {
			proposed = true
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	err = runner.Run(context.Background(), toolRunInput(), func(types.AGUIEvent) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "unsupported tool") {
		t.Fatalf("expected unsupported tool error, got %v", err)
	}
	if proposed {
		t.Fatal("an unknown tool must not reach the proposal handler")
	}
}

func toolRunInput() types.AGUIRunInput {
	return types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{{Role: "user", Content: "创建工单"}},
	}
}
