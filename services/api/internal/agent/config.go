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

func newHTTPClient() *http.Client {
	// Streaming responses are bounded by the request context rather than a
	// whole-request client timeout.
	return &http.Client{}
}
