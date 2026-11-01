package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOpenAIEmbedderEmbedsInputsInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected embedding request path or authorization: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		if request.Model != "embedding-model" || request.Dimensions != 3 || !reflect.DeepEqual(request.Input, []string{"第一块", "第二块"}) {
			t.Errorf("unexpected embedding request: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
			{"index": 1, "embedding": []float32{0, 1, 0}},
			{"index": 0, "embedding": []float32{1, 0, 0}},
		}})
	}))
	defer server.Close()
	embedder, err := NewOpenAIEmbedder(OpenAIConfig{
		BaseURL:        server.URL,
		APIKey:         "test-key",
		EmbeddingModel: "embedding-model",
	}, server.Client(), 3)
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}
	vectors, err := embedder.Embed(context.Background(), []string{"第一块", "第二块"})
	if err != nil {
		t.Fatalf("embed inputs: %v", err)
	}
	if len(vectors) != 2 || !reflect.DeepEqual(vectors[0].Slice(), []float32{1, 0, 0}) ||
		!reflect.DeepEqual(vectors[1].Slice(), []float32{0, 1, 0}) {
		t.Fatalf("unexpected embedding order: %v", vectors)
	}
}

func TestOpenAIEmbedderRejectsProviderFailures(t *testing.T) {
	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "secret provider detail", http.StatusUnauthorized)
		}))
		defer server.Close()
		embedder, err := NewOpenAIEmbedder(OpenAIConfig{BaseURL: server.URL, EmbeddingModel: "embedding-model"}, server.Client(), 3)
		if err != nil {
			t.Fatalf("create embedder: %v", err)
		}
		if _, err := embedder.Embed(context.Background(), []string{"文本"}); err == nil {
			t.Fatal("expected an HTTP error")
		}
	})
	t.Run("wrong dimensions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1, 0}}}})
		}))
		defer server.Close()
		embedder, err := NewOpenAIEmbedder(OpenAIConfig{BaseURL: server.URL, EmbeddingModel: "embedding-model"}, server.Client(), 3)
		if err != nil {
			t.Fatalf("create embedder: %v", err)
		}
		if _, err := embedder.Embed(context.Background(), []string{"文本"}); err == nil {
			t.Fatal("expected a dimension validation error")
		}
	})
}
