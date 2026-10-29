package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nexusai/services/api/internal/types"
)

const maxModelEventSize = 1 << 20

type OpenAIRunner struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// 实现接口
var _ Runner = (*OpenAIRunner)(nil)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []chatMessage `json:"messages"`
}

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func NewOpenAIRunner(config OpenAIConfig, client *http.Client) (*OpenAIRunner, error) {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		return nil, fmt.Errorf("NEXUS_OPENAI_MODEL is required in openai mode")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	if client == nil {
		client = newHTTPClient()
	}
	return &OpenAIRunner{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(config.APIKey),
		model:   model,
		client:  client,
	}, nil
}

func (r *OpenAIRunner) Run(ctx context.Context, input types.AGUIRunInput, emit EmitFunc) error {
	messages, err := convertMessages(input.Messages)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(chatCompletionRequest{
		Model:    r.model,
		Stream:   true,
		Messages: messages,
	})
	if err != nil {
		return fmt.Errorf("encode model request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		r.baseURL+"/chat/completions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("create model request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("call model provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("model provider returned HTTP %d", resp.StatusCode)
	}

	messageID := input.RunID + "-assistant"
	if err := emit(types.AGUIEvent{Type: "RUN_STARTED", ThreadID: input.ThreadID, RunID: input.RunID}); err != nil {
		return err
	}
	if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_START", MessageID: messageID, Role: "assistant"}); err != nil {
		return err
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), maxModelEventSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("decode model stream event: %w", err)
		}
		if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
			return fmt.Errorf("model provider returned a stream error")
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := emit(types.AGUIEvent{
				Type:      "TEXT_MESSAGE_CONTENT",
				MessageID: messageID,
				Delta:     choice.Delta.Content,
			}); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read model stream: %w", err)
	}
	if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: messageID}); err != nil {
		return err
	}
	return emit(types.AGUIEvent{Type: "RUN_FINISHED", ThreadID: input.ThreadID, RunID: input.RunID})
}

func convertMessages(input []types.AGUIMessage) ([]chatMessage, error) {
	messages := make([]chatMessage, 0, len(input))
	for _, message := range input {
		switch message.Role {
		case "system", "user", "assistant":
		default:
			continue
		}
		content, ok := message.Content.(string)
		if !ok {
			return nil, fmt.Errorf("%s messages must contain plain text", message.Role)
		}
		messages = append(messages, chatMessage{Role: message.Role, Content: content})
	}
	return messages, nil
}
