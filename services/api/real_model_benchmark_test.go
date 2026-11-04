package main

import (
	"context"
	"fmt"
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
	"nexusai/services/api/internal/types"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

// TestRealModelRetrievalBenchmark 量化知识库检索的质量与延迟：用一批覆盖多文档的
// 测试问题，统计命中率、各查询的余弦距离与检索延迟。只在 NEXUS_TEST_REAL_MODEL=1
// 时运行，复用 NEXUS_POSTGRES_DSN 指向的现有库（含已导入文档），不创建临时库。
func TestRealModelRetrievalBenchmark(t *testing.T) {
	if strings.TrimSpace(os.Getenv("NEXUS_TEST_REAL_MODEL")) != "1" {
		t.Skip("set NEXUS_TEST_REAL_MODEL=1 to run the real-model retrieval benchmark")
	}
	if err := godotenv.Load(); err != nil {
		t.Fatalf("load .env: %v", err)
	}
	var c config.Config
	if err := conf.Load("etc/nexus-api.yaml", &c); err != nil {
		t.Fatalf("load config: %v", err)
	}
	agentCfg := agent.ConfigFromEnv(c.Agent)
	if agentCfg.Embedding.Model == "" {
		t.Skip("no embedding model configured")
	}

	dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set NEXUS_POSTGRES_DSN to the PostgreSQL + pgvector instance")
	}
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

	// title -> documentId 映射，用于程序化判定命中。
	docs, err := store.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	titleToID := make(map[string]string, len(docs))
	for _, d := range docs {
		titleToID[d.Title] = d.DocumentID
	}

	queries := []struct {
		query     string
		wantTitle string
	}{
		{"蓝雀缓存雪崩如何应对？", "蓝雀缓存雪崩与击穿应对"},
		{"蓝雀缓存 maxmemory 建议设置多少？", "蓝雀缓存配置与运维"},
		{"蓝雀缓存用什么淘汰策略？", "蓝雀缓存服务架构"},
		{"什么是缓存击穿？", "蓝雀缓存雪崩与击穿应对"},
		{"蓝雀缓存支持哪些数据结构？", "蓝雀缓存服务架构"},
		{"缓存穿透如何解决？", "蓝雀缓存雪崩与击穿应对"},
		{"蓝雀缓存主从复制是同步还是异步？", "蓝雀缓存服务架构"},
		{"蓝雀缓存命中率低于多少需要排查？", "蓝雀缓存配置与运维"},
		{"星帆平台紫灯告警第一步做什么？", "星帆运维平台应急手册"},
		{"蓝雀缓存支持哪些持久化方式？", "蓝雀缓存配置与运维"},
	}

	t.Logf("知识库文档 %d 篇", len(docs))
	var total time.Duration
	hits, valid := 0, 0
	for i, q := range queries {
		wantID, ok := titleToID[q.wantTitle]
		if !ok {
			t.Logf("[%d] %q 期望文档未在库中：%q", i+1, q.query, q.wantTitle)
			continue
		}
		valid++
		start := time.Now()
		matches, err := service.Search(ctx, q.query, knowledge.DefaultSearchLimit)
		elapsed := time.Since(start)
		if err != nil {
			t.Logf("[%d] %q 检索出错：%v", i+1, q.query, err)
			continue
		}
		total += elapsed
		hit, rank := false, 0
		for j, m := range matches {
			if m.DocumentID == wantID {
				hit, rank = true, j+1
				break
			}
		}
		status := "MISS"
		if hit {
			hits++
			status = fmt.Sprintf("HIT@%d", rank)
		}
		topDist := "无结果"
		if len(matches) > 0 {
			topDist = fmt.Sprintf("top1=%.4f", matches[0].Distance)
		}
		t.Logf("[%d] %-22s -> %-6s want=%-16s %s 命中%d条 %s",
			i+1, q.query, status, q.wantTitle, topDist, len(matches), elapsed.Round(time.Millisecond))
	}
	if valid > 0 {
		t.Logf("汇总：命中率 %d/%d = %.0f%%，平均检索延迟 %s",
			hits, valid, 100*float64(hits)/float64(valid),
			(total / time.Duration(valid)).Round(time.Millisecond))
	}
}

// TestRealModelE2EBenchmark 量化端到端问答的延迟与工具调用：走完整两段式流程
// （第一次模型调用决定检索 → 工具执行 → 第二次模型调用生成回答 → 语义核验），
// 记录端到端延迟、工具调用次数与回答片段。只用少量代表性问题以控制真实模型成本。
func TestRealModelE2EBenchmark(t *testing.T) {
	if strings.TrimSpace(os.Getenv("NEXUS_TEST_REAL_MODEL")) != "1" {
		t.Skip("set NEXUS_TEST_REAL_MODEL=1 to run the real-model e2e benchmark")
	}
	if err := godotenv.Load(); err != nil {
		t.Fatalf("load .env: %v", err)
	}
	var c config.Config
	if err := conf.Load("etc/nexus-api.yaml", &c); err != nil {
		t.Fatalf("load config: %v", err)
	}
	agentCfg := agent.ConfigFromEnv(c.Agent)
	if agentCfg.Embedding.Model == "" {
		t.Skip("no embedding model configured")
	}
	if agentCfg.OpenAI.Model == "" {
		t.Skip("no chat model configured")
	}

	dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set NEXUS_POSTGRES_DSN to the PostgreSQL + pgvector instance")
	}
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
	var judge *agent.OpenAISupportJudge
	if agentCfg.SemanticFactCheck {
		judge, err = agent.NewOpenAISupportJudge(agentCfg.OpenAI, nil)
		if err != nil {
			t.Fatalf("configure real support judge: %v", err)
		}
		service.SetSupportJudge(judge)
	}

	runner, err := agent.NewOpenAIRunnerWithTools(agentCfg.OpenAI, nil, []agent.Tool{service.Tool()})
	if err != nil {
		t.Fatalf("configure openai runner: %v", err)
	}

	queries := []string{
		"蓝雀缓存雪崩如何应对？",
		"蓝雀缓存 maxmemory 建议设置多少？",
		"蓝雀缓存用什么淘汰策略？",
		"什么是缓存击穿？",
		"星帆平台紫灯告警第一步做什么？",
	}

	var total time.Duration
	ran := 0
	for i, q := range queries {
		input := types.AGUIRunInput{
			ThreadID: "bench-e2e-thread",
			RunID:    fmt.Sprintf("bench-e2e-run-%d", i),
			Messages: []types.AGUIMessage{{Role: "user", Content: q}},
		}
		var answer strings.Builder
		toolCalls := 0
		start := time.Now()
		err := runner.Run(ctx, input, func(ev types.AGUIEvent) error {
			switch ev.Type {
			case "TEXT_MESSAGE_CONTENT":
				answer.WriteString(ev.Delta)
			case "TOOL_CALL_START":
				toolCalls++
			}
			return nil
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Logf("[%d] %-22s 出错：%v", i+1, q, err)
			continue
		}
		total += elapsed
		ran++
		u := runner.Usage()
		var judgeTokens int
		if judge != nil {
			judgeTokens = judge.Usage().TotalTokens
		}
		ans := strings.Join(strings.Fields(answer.String()), " ")
		if len(ans) > 60 {
			ans = ans[:60] + "…"
		}
		t.Logf("[%d] %-22s 端到端 %-8s 工具%d次 token(对话%d/核验%d) 回答:%s",
			i+1, q, elapsed.Round(time.Millisecond), toolCalls, u.TotalTokens, judgeTokens, ans)
	}
	if ran > 0 {
		t.Logf("汇总：平均端到端延迟 %s（语义核验 %t）",
			(total / time.Duration(ran)).Round(time.Millisecond), agentCfg.SemanticFactCheck)
	}
}
