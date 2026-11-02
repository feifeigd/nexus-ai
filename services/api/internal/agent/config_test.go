package agent

import "testing"

func TestConfigFromEnvOverridesAgentSettings(t *testing.T) {
	t.Setenv("NEXUS_AGENT_MODE", "openai")
	t.Setenv("NEXUS_OPENAI_BASE_URL", " http://localhost:9999/v1 ")
	t.Setenv("NEXUS_OPENAI_API_KEY", " test-key ")
	t.Setenv("NEXUS_OPENAI_MODEL", " test-model ")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_BASE_URL", " http://localhost:9998/v1 ")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_API_KEY", " test-embedding-key ")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_MODEL", " test-embedding-model ")

	config := ConfigFromEnv(Config{Mode: "mock"})
	if config.Mode != "openai" {
		t.Errorf("unexpected mode: %q", config.Mode)
	}
	if config.OpenAI.BaseURL != "http://localhost:9999/v1" {
		t.Errorf("unexpected base URL: %q", config.OpenAI.BaseURL)
	}
	if config.OpenAI.APIKey != "test-key" {
		t.Errorf("unexpected API key: %q", config.OpenAI.APIKey)
	}
	if config.OpenAI.Model != "test-model" {
		t.Errorf("unexpected model: %q", config.OpenAI.Model)
	}
	if config.Embedding.BaseURL != "http://localhost:9998/v1" {
		t.Errorf("unexpected embedding base URL: %q", config.Embedding.BaseURL)
	}
	if config.Embedding.APIKey != "test-embedding-key" {
		t.Errorf("unexpected embedding API key: %q", config.Embedding.APIKey)
	}
	if config.Embedding.Model != "test-embedding-model" {
		t.Errorf("unexpected embedding model: %q", config.Embedding.Model)
	}
}

func TestConfigFromEnvReusesSingleProviderForEmbedding(t *testing.T) {
	t.Setenv("NEXUS_OPENAI_BASE_URL", "http://localhost:9999/v1")
	t.Setenv("NEXUS_OPENAI_API_KEY", "test-key")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_BASE_URL", "")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_API_KEY", "")

	config := ConfigFromEnv(Config{})
	if config.Embedding.BaseURL != config.OpenAI.BaseURL || config.Embedding.APIKey != config.OpenAI.APIKey {
		t.Fatalf("expected embedding provider to reuse chat provider: %+v", config.Embedding)
	}
}

func TestConfigFromEnvDoesNotShareChatKeyWithDifferentEmbeddingProvider(t *testing.T) {
	t.Setenv("NEXUS_OPENAI_BASE_URL", "http://localhost:9999/v1")
	t.Setenv("NEXUS_OPENAI_API_KEY", "chat-secret")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_BASE_URL", "http://localhost:9998/v1")
	t.Setenv("NEXUS_OPENAI_EMBEDDING_API_KEY", "")

	config := ConfigFromEnv(Config{})
	if config.Embedding.APIKey != "" {
		t.Fatal("expected a different embedding provider to require its own API key")
	}
}

func TestNewRunnerDefaultsToMock(t *testing.T) {
	runner, err := NewRunner(Config{})
	if err != nil {
		t.Fatalf("create default runner: %v", err)
	}
	if _, ok := runner.(MockRunner); !ok {
		t.Fatalf("expected MockRunner, got %T", runner)
	}
}

func TestNewRunnerRejectsUnknownMode(t *testing.T) {
	if _, err := NewRunner(Config{Mode: "unknown"}); err == nil {
		t.Fatal("expected an unsupported mode error")
	}
}
