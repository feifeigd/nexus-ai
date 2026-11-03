package agent

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
)

type Config struct {
	Mode      string `json:",default=mock"`
	OpenAI    OpenAIConfig
	Embedding OpenAIConfig
	// SemanticFactCheck 启用知识回答的语义级事实核验（NLI/LLM 判定），
	// 默认关闭以控制每次回答额外的模型调用成本与延迟。
	SemanticFactCheck bool `json:",optional"`
}

type OpenAIConfig struct {
	BaseURL string `json:",optional"`
	APIKey  string `json:",optional"`
	Model   string `json:",optional"`
}

func ConfigFromEnv(base Config) Config {
	base.Mode = envOrDefault("NEXUS_AGENT_MODE", base.Mode)
	base.OpenAI.BaseURL = envOrDefault("NEXUS_OPENAI_BASE_URL", base.OpenAI.BaseURL)
	base.OpenAI.APIKey = envOrDefault("NEXUS_OPENAI_API_KEY", base.OpenAI.APIKey)
	base.OpenAI.Model = envOrDefault("NEXUS_OPENAI_MODEL", base.OpenAI.Model)
	base.Embedding.BaseURL = envOrDefault("NEXUS_OPENAI_EMBEDDING_BASE_URL", base.Embedding.BaseURL)
	base.Embedding.APIKey = envOrDefault("NEXUS_OPENAI_EMBEDDING_API_KEY", base.Embedding.APIKey)
	base.Embedding.Model = envOrDefault("NEXUS_OPENAI_EMBEDDING_MODEL", base.Embedding.Model)
	base.SemanticFactCheck = envBool("NEXUS_KNOWLEDGE_SEMANTIC_CHECK", base.SemanticFactCheck)
	// Keep single-provider configurations working without sharing chat credentials
	// with an explicitly different embedding provider.
	if base.Embedding.BaseURL == "" {
		base.Embedding.BaseURL = base.OpenAI.BaseURL
		if base.Embedding.APIKey == "" {
			base.Embedding.APIKey = base.OpenAI.APIKey
		}
	}
	return base
}

func NewRunner(config Config, tools ...Tool) (Runner, error) {
	switch strings.ToLower(strings.TrimSpace(config.Mode)) {
	case "", "mock":
		return MockRunner{}, nil
	case "openai":
		return NewOpenAIRunnerWithTools(config.OpenAI, nil, tools)
	default:
		return nil, fmt.Errorf("unsupported agent mode %q", config.Mode)
	}
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off", "":
		return false
	default:
		return fallback
	}
}

func newHTTPClient() *http.Client {
	// Streaming responses are bounded by the request context rather than a
	// whole-request client timeout.
	return &http.Client{}
}
