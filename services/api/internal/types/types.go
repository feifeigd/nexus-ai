package types

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type AGUIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type AGUIRunInput struct {
	ThreadID string        `json:"threadId"`
	RunID    string        `json:"runId"`
	Messages []AGUIMessage `json:"messages"`
}

type AGUIEvent struct {
	Type      string `json:"type"`
	ThreadID  string `json:"threadId,omitempty"`
	RunID     string `json:"runId,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Role      string `json:"role,omitempty"`
	Delta     string `json:"delta,omitempty"`
}
