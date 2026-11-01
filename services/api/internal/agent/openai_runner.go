package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"nexusai/services/api/internal/types"
)

const maxModelEventSize = 1 << 20

const latestUserToolInstruction = "调用工具时，只能依据最新一条用户消息中明确提出的操作请求。不得因为历史消息曾要求操作，就在‘然后呢’‘继续’‘接着’等后续追问中重复调用工具；这类追问应基于已有对话回答，并提示用户到相应业务页面查看进度。只读工具返回后，只能依据工具结果回答；结果不包含答案时明确说明，不得编造。"

type OpenAIRunner struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	tools   []Tool // 支持的工具
}

// 实现接口
var _ Runner = (*OpenAIRunner)(nil)

type chatMessage struct {
	Role       string                   `json:"role"`
	Content    string                   `json:"content"`
	ToolCallID string                   `json:"tool_call_id,omitempty"`
	ToolCalls  []chatCompletionToolCall `json:"tool_calls,omitempty"`
}

type chatCompletionRequest struct {
	Model    string               `json:"model"`
	Stream   bool                 `json:"stream"`
	Messages []chatMessage        `json:"messages"`
	Tools    []chatCompletionTool `json:"tools,omitempty"`
}

type chatCompletionTool struct {
	Type     string                     `json:"type"` // function/
	Function chatCompletionToolFunction `json:"function"`
}

type chatCompletionToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content   string                   `json:"content"`
			ToolCalls []chatCompletionToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

type chatCompletionToolCall struct {
	Index    int    `json:"index,omitempty"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type pendingToolCall struct {
	id        string
	name      string
	arguments strings.Builder
	started   bool
}

func NewOpenAIRunner(config OpenAIConfig, client *http.Client) (*OpenAIRunner, error) {
	return NewOpenAIRunnerWithTools(config, client, nil)
}

func NewOpenAIRunnerWithTools(config OpenAIConfig, client *http.Client, tools []Tool) (*OpenAIRunner, error) {
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
	if err := validateTools(tools); err != nil {
		return nil, err
	}
	return &OpenAIRunner{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(config.APIKey),
		model:   model,
		client:  client,
		tools:   tools,
	}, nil
}

func (r *OpenAIRunner) Run(ctx context.Context, input types.AGUIRunInput, emit EmitFunc) error {
	messages, err := convertMessages(input.Messages)
	if err != nil {
		return err
	}
	if len(r.tools) > 0 {
		messages = append([]chatMessage{{Role: "system", Content: latestUserToolInstruction}}, messages...)
	}
	tools := make([]chatCompletionTool, 0, len(r.tools))
	toolsByName := make(map[string]Tool, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, chatCompletionTool{
			Type: "function",
			Function: chatCompletionToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
		toolsByName[tool.Name] = tool
	}
	payload, err := json.Marshal(chatCompletionRequest{
		Model:    r.model,
		Stream:   true,
		Messages: messages,
		Tools:    tools,
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
		return modelProviderHTTPError(resp, "")
	}

	messageID := input.RunID + "-assistant"
	if err := emit(types.AGUIEvent{Type: "RUN_STARTED", ThreadID: input.ThreadID, RunID: input.RunID}); err != nil {
		return err
	}
	textStarted := false
	startText := func() error {
		if textStarted {
			return nil
		}
		if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_START", MessageID: messageID, Role: "assistant"}); err != nil {
			return err
		}
		textStarted = true
		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), maxModelEventSize)
	streamCompleted := false
	pendingCalls := make(map[int]*pendingToolCall)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			streamCompleted = true
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
			if choice.FinishReason != "" {
				streamCompleted = true
			}
			if choice.Delta.Content != "" {
				if err := startText(); err != nil {
					return err
				}
				if err := emit(types.AGUIEvent{
					Type:      "TEXT_MESSAGE_CONTENT",
					MessageID: messageID,
					Delta:     choice.Delta.Content,
				}); err != nil {
					return err
				}
			}
			for _, delta := range choice.Delta.ToolCalls {
				call := pendingCalls[delta.Index]
				if call == nil {
					call = &pendingToolCall{}
					pendingCalls[delta.Index] = call
				}
				if delta.ID != "" {
					if call.id != "" && call.id != delta.ID {
						return fmt.Errorf("model changed tool call ID at index %d", delta.Index)
					}
					call.id = delta.ID
				}
				if delta.Function.Name != "" {
					if call.name != "" && call.name != delta.Function.Name {
						return fmt.Errorf("model changed tool name at index %d", delta.Index)
					}
					call.name = delta.Function.Name
				}
				if !call.started && call.id != "" && call.name != "" {
					if err := emit(types.AGUIEvent{
						Type:         "TOOL_CALL_START",
						ToolCallID:   call.id,
						ToolCallName: call.name,
					}); err != nil {
						return err
					}
					call.started = true
				}
				if delta.Function.Arguments != "" {
					if !call.started {
						return fmt.Errorf("model sent tool arguments before tool metadata")
					}
					call.arguments.WriteString(delta.Function.Arguments)
					if err := emit(types.AGUIEvent{
						Type:       "TOOL_CALL_ARGS",
						ToolCallID: call.id,
						Delta:      delta.Function.Arguments,
					}); err != nil {
						return err
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read model stream: %w", err)
	}
	if !streamCompleted {
		return fmt.Errorf("model provider stream ended before completion")
	}

	indexes := make([]int, 0, len(pendingCalls))
	for index := range pendingCalls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	assistantCalls := make([]chatCompletionToolCall, 0, len(indexes))
	toolMessages := make([]chatMessage, 0, len(indexes))
	followUp := false
	for _, index := range indexes {
		call := pendingCalls[index]
		if !call.started || call.arguments.Len() == 0 {
			return fmt.Errorf("model returned an incomplete tool call at index %d", index)
		}
		tool, ok := toolsByName[call.name]
		if !ok {
			return fmt.Errorf("model requested unsupported tool %q", call.name)
		}
		arguments := json.RawMessage([]byte(call.arguments.String()))
		if !json.Valid(arguments) {
			return fmt.Errorf("model returned invalid JSON for tool %q", call.name)
		}
		proposal := ToolProposal{
			ThreadID:   input.ThreadID,
			RunID:      input.RunID,
			ToolCallID: call.id,
			Name:       call.name,
			Arguments:  arguments,
		}
		result := "已提交人工审批，请在“工具审批”页面处理；此时尚未创建工单。"
		if tool.Execute != nil {
			var err error
			result, err = tool.Execute(ctx, proposal)
			if err != nil {
				return fmt.Errorf("execute tool %q: %w", call.name, err)
			}
			followUp = true
		} else if err := tool.Propose(ctx, proposal); err != nil {
			return fmt.Errorf("record tool proposal: %w", err)
		}
		assistantCall := chatCompletionToolCall{ID: call.id, Type: "function"}
		assistantCall.Function.Name = call.name
		assistantCall.Function.Arguments = call.arguments.String()
		assistantCalls = append(assistantCalls, assistantCall)
		toolMessages = append(toolMessages, chatMessage{
			Role:       "tool",
			ToolCallID: call.id,
			Content:    result,
		})
		if err := emit(types.AGUIEvent{Type: "TOOL_CALL_END", ToolCallID: call.id}); err != nil {
			return err
		}
		if err := emit(types.AGUIEvent{
			Type:       "TOOL_CALL_RESULT",
			MessageID:  call.id + "-result",
			ToolCallID: call.id,
			Content:    result,
		}); err != nil {
			return err
		}
	}
	if textStarted {
		if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: messageID}); err != nil {
			return err
		}
	}
	// 发起第二次模型请求
	if followUp {
		followUpMessages := make([]chatMessage, 0, len(messages)+1+len(toolMessages))
		followUpMessages = append(followUpMessages, messages...)
		followUpMessages = append(followUpMessages, chatMessage{Role: "assistant", ToolCalls: assistantCalls})
		followUpMessages = append(followUpMessages, toolMessages...)
		if err := r.streamFollowUp(ctx, followUpMessages, messageID+"-follow-up", emit); err != nil {
			return err
		}
	}
	return emit(types.AGUIEvent{Type: "RUN_FINISHED", ThreadID: input.ThreadID, RunID: input.RunID})
}

func (r *OpenAIRunner) streamFollowUp(ctx context.Context, messages []chatMessage, messageID string, emit EmitFunc) error {
	payload, err := json.Marshal(chatCompletionRequest{Model: r.model, Stream: true, Messages: messages})
	if err != nil {
		return fmt.Errorf("encode read-only tool follow-up request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create read-only tool follow-up request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("call model provider for read-only tool follow-up: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return modelProviderHTTPError(resp, " for read-only tool follow-up")
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), maxModelEventSize)
	textStarted := false
	streamCompleted := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			streamCompleted = true
			break
		}
		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("decode read-only tool follow-up stream: %w", err)
		}
		if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
			return fmt.Errorf("model provider returned a read-only tool follow-up stream error")
		}
		for _, choice := range chunk.Choices {
			if choice.FinishReason != "" {
				streamCompleted = true
			}
			if len(choice.Delta.ToolCalls) > 0 {
				return fmt.Errorf("model requested another tool call during read-only tool follow-up")
			}
			if choice.Delta.Content == "" {
				continue
			}
			if !textStarted {
				if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_START", MessageID: messageID, Role: "assistant"}); err != nil {
					return err
				}
				textStarted = true
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
		return fmt.Errorf("read read-only tool follow-up stream: %w", err)
	}
	if !streamCompleted {
		return fmt.Errorf("model provider read-only tool follow-up ended before completion")
	}
	if textStarted {
		if err := emit(types.AGUIEvent{Type: "TEXT_MESSAGE_END", MessageID: messageID}); err != nil {
			return err
		}
	}
	return nil
}

func modelProviderHTTPError(resp *http.Response, operation string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := fmt.Sprintf("model provider returned HTTP %d%s", resp.StatusCode, operation)
	if detail := strings.TrimSpace(string(body)); detail != "" {
		message += ": " + detail
	}
	return fmt.Errorf("%s", message)
}

func validateTools(tools []Tool) error {
	names := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return fmt.Errorf("tool name is required")
		}
		// 不能同时配置这两个字段
		if (tool.Execute == nil) == (tool.Propose == nil) {
			return fmt.Errorf("tool %q requires exactly one execution or proposal handler", tool.Name)
		}
		if _, exists := names[tool.Name]; exists {
			return fmt.Errorf("duplicate tool %q", tool.Name)
		}
		if !json.Valid(tool.Parameters) {
			return fmt.Errorf("tool %q has an invalid parameter schema", tool.Name)
		}
		names[tool.Name] = struct{}{}
	}
	return nil
}

func convertMessages(input []types.AGUIMessage) ([]chatMessage, error) {
	messages := make([]chatMessage, 0, len(input))
	for _, message := range input {
		switch message.Role {
		case "system", "user", "assistant":
		default:
			continue
		}
		messages = append(messages, chatMessage{Role: message.Role, Content: message.Content})
	}
	return messages, nil
}
