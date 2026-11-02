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
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" ||
			!strings.Contains(request.Messages[0].Content, "最新一条用户消息") ||
			!strings.Contains(request.Messages[0].Content, "不得因为历史消息") ||
			request.Messages[1].Role != "user" || request.Messages[1].Content != "创建工单" {
			t.Fatalf("model did not receive the latest-user tool instruction: %+v", request.Messages)
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
		"TOOL_CALL_RESULT",
		"RUN_FINISHED",
	}
	if !reflect.DeepEqual(eventTypes, wantTypes) {
		t.Fatalf("expected events %v, got %v", wantTypes, eventTypes)
	}
	result := events[len(events)-2]
	if result.MessageID != "call-1-result" || result.ToolCallID != "call-1" ||
		!strings.Contains(result.Content, "已提交人工审批") || !strings.Contains(result.Content, "尚未创建工单") {
		t.Fatalf("unexpected tool result: %+v", result)
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

func TestOpenAIRunnerExecutesReadOnlyToolAndStreamsFollowUp(t *testing.T) {
	const toolResult = `{"matches":[{"documentId":"document-1","content":"蓝雀告警需要先检查 API 服务。"}]}`
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		switch requestCount {
		case 1:
			if len(request.Tools) != 1 || request.Tools[0].Function.Name != "search_knowledge" {
				t.Errorf("model did not receive the read-only tool: %+v", request.Tools)
			}
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-search\",\"type\":\"function\",\"function\":{\"name\":\"search_knowledge\",\"arguments\":\"{\\\"query\\\":\\\"蓝雀告警\\\"}\"}}]}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
		case 2:
			if len(request.Tools) != 0 {
				t.Errorf("follow-up must not allow another tool call: %+v", request.Tools)
			}
			if len(request.Messages) != 4 || request.Messages[2].Role != "assistant" ||
				len(request.Messages[2].ToolCalls) != 1 || request.Messages[2].ToolCalls[0].ID != "call-search" ||
				request.Messages[3].Role != "tool" || request.Messages[3].ToolCallID != "call-search" ||
				request.Messages[3].Content != toolResult {
				t.Errorf("tool result was not returned to the model: %+v", request.Messages)
			}
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"根据文档，\"}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"先检查 API 服务。\"}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		default:
			t.Errorf("unexpected model request %d", requestCount)
		}
	}))
	defer server.Close()

	var proposal ToolProposal
	runner, err := NewOpenAIRunnerWithTools(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client(), []Tool{{
		Name:        "search_knowledge",
		Description: "检索知识库",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Execute: func(_ context.Context, value ToolProposal) (string, error) {
			proposal = value
			return toolResult, nil
		},
	}})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	var events []types.AGUIEvent
	if err := runner.Run(context.Background(), toolRunInput(), func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("run read-only tool stream: %v", err)
	}
	if requestCount != 2 || proposal.Name != "search_knowledge" || string(proposal.Arguments) != `{"query":"蓝雀告警"}` {
		t.Fatalf("unexpected read-only execution: requests=%d proposal=%+v", requestCount, proposal)
	}
	eventTypes := make([]string, 0, len(events))
	var answer strings.Builder
	for _, event := range events {
		eventTypes = append(eventTypes, event.Type)
		if event.Type == "TEXT_MESSAGE_CONTENT" {
			answer.WriteString(event.Delta)
		}
	}
	wantTypes := []string{
		"RUN_STARTED",
		"TOOL_CALL_START",
		"TOOL_CALL_ARGS",
		"TOOL_CALL_END",
		"TOOL_CALL_RESULT",
		"TEXT_MESSAGE_START",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_END",
		"RUN_FINISHED",
	}
	if !reflect.DeepEqual(eventTypes, wantTypes) {
		t.Fatalf("expected events %v, got %v", wantTypes, eventTypes)
	}
	if answer.String() != "根据文档，先检查 API 服务。" {
		t.Fatalf("unexpected follow-up answer: %q", answer.String())
	}
}

func TestOpenAIRunnerAppendsAnswerValidationNote(t *testing.T) {
	const toolResult = `{"matches":[{"documentId":"document-1","content":"先检查 API 服务。"}]}`
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		switch requestCount {
		case 1:
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-search\",\"type\":\"function\",\"function\":{\"name\":\"search_knowledge\",\"arguments\":\"{\\\"query\\\":\\\"蓝雀告警\\\"}\"}}]}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
		case 2:
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"根据文档，先检查 API 服务。\"}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		default:
			t.Errorf("unexpected model request %d", requestCount)
		}
	}))
	defer server.Close()

	runner, err := NewOpenAIRunnerWithTools(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client(), []Tool{{
		Name:        "search_knowledge",
		Description: "检索知识库",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Execute: func(context.Context, ToolProposal) (string, error) {
			return toolResult, nil
		},
		ValidateAnswer: func(_ context.Context, answer, _ string) (string, error) {
			if strings.Contains(answer, "先检查") {
				return "回答中以下来源不在本次检索结果内，请谨慎核实：document_deadbeef", nil
			}
			return "", nil
		},
	}})
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	var events []types.AGUIEvent
	if err := runner.Run(context.Background(), toolRunInput(), func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("run read-only tool stream: %v", err)
	}

	var answer strings.Builder
	contentCount := 0
	for _, event := range events {
		if event.Type == "TEXT_MESSAGE_CONTENT" {
			contentCount++
			answer.WriteString(event.Delta)
		}
	}
	if contentCount != 2 {
		t.Fatalf("expected model answer plus validation note, got %d content deltas", contentCount)
	}
	if !strings.Contains(answer.String(), "先检查 API 服务。") {
		t.Fatalf("missing model answer: %q", answer.String())
	}
	if !strings.Contains(answer.String(), "document_deadbeef") {
		t.Fatalf("expected validation note appended to answer, got %q", answer.String())
	}
}

func TestOpenAIRunnerRejectsAnswerValidationWithoutExecute(t *testing.T) {
	_, err := NewOpenAIRunnerWithTools(OpenAIConfig{Model: "test-model"}, nil, []Tool{{
		Name:       "invalid_tool",
		Parameters: json.RawMessage(`{"type":"object"}`),
		Propose: func(context.Context, ToolProposal) error {
			return nil
		},
		ValidateAnswer: func(context.Context, string, string) (string, error) {
			return "", nil
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "execute handler") {
		t.Fatalf("expected execute-handler validation, got %v", err)
	}
}

func TestOpenAIRunnerRejectsToolWithTwoHandlers(t *testing.T) {
	_, err := NewOpenAIRunnerWithTools(OpenAIConfig{Model: "test-model"}, nil, []Tool{{
		Name:       "invalid_tool",
		Parameters: json.RawMessage(`{"type":"object"}`),
		Execute: func(context.Context, ToolProposal) (string, error) {
			return "", nil
		},
		Propose: func(context.Context, ToolProposal) error {
			return nil
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected exactly-one handler validation, got %v", err)
	}
}

func toolRunInput() types.AGUIRunInput {
	return types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{{Role: "user", Content: "创建工单"}},
	}
}
