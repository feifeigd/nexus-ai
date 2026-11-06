package main

import (
	"fmt"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	dataent "nexusai/services/api/internal/data/ent"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/knowledge"
	"nexusai/services/api/internal/sshx"
	"nexusai/services/api/internal/svc"
)

func provideDataConfig(c *config.Config) *data.Data {
	return &c.Data
}

func provideAgentConfig(c *config.Config) agent.Config {
	return agent.ConfigFromEnv(c.Agent)
}

func newRunner(c agent.Config, approvals *approval.EntStore, knowledgeService svc.KnowledgeService) (agent.Runner, error) {
	tools := []agent.Tool{approvals.Tool()}
	if provider, ok := knowledgeService.(interface{ Tool() agent.Tool }); ok {
		tools = append(tools, provider.Tool())
	}
	runner, err := agent.NewRunner(c, tools...)
	if err != nil {
		return nil, fmt.Errorf("configure agent: %w", err)
	}
	return runner, nil
}

func newKnowledgeStore(client *dataent.Client, driver string) *data.KnowledgeStore {
	return data.NewKnowledgeStore(client, driver)
}

// newSSHRunner 用 SSH 配置构造远程命令执行器。Timeout 为 0 时由 sshx 使用默认 30s。
func newSSHRunner(c *config.Config) sshx.Runner {
	return sshx.NewClient(sshx.Config{
		User:    c.SSH.User,
		Port:    c.SSH.Port,
		Timeout: time.Duration(c.SSH.Timeout) * time.Second,
	})
}

func newKnowledgeService(c agent.Config, store *data.KnowledgeStore) (svc.KnowledgeService, error) {
	if c.Embedding.Model == "" {
		return nil, nil
	}
	embedder, err := agent.NewOpenAIEmbedder(c.Embedding, nil, entschema.EmbeddingDimensions)
	if err != nil {
		return nil, fmt.Errorf("configure knowledge embedder: %w", err)
	}
	service, err := knowledge.NewService(embedder, store, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("configure knowledge service: %w", err)
	}
	if c.SemanticFactCheck {
		judge, err := agent.NewOpenAISupportJudge(c.OpenAI, nil)
		if err != nil {
			return nil, fmt.Errorf("configure knowledge support judge: %w", err)
		}
		service.SetSupportJudge(judge)
	}
	return service, nil
}
