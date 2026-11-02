package agent

import (
	"context"
	"encoding/json"
)

// Tool describes a server-owned tool. Exactly one of Execute or Propose must
// be configured. Execute is reserved for read-only operations; Propose must
// validate and record privileged operations without executing them before
// human approval.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Execute     ToolExecutionFunc // 立即执行只读操作。查询、检索等只读操作
	Propose     ToolProposalFunc  // 提交敏感操作的人工审批申请。创建、修改、删除等敏感操作，只记录审批申请，不执行实际业务操作
	// ValidateAnswer optionally validates the follow-up answer produced after
	// this tool's result was included in a second model request. It returns an
	// empty string when the answer is acceptable, or a non-empty note describing
	// detected problems, such as citations to sources outside the tool result.
	// Reserved for read-only tools that return sourced content.
	ValidateAnswer func(ctx context.Context, answer, toolResult string) (string, error)
}

// 模型提出的一次完整工具调用
type ToolProposal struct {
	ThreadID   string
	RunID      string
	ToolCallID string
	Name       string
	Arguments  json.RawMessage
}

type ToolExecutionFunc func(context.Context, ToolProposal) (string, error)
type ToolProposalFunc func(context.Context, ToolProposal) error
