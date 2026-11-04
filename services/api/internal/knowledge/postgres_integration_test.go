package knowledge

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/testpg"
)

// TestSearchPipelineAgainstPostgres verifies the full search_knowledge pipeline
// against a real PostgreSQL + pgvector database: HNSW cosine recall, distance
// threshold filtering, keyword re-ranking, and answer validation. It is skipped
// unless NEXUS_TEST_POSTGRES_DSN points at a disposable database.
func TestSearchPipelineAgainstPostgres(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("NEXUS_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		t.Skip("set NEXUS_TEST_POSTGRES_DSN to a disposable PostgreSQL database or schema")
	}
	dsn, dropDB, err := testpg.NewDatabase(baseDSN)
	if err != nil {
		t.Fatalf("create disposable postgres database: %v", err)
	}
	defer dropDB()

	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:      dialect.Postgres,
		Source:      dsn,
		AutoMigrate: true,
	}})
	if err != nil {
		t.Fatalf("create postgres data source: %v", err)
	}
	defer closeData()

	ctx := context.Background()
	store := data.NewKnowledgeStore(client, dialect.Postgres)

	// Deterministic vectors control the cosine geometry without a real embedding
	// provider. The query embedding points at (1, 0).
	queryVector := knowledgePostgresVector(1, 0)
	service, err := NewService(embedderFunc(func(context.Context, []string) ([]pgvector.Vector, error) {
		return []pgvector.Vector{queryVector}, nil
	}), store, DefaultChunkSize, DefaultChunkOverlap)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}

	keywordContent := "蓝雀告警需要先检查 API 服务状态"
	nearContent := "数据库迁移需要停机维护窗口"
	farContent := "容器编排采用声明式配置"

	documentID, err := store.CreateDocument(ctx, "集成检索测试", "完整文档", "tenant-a", []data.KnowledgeChunkInput{
		{Content: nearContent, Embedding: knowledgePostgresVector(1, 0)},        // 余弦距离 0，无关键词重叠
		{Content: keywordContent, Embedding: knowledgePostgresVector(0.9, 0.1)}, // 距离约 0.006，关键词重叠
		{Content: farContent, Embedding: knowledgePostgresVector(0, 1)},         // 距离 1，应被阈值过滤
	})
	if err != nil {
		t.Fatalf("ingest postgres document: %v", err)
	}
	defer func() {
		if err := client.Document.DeleteOneID(documentID).Exec(ctx); err != nil {
			t.Errorf("clean up postgres document: %v", err)
		}
	}()

	matches, err := service.Search(ctx, "蓝雀告警", DefaultSearchLimit, "tenant-a")
	if err != nil {
		t.Fatalf("search knowledge pipeline: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected threshold to filter the far chunk, got %d matches: %+v", len(matches), matches)
	}
	// 关键词命中的分块余弦距离略差，但应被重排序提到更近分块之前。
	if matches[0].Content != keywordContent || matches[1].Content != nearContent {
		t.Fatalf("unexpected re-ranked order: %+v", matches)
	}

	tool := service.Tool()
	toolResult, err := tool.Execute(ctx, agent.ToolProposal{
		Arguments: json.RawMessage(`{"query":"蓝雀告警","limit":5}`),
	})
	if err != nil {
		t.Fatalf("execute search_knowledge tool: %v", err)
	}
	var result searchKnowledgeResult
	if err := json.Unmarshal([]byte(toolResult), &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if len(result.Matches) != 2 {
		t.Fatalf("tool result should carry the two surviving matches: %+v", result.Matches)
	}

	fabricated := "document_" + strings.Repeat("f", 32)
	answer := "根据 " + fabricated + "，缓存策略采用淘汰算法。"
	note, err := tool.ValidateAnswer(ctx, answer, toolResult)
	if err != nil {
		t.Fatalf("validate answer: %v", err)
	}
	if !strings.Contains(note, fabricated) {
		t.Fatalf("expected fabricated citation to be reported, got %q", note)
	}
	if !strings.Contains(note, "缓存策略") {
		t.Fatalf("expected unsupported sentence to be reported, got %q", note)
	}
}

func knowledgePostgresVector(first, second float32) pgvector.Vector {
	values := make([]float32, entschema.EmbeddingDimensions)
	values[0] = first
	values[1] = second
	return pgvector.NewVector(values)
}
