package agent

import (
	"context"
	"encoding/json"
)

// Tool describes a server-owned tool. Propose must validate and record the
// request without executing a privileged operation before human approval.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Propose     ToolProposalFunc
}

// 模型提出的一次完整工具调用
type ToolProposal struct {
	ThreadID   string
	RunID      string
	ToolCallID string
	Name       string
	Arguments  json.RawMessage
}

type ToolProposalFunc func(context.Context, ToolProposal) error
