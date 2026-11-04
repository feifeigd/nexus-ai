package main

import (
	"fmt"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	dataent "nexusai/services/api/internal/data/ent"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/knowledge"
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

func newTokenUsageStore(client *dataent.Client) *data.TokenUsageStore {
	return data.NewTokenUsageStore(client)
}

func newUserStore(client *dataent.Client) *data.UserStore {
	return data.NewUserStore(client)
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
