package approval

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"nexusai/services/api/internal/agent"
)

const (
	createTicketToolName = "create_ticket"
	// 字数限制
	maxTitleLength       = 200
	maxDescriptionLength = 2000
)

type Status string

const (
	StatusPending  Status = "pending"  // Approval.DecidedAt == nil
	StatusApproved Status = "approved" // 批准
	StatusRejected Status = "rejected" // 拒绝
)

type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusResolved   TicketStatus = "resolved"
)

var (
	ErrNotFound                = errors.New("approval not found")
	ErrAlreadyDecided          = errors.New("approval has already been decided")
	ErrTicketNotFound          = errors.New("ticket not found")
	ErrInvalidTicketStatus     = errors.New("invalid ticket status")
	ErrInvalidTicketTransition = errors.New("invalid ticket status transition")
)

type CreateTicketInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Approval struct {
	ID         string            `json:"id"`
	ThreadID   string            `json:"threadId"`
	RunID      string            `json:"runId"`
	ToolCallID string            `json:"toolCallId"`
	ToolName   string            `json:"toolName"`
	Arguments  CreateTicketInput `json:"arguments"` //  LLM 输出的工具调用参数
	Status     Status            `json:"status"`
	TenantID   string            `json:"tenantId,omitempty"` // 提出该审批的用户所属租户（每用户一个租户）
	TicketID   string            `json:"ticketId,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`           // 模型工具提议通过校验、生成待审批记录的时间
	DecidedAt  *time.Time        `json:"decidedAt,omitempty"` // 人工对这条审批做出决定的时间，人工批准或拒绝该提议的时间
}

type Ticket struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	CreatedAt   time.Time    `json:"createdAt"`
}

func validateTicketTransition(current, next TicketStatus) error {
	if next != TicketStatusOpen && next != TicketStatusInProgress && next != TicketStatusResolved {
		return ErrInvalidTicketStatus
	}
	if current == TicketStatusOpen && next == TicketStatusInProgress ||
		current == TicketStatusInProgress && next == TicketStatusResolved {
		return nil
	}
	return ErrInvalidTicketTransition
}

type Store interface {
	Tool() agent.Tool
	List(ctx context.Context, tenantID string) ([]Approval, error)
	ListTickets(ctx context.Context, tenantID string) ([]Ticket, error)
	UpdateTicketStatus(ctx context.Context, id string, status TicketStatus, tenantID string) (Ticket, error)
	Approve(ctx context.Context, id, tenantID string) (Approval, error)
	Reject(ctx context.Context, id, tenantID string) (Approval, error)
}

func ticketTool(propose func(context.Context, agent.ToolProposal) error) agent.Tool {
	return agent.Tool{
		Name:        createTicketToolName,
		Description: "提议创建一张工单。必须获得用户明确指令，创建前需要人工审批。",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {"type": "string", "minLength": 1, "maxLength": 200},
				"description": {"type": "string", "maxLength": 2000}
			},
			"required": ["title", "description"],
			"additionalProperties": false
		}`),
		Propose: propose,
	}
}

func newPendingApproval(ctx context.Context, proposal agent.ToolProposal, now time.Time) (Approval, error) {
	if err := ctx.Err(); err != nil {
		return Approval{}, err
	}
	if proposal.Name != createTicketToolName {
		return Approval{}, fmt.Errorf("unsupported tool %q", proposal.Name)
	}
	if strings.TrimSpace(proposal.ThreadID) == "" || strings.TrimSpace(proposal.RunID) == "" || strings.TrimSpace(proposal.ToolCallID) == "" {
		return Approval{}, errors.New("tool proposal correlation IDs are required")
	}

	var wireInput struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	decoder := json.NewDecoder(bytes.NewReader(proposal.Arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wireInput); err != nil {
		return Approval{}, fmt.Errorf("invalid create_ticket arguments: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Approval{}, fmt.Errorf("invalid create_ticket arguments: %w", err)
	}
	if wireInput.Title == nil || wireInput.Description == nil {
		return Approval{}, errors.New("ticket title and description are required")
	}
	input := CreateTicketInput{
		Title:       strings.TrimSpace(*wireInput.Title),
		Description: strings.TrimSpace(*wireInput.Description),
	}
	if input.Title == "" {
		return Approval{}, errors.New("ticket title is required")
	}
	if utf8.RuneCountInString(input.Title) > maxTitleLength {
		return Approval{}, fmt.Errorf("ticket title exceeds %d characters", maxTitleLength)
	}
	if utf8.RuneCountInString(input.Description) > maxDescriptionLength {
		return Approval{}, fmt.Errorf("ticket description exceeds %d characters", maxDescriptionLength)
	}

	id, err := newID("approval")
	if err != nil {
		return Approval{}, err
	}
	return Approval{
		ID:         id,
		ThreadID:   proposal.ThreadID,
		RunID:      proposal.RunID,
		ToolCallID: proposal.ToolCallID,
		ToolName:   proposal.Name,
		Arguments:  input,
		Status:     StatusPending,
		CreatedAt:  now,
	}, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func newID(prefix string) (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate %s ID: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(value[:]), nil
}
