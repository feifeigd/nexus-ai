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
	if ctx.Config != &c || ctx.Approvals == nil || ctx.Knowledge != nil {
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
