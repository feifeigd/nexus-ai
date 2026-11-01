package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"

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
