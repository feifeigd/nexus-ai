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

func TestOpenAIRunnerStreamsChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request path: %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected Authorization header: %q", got)
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode model request: %v", err)
		}
		if request.Model != "test-model" || !request.Stream {
			t.Errorf("unexpected model request: %+v", request)
		}
		wantMessages := []chatMessage{
			{Role: "system", Content: "保持简洁"},
			{Role: "user", Content: "旧问题"},
			{Role: "assistant", Content: "旧回答"},
			{Role: "user", Content: "你好"},
		}
		if !reflect.DeepEqual(request.Messages, wantMessages) {
			t.Errorf("expected messages %+v, got %+v", wantMessages, request.Messages)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"，NexusAI\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	runner, err := NewOpenAIRunner(OpenAIConfig{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	}, server.Client())
	if err != nil {
		t.Fatalf("create OpenAI runner: %v", err)
	}
	input := types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{
			{Role: "system", Content: "保持简洁"},
			{Role: "user", Content: "旧问题"},
			{Role: "assistant", Content: "旧回答"},
			{Role: "user", Content: "你好"},
		},
	}
	var events []types.AGUIEvent
	err = runner.Run(context.Background(), input, func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("run OpenAI stream: %v", err)
	}

	var gotTypes []string
	var text strings.Builder
	for _, event := range events {
		gotTypes = append(gotTypes, event.Type)
		text.WriteString(event.Delta)
	}
	wantTypes := []string{
		"RUN_STARTED",
		"TEXT_MESSAGE_START",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_END",
		"RUN_FINISHED",
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Errorf("expected events %v, got %v", wantTypes, gotTypes)
	}
	if got := text.String(); got != "你好，NexusAI" {
		t.Errorf("unexpected reply text: %q", got)
	}
}

func TestOpenAIRunnerRejectsProviderErrorBeforeEmitting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	runner, err := NewOpenAIRunner(OpenAIConfig{
		BaseURL: server.URL,
		Model:   "test-model",
	}, server.Client())
	if err != nil {
		t.Fatalf("create OpenAI runner: %v", err)
	}
	input := types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{{Role: "user", Content: "你好"}},
	}
	emitCalled := false
	err = runner.Run(context.Background(), input, func(types.AGUIEvent) error {
		emitCalled = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected an HTTP 401 error, got %v", err)
	}
	if emitCalled {
		t.Fatal("provider errors must be returned before starting AG-UI events")
	}
}

func TestOpenAIRunnerRejectsIncompleteStream(t *testing.T) {
	tests := map[string]struct {
		contentType string
		body        string
	}{
		"truncated SSE": {
			contentType: "text/event-stream",
			body:        "data: {\"choices\":[{\"delta\":{\"content\":\"不完整\"}}]}\n\n",
		},
		"non-SSE response": {
			contentType: "application/json",
			body:        `{"choices":[]}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()

			runner, err := NewOpenAIRunner(OpenAIConfig{
				BaseURL: server.URL,
				Model:   "test-model",
			}, server.Client())
			if err != nil {
				t.Fatalf("create OpenAI runner: %v", err)
			}
			input := types.AGUIRunInput{
				ThreadID: "thread-1",
				RunID:    "run-1",
				Messages: []types.AGUIMessage{{Role: "user", Content: "你好"}},
			}
			var events []types.AGUIEvent
			err = runner.Run(context.Background(), input, func(event types.AGUIEvent) error {
				events = append(events, event)
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), "before completion") {
				t.Fatalf("expected an incomplete stream error, got %v", err)
			}
			for _, event := range events {
				if event.Type == "TEXT_MESSAGE_END" || event.Type == "RUN_FINISHED" {
					t.Fatalf("incomplete stream must not emit successful completion events: %+v", events)
				}
			}
		})
	}
}

func TestOpenAIRunnerAcceptsFinishReasonWithoutDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"完整回答\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	}))
	defer server.Close()

	runner, err := NewOpenAIRunner(OpenAIConfig{
		BaseURL: server.URL,
		Model:   "test-model",
	}, server.Client())
	if err != nil {
		t.Fatalf("create OpenAI runner: %v", err)
	}
	input := types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{{Role: "user", Content: "你好"}},
	}
	var events []types.AGUIEvent
	err = runner.Run(context.Background(), input, func(event types.AGUIEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("run stream with finish reason: %v", err)
	}
	wantTypes := []string{
		"RUN_STARTED",
		"TEXT_MESSAGE_START",
		"TEXT_MESSAGE_CONTENT",
		"TEXT_MESSAGE_END",
		"RUN_FINISHED",
	}
	var gotTypes []string
	for _, event := range events {
		gotTypes = append(gotTypes, event.Type)
	}
	if !reflect.DeepEqual(gotTypes, wantTypes) {
		t.Fatalf("expected events %v, got %v", wantTypes, gotTypes)
	}
	if got := events[2].Delta; got != "完整回答" {
		t.Fatalf("unexpected reply text: %q", got)
	}
}

func TestNewOpenAIRunnerRequiresModel(t *testing.T) {
	if _, err := NewOpenAIRunner(OpenAIConfig{}, nil); err == nil {
		t.Fatal("expected a missing model error")
	}
}

func TestConvertMessagesAcceptsAssistantWithoutContent(t *testing.T) {
	messages, err := convertMessages([]types.AGUIMessage{
		{Role: "assistant", Content: nil},
		{Role: "user", Content: "继续"},
	})
	if err != nil {
		t.Fatalf("assistant tool-call history must be accepted: %v", err)
	}
	want := []chatMessage{
		{Role: "assistant", Content: ""},
		{Role: "user", Content: "继续"},
	}
	if !reflect.DeepEqual(messages, want) {
		t.Fatalf("expected messages %+v, got %+v", want, messages)
	}
}

func TestConvertMessagesRejectsNonTextUserContent(t *testing.T) {
	if _, err := convertMessages([]types.AGUIMessage{
		{Role: "user", Content: []any{"not", "text"}},
	}); err == nil {
		t.Fatal("non-text user content must still be rejected")
	}
}
