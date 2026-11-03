package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestOpenAISupportJudgeReturnsVerdicts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request path: %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected Authorization header: %q", got)
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode support judge request: %v", err)
		}
		if request.Stream {
			t.Error("support judge must use a non-streaming request")
		}
		if request.Model != "test-model" {
			t.Errorf("unexpected model: %q", request.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "{\"supported\":[true,false]}"}},
			},
		})
	}))
	defer server.Close()

	judge, err := NewOpenAISupportJudge(OpenAIConfig{BaseURL: server.URL + "/v1", APIKey: "test-key", Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("create support judge: %v", err)
	}
	got, err := judge.Judge(context.Background(), []string{"句子一", "句子二"}, []string{"分块一", "分块二"})
	if err != nil {
		t.Fatalf("judge sentences: %v", err)
	}
	if !reflect.DeepEqual(got, []bool{true, false}) {
		t.Fatalf("unexpected verdicts: %v", got)
	}
}

func TestOpenAISupportJudgeRejectsMismatchedVerdicts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "{\"supported\":[true]}"}},
			},
		})
	}))
	defer server.Close()

	judge, err := NewOpenAISupportJudge(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("create support judge: %v", err)
	}
	if _, err := judge.Judge(context.Background(), []string{"一", "二"}, []string{"分块"}); err == nil {
		t.Fatal("expected a verdict count mismatch error")
	}
}

func TestOpenAISupportJudgeRejectsNonJSONContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "抱歉，我无法判断。"}},
			},
		})
	}))
	defer server.Close()

	judge, err := NewOpenAISupportJudge(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("create support judge: %v", err)
	}
	if _, err := judge.Judge(context.Background(), []string{"句子"}, []string{"分块"}); err == nil {
		t.Fatal("expected a non-JSON content error")
	}
}

func TestOpenAISupportJudgeSurfacesProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	judge, err := NewOpenAISupportJudge(OpenAIConfig{BaseURL: server.URL, Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("create support judge: %v", err)
	}
	if _, err := judge.Judge(context.Background(), []string{"句子"}, []string{"分块"}); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected an HTTP 429 error, got %v", err)
	}
}

func TestNewOpenAISupportJudgeRequiresModel(t *testing.T) {
	if _, err := NewOpenAISupportJudge(OpenAIConfig{}, nil); err == nil {
		t.Fatal("expected a missing model error")
	}
}
