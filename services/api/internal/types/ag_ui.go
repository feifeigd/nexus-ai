package types

// AGUIEvent remains hand-maintained because it is an outbound streaming event,
// not an HTTP request or response body described by the API DSL.
type AGUIEvent struct {
	Type         string `json:"type"`
	ThreadID     string `json:"threadId,omitempty"`
	RunID        string `json:"runId,omitempty"`
	MessageID    string `json:"messageId,omitempty"`
	Role         string `json:"role,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Message      string `json:"message,omitempty"`
	ToolCallID   string `json:"toolCallId,omitempty"`
	ToolCallName string `json:"toolCallName,omitempty"`
}
