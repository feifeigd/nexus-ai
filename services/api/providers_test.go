package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/types"

	"entgo.io/ent/dialect"
)

func TestWireContextBuildsSQLiteServices(t *testing.T) {
	setWireTestEnv(t)
	c := wireTestConfig("wire_context_sqlite")
	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		t.Fatalf("wire sqlite context: %v", err)
	}
	defer cleanup()
	if ctx.Config != &c || ctx.Approvals == nil || ctx.Knowledge != nil || ctx.Documents == nil {
		t.Fatalf("unexpected wired context: %+v", ctx)
	}
	if _, ok := ctx.Runner.(agent.MockRunner); !ok {
		t.Fatalf("expected mock runner, got %T", ctx.Runner)
	}
}

func TestWireContextBuildsOptionalKnowledgeService(t *testing.T) {
	setWireTestEnv(t)
	t.Setenv("NEXUS_OPENAI_EMBEDDING_MODEL", "embedding-model")
	c := wireTestConfig("wire_context_knowledge")
	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		t.Fatalf("wire knowledge context: %v", err)
	}
	defer cleanup()
	if ctx.Knowledge == nil {
		t.Fatal("expected knowledge service when an embedding model is configured")
	}
}

func TestWireContextRegistersKnowledgeTool(t *testing.T) {
	var toolNames []string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode model request: %v", err)
			return
		}
		toolNames = nil
		for _, tool := range request.Tools {
			toolNames = append(toolNames, tool.Function.Name)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer provider.Close()
	setWireTestEnv(t)
	t.Setenv("NEXUS_AGENT_MODE", "openai")
	t.Setenv("NEXUS_OPENAI_BASE_URL", provider.URL)
	t.Setenv("NEXUS_OPENAI_MODEL", "test-model")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_MODEL", "embedding-model")
	c := wireTestConfig("wire_context_knowledge_tool")
	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		t.Fatalf("wire knowledge tool context: %v", err)
	}
	defer cleanup()
	if err := ctx.Runner.Run(context.Background(), types.AGUIRunInput{
		ThreadID: "thread-1",
		RunID:    "run-1",
		Messages: []types.AGUIMessage{{Role: "user", Content: "查询知识库"}},
	}, func(types.AGUIEvent) error { return nil }); err != nil {
		t.Fatalf("run wired agent: %v", err)
	}
	joined := strings.Join(toolNames, ",")
	if !strings.Contains(joined, "create_ticket") || !strings.Contains(joined, "search_knowledge") {
		t.Fatalf("wired runner is missing server tools: %v", toolNames)
	}
}

func TestWireContextUsesIndependentChatAndEmbeddingProviders(t *testing.T) {
	chatCalled := false
	chatProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCalled = true
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer chat-key" {
			t.Errorf("unexpected chat request path or authorization: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode chat request: %v", err)
			return
		}
		if request.Model != "chat-model" {
			t.Errorf("unexpected chat model: %q", request.Model)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer chatProvider.Close()

	embeddingCalled := false
	embeddingProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		embeddingCalled = true
		if r.URL.Path != "/embeddings" || r.Header.Get("Authorization") != "Bearer embedding-key" {
			t.Errorf("unexpected embedding request path or authorization: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode embedding request: %v", err)
			return
		}
		if request.Model != "embedding-model" {
			t.Errorf("unexpected embedding model: %q", request.Model)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
			"index": 0, "embedding": make([]float32, entschema.EmbeddingDimensions),
		}}})
	}))
	defer embeddingProvider.Close()

	setWireTestEnv(t)
	t.Setenv("NEXUS_AGENT_MODE", "openai")
	t.Setenv("NEXUS_OPENAI_BASE_URL", chatProvider.URL)
	t.Setenv("NEXUS_OPENAI_API_KEY", "chat-key")
	t.Setenv("NEXUS_OPENAI_MODEL", "chat-model")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_BASE_URL", embeddingProvider.URL)
	t.Setenv("NEXUS_OPENAI_EMBEDDING_API_KEY", "embedding-key")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_MODEL", "embedding-model")
	c := wireTestConfig("wire_context_independent_providers")
	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		t.Fatalf("wire independent providers: %v", err)
	}
	defer cleanup()
	if ctx.Knowledge == nil {
		t.Fatal("expected knowledge service for the independent embedding provider")
	}
	if _, err := ctx.Knowledge.IngestDocument(context.Background(), "运维手册", "故障排查"); err != nil {
		t.Fatalf("ingest through independent embedding provider: %v", err)
	}
	if err := ctx.Runner.Run(context.Background(), types.AGUIRunInput{
		ThreadID: "thread-independent",
		RunID:    "run-independent",
		Messages: []types.AGUIMessage{{Role: "user", Content: "你好"}},
	}, func(types.AGUIEvent) error { return nil }); err != nil {
		t.Fatalf("run independent chat provider: %v", err)
	}
	if !chatCalled || !embeddingCalled {
		t.Fatalf("expected both providers to be called: chat=%t embedding=%t", chatCalled, embeddingCalled)
	}
}

func TestDriverNameUsesPostgresEnvironmentOverride(t *testing.T) {
	setWireTestEnv(t)
	t.Setenv("NEXUS_POSTGRES_DSN", "postgres://nexus:test@127.0.0.1:5432/nexus_test?sslmode=disable")
	c := wireTestConfig("unused")
	if got := data.DriverName(&c.Data); got != dialect.Postgres {
		t.Fatalf("expected effective postgres driver, got %q", got)
	}
}

func TestWireContextPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("NEXUS_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set NEXUS_TEST_POSTGRES_DSN to a disposable PostgreSQL database or schema")
	}
	setWireTestEnv(t)
	t.Setenv("NEXUS_POSTGRES_DSN", dsn)
	c := wireTestConfig("unused")
	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		t.Fatalf("wire postgres context: %v", err)
	}
	defer cleanup()
	if _, err := ctx.Approvals.List(context.Background()); err != nil {
		t.Fatalf("query approvals through wired postgres store: %v", err)
	}
}

func TestWireContextRejectsInvalidRunner(t *testing.T) {
	setWireTestEnv(t)
	t.Setenv("NEXUS_AGENT_MODE", "unknown")
	c := wireTestConfig("wire_context_invalid_runner")
	ctx, cleanup, err := wireContext(&c)
	if err == nil || ctx != nil || cleanup != nil {
		t.Fatalf("expected runner construction failure, ctx=%v cleanupSet=%t err=%v", ctx, cleanup != nil, err)
	}
}

func setWireTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("NEXUS_POSTGRES_DSN", "")
	t.Setenv("NEXUS_AGENT_MODE", "mock")
	t.Setenv("NEXUS_OPENAI_BASE_URL", "")
	t.Setenv("NEXUS_OPENAI_API_KEY", "")
	t.Setenv("NEXUS_OPENAI_MODEL", "")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_BASE_URL", "")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_API_KEY", "")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_MODEL", "")
}

func wireTestConfig(name string) config.Config {
	return config.Config{
		Agent: agent.Config{Mode: "mock"},
		Data: data.Data{Database: data.Database{
			Driver:       dialect.SQLite,
			Source:       "file:" + name + "?mode=memory&cache=shared&_fk=1&_txlock=immediate",
			MaxOpenConns: 1,
			MaxIdleConns: 1,
			AutoMigrate:  true,
		}},
	}
}
