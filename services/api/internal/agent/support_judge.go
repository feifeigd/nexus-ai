package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const supportJudgeSystemPrompt = "你是知识库回答的事实核验器。对每个候选句子，判断它是否被给定的知识分块支撑（句子内容可由分块直接得到或合理改写得到）。只输出一个 JSON 对象，格式为 {\"supported\":[true,false,...]}，数组长度必须与候选句子数量一致，不要输出任何解释。"

// OpenAISupportJudge 用 OpenAI 兼容对话模型做句子级支撑判定（NLI/LLM 判定）。
// 每次用一次非流式请求批量判定全部句子，避免逐句多次调用带来的额外延迟。
type OpenAISupportJudge struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAISupportJudge(config OpenAIConfig, client *http.Client) (*OpenAISupportJudge, error) {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		return nil, fmt.Errorf("NEXUS_OPENAI_MODEL is required for semantic fact checking")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	if client == nil {
		client = newHTTPClient()
	}
	return &OpenAISupportJudge{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(config.APIKey),
		model:   model,
		client:  client,
	}, nil
}

type supportJudgeResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func (j *OpenAISupportJudge) Judge(ctx context.Context, sentences []string, chunks []string) ([]bool, error) {
	if len(sentences) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(chatCompletionRequest{
		Model:  j.model,
		Stream: false,
		Messages: []chatMessage{
			{Role: "system", Content: supportJudgeSystemPrompt},
			{Role: "user", Content: buildSupportJudgePrompt(sentences, chunks)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode support judge request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create support judge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if j.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.apiKey)
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call support judge provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("support judge provider returned HTTP %d", resp.StatusCode)
	}
	var decoded supportJudgeResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode support judge response: %w", err)
	}
	if len(decoded.Error) > 0 && string(decoded.Error) != "null" {
		return nil, fmt.Errorf("support judge provider returned an error response")
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("support judge provider returned no choices")
	}
	return parseSupportVerdicts(decoded.Choices[0].Message.Content, len(sentences))
}

func buildSupportJudgePrompt(sentences, chunks []string) string {
	var b strings.Builder
	b.WriteString("知识分块：\n")
	for i, chunk := range chunks {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, chunk)
	}
	b.WriteString("\n候选句子：\n")
	for i, sentence := range sentences {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, sentence)
	}
	b.WriteString("\n请输出 JSON：")
	return b.String()
}

func parseSupportVerdicts(content string, want int) ([]bool, error) {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("support judge returned non-JSON content")
	}
	var parsed struct {
		Supported []bool `json:"supported"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("decode support judge verdicts: %w", err)
	}
	if len(parsed.Supported) != want {
		return nil, fmt.Errorf("support judge returned %d verdicts for %d sentences", len(parsed.Supported), want)
	}
	return parsed.Supported, nil
}
