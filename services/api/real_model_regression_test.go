package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/knowledge"
	"nexusai/services/api/internal/testpg"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

// TestRealModelKnowledgePipeline 是第 29/30 步的真实供应商回归：用真实嵌入模型做
// 检索、用真实对话模型做语义级事实核验，并记录各阶段延迟与观察结果，供人工评估
// 检索质量、判定质量、额外延迟与成本。它只在显式设置 NEXUS_TEST_REAL_MODEL=1 时
// 运行，因为真实模型调用非确定且会产生费用；断言仅覆盖结构正确性（能成功入库、
// 能召回、核验不报错），检索/判定质量通过日志观察而非硬断言。
func TestRealModelKnowledgePipeline(t *testing.T) {
	if strings.TrimSpace(os.Getenv("NEXUS_TEST_REAL_MODEL")) != "1" {
		t.Skip("set NEXUS_TEST_REAL_MODEL=1 to run the real-model knowledge regression")
	}

	// 与 main() 一致：先加载 .env（密钥），再加载 yaml 默认值并叠加环境变量。
	if err := godotenv.Load(); err != nil {
		t.Fatalf("load .env: %v", err)
	}
	var c config.Config
	if err := conf.Load("etc/nexus-api.yaml", &c); err != nil {
		t.Fatalf("load config: %v", err)
	}
	agentCfg := agent.ConfigFromEnv(c.Agent)
	if agentCfg.Embedding.Model == "" {
		t.Skip("no embedding model configured; set NEXUS_OPENAI_EMBEDDING_MODEL or etc/nexus-api.yaml")
	}

	baseDSN := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN"))
	if baseDSN == "" {
		t.Skip("set NEXUS_POSTGRES_DSN to a PostgreSQL + pgvector instance")
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

	embedder, err := agent.NewOpenAIEmbedder(agentCfg.Embedding, nil, entschema.EmbeddingDimensions)
	if err != nil {
		t.Fatalf("configure real embedder: %v", err)
	}
	service, err := knowledge.NewService(embedder, store, knowledge.DefaultChunkSize, knowledge.DefaultChunkOverlap)
	if err != nil {
		t.Fatalf("configure knowledge service: %v", err)
	}
	if agentCfg.SemanticFactCheck {
		judge, err := agent.NewOpenAISupportJudge(agentCfg.OpenAI, nil)
		if err != nil {
			t.Fatalf("configure real support judge: %v", err)
		}
		service.SetSupportJudge(judge)
	}

	title := "蓝雀缓存服务故障排查"
	content := "蓝雀缓存服务使用 LRU 淘汰策略，内存不足时会优先淘汰最近最少使用的键。出现缓存雪崩时，应对热点键设置随机过期时间，并启用熔断与限流以保护下游数据库。"

	ingestStart := time.Now()
	documentID, err := service.IngestDocument(ctx, title, content, "tenant-a")
	if err != nil {
		t.Fatalf("ingest document through real embedding model: %v", err)
	}
	t.Logf("[ingest] document=%s latency=%s", documentID, time.Since(ingestStart).Round(time.Millisecond))

	// 第 30 步：真实嵌入模型下的检索（余弦召回 + 阈值过滤 + BM25 关键词打分融合）。
	query := "蓝雀缓存雪崩如何应对？"
	searchStart := time.Now()
	matches, err := service.Search(ctx, query, knowledge.DefaultSearchLimit, "tenant-a")
	if err != nil {
		t.Fatalf("search through real embedding model: %v", err)
	}
	t.Logf("[search] query=%q latency=%s matched=%d", query, time.Since(searchStart).Round(time.Millisecond), len(matches))
	for i, m := range matches {
		t.Logf("[search] #%d document=%s chunk=%s ordinal=%d distance=%.4f content=%.60q",
			i, m.DocumentID, m.ChunkID, m.Ordinal, m.Distance, m.Content)
	}
	if len(matches) == 0 {
		t.Fatalf("expected the topic-matched query to retrieve at least one chunk")
	}

	// 第 29 步：真实对话模型下的语义级事实核验。supported 句子来自已入库内容，
	// unsupported 句子与入库内容无关，观察判定器能否区分二者。
	tool := service.Tool()
	toolResult, err := tool.Execute(ctx, agent.ToolProposal{
		Arguments: json.RawMessage(`{"query":"蓝雀缓存雪崩如何应对？","limit":5}`),
	})
	if err != nil {
		t.Fatalf("execute search_knowledge through real model: %v", err)
	}

	supported := "缓存雪崩时应对热点键设置随机过期时间，并启用熔断与限流。"
	unsupported := "数据库主从复制采用半同步复制模式。"
	answer := supported + " " + unsupported

	validateStart := time.Now()
	note, err := tool.ValidateAnswer(ctx, answer, toolResult)
	if err != nil {
		t.Fatalf("validate answer through semantic judge: %v", err)
	}
	t.Logf("[validate] semantic=%t latency=%s note=%q",
		agentCfg.SemanticFactCheck, time.Since(validateStart).Round(time.Millisecond), note)
}
