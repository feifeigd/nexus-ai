package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/types"
)

func TestAGUIHandlerStreamsMockRun(t *testing.T) {
	body := `{
		"threadId":"thread-1",
		"runId":"run-1",
		"messages":[
			{"id":"message-1","role":"user","content":"旧消息"},
			{"id":"message-2","role":"assistant","content":null},
			{"id":"message-3","role":"user","content":"你好，NexusAI"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/ag-ui", strings.NewReader(body))
	resp := httptest.NewRecorder()

	AGUIHandler(agent.MockRunner{}).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Errorf("unexpected Content-Type: %q", got)
	}
	if !resp.Flushed {
		t.Error("expected the response to be flushed")
	}

	events := decodeAGUIEvents(t, resp.Body.String())
	wantTypes := []string{
		"RUN_STARTED",
		"TEXT_MESSAGE_START",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_END",
		"RUN_FINISHED",
	}
	gotTypes := make([]string, 0, len(events))
	for _, event := range events {
		gotTypes = append(gotTypes, event.Type)
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Fatalf("expected event sequence %v, got %v", wantTypes, gotTypes)
	}
	if events[0].ThreadID != "thread-1" || events[0].RunID != "run-1" {
		t.Errorf("RUN_STARTED does not echo thread and run IDs: %+v", events[0])
	}
	if events[1].MessageID != "run-1-assistant" || events[1].Role != "assistant" {
		t.Errorf("unexpected TEXT_MESSAGE_START: %+v", events[1])
	}
	if got := events[2].Delta + events[3].Delta; got != "NexusAI Mock Agent 已收到：你好，NexusAI" {
		t.Errorf("unexpected streamed text: %q", got)
	}
	if events[5].ThreadID != "thread-1" || events[5].RunID != "run-1" {
		t.Errorf("RUN_FINISHED does not echo thread and run IDs: %+v", events[5])
	}
}

func TestAGUIHandlerRejectsInvalidInput(t *testing.T) {
	tests := map[string]string{
		"invalid JSON":  `{`,
		"missing IDs":   `{"threadId":"","runId":"","messages":[{"role":"user","content":"hello"}]}`,
		"missing user":  `{"threadId":"thread-1","runId":"run-1","messages":[{"role":"assistant","content":"hello"}]}`,
		"non-text user": `{"threadId":"thread-1","runId":"run-1","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/ag-ui", strings.NewReader(body))
			resp := httptest.NewRecorder()

			AGUIHandler(agent.MockRunner{}).ServeHTTP(resp, req)

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
			}
			if strings.Contains(resp.Header().Get("Content-Type"), "text/event-stream") {
				t.Error("invalid input must be rejected before starting the event stream")
			}
		})
	}
}

type runnerFunc func(context.Context, types.AGUIRunInput, agent.EmitFunc) error

func (f runnerFunc) Run(ctx context.Context, input types.AGUIRunInput, emit agent.EmitFunc) error {
	return f(ctx, input, emit)
}

func TestAGUIHandlerReturnsBadGatewayBeforeStream(t *testing.T) {
	runner := runnerFunc(func(context.Context, types.AGUIRunInput, agent.EmitFunc) error {
		return errors.New("provider unavailable")
	})
	req := httptest.NewRequest(http.MethodPost, "/ag-ui", strings.NewReader(
		`{"threadId":"thread-1","runId":"run-1","messages":[{"role":"user","content":"hello"}]}`,
	))
	resp := httptest.NewRecorder()

	AGUIHandler(runner).ServeHTTP(resp, req)

	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.Code)
	}
	if strings.Contains(resp.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatal("provider failure must be rejected before starting SSE")
	}
}

func TestAGUIHandlerEmitsRunErrorAfterStreamStarts(t *testing.T) {
	runner := runnerFunc(func(_ context.Context, input types.AGUIRunInput, emit agent.EmitFunc) error {
		if err := emit(types.AGUIEvent{Type: "RUN_STARTED", ThreadID: input.ThreadID, RunID: input.RunID}); err != nil {
			return err
		}
		return errors.New("provider secret details")
	})
	req := httptest.NewRequest(http.MethodPost, "/ag-ui", strings.NewReader(
		`{"threadId":"thread-1","runId":"run-1","messages":[{"role":"user","content":"hello"}]}`,
	))
	resp := httptest.NewRecorder()

	AGUIHandler(runner).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	events := decodeAGUIEvents(t, resp.Body.String())
	if len(events) != 2 || events[1].Type != "RUN_ERROR" {
		t.Fatalf("expected RUN_STARTED followed by RUN_ERROR, got %+v", events)
	}
	if events[1].Message != "agent run failed" || strings.Contains(resp.Body.String(), "provider secret details") {
		t.Fatalf("unexpected public error event: %+v", events[1])
	}
}

func decodeAGUIEvents(t *testing.T, body string) []types.AGUIEvent {
	t.Helper()

	var events []types.AGUIEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			t.Fatalf("unexpected SSE line: %q", line)
		}
		var event types.AGUIEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatalf("decode SSE event: %v", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan SSE response: %v", err)
	}
	return events
}
