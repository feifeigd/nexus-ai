package agent

import "testing"

func TestConfigFromEnvOverridesAgentSettings(t *testing.T) {
	t.Setenv("NEXUS_AGENT_MODE", "openai")
	t.Setenv("NEXUS_OPENAI_BASE_URL", " http://localhost:9999/v1 ")
	t.Setenv("NEXUS_OPENAI_API_KEY", " test-key ")
	t.Setenv("NEXUS_OPENAI_MODEL", " test-model ")

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
