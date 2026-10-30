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
	"sort"
	"strings"
	"sync"
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

var (
	ErrNotFound       = errors.New("approval not found")
	ErrAlreadyDecided = errors.New("approval has already been decided")
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
	TicketID   string            `json:"ticketId,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`           // 模型工具提议通过校验、生成待审批记录的时间
	DecidedAt  *time.Time        `json:"decidedAt,omitempty"` // 人工对这条审批做出决定的时间，人工批准或拒绝该提议的时间
}

type Ticket struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Store struct {
	mu        sync.RWMutex
	approvals map[string]*Approval
	tickets   map[string]Ticket
	now       func() time.Time
}

func NewStore() *Store {
	return &Store{
		approvals: make(map[string]*Approval),
		tickets:   make(map[string]Ticket),
		now:       time.Now,
	}
}

// Tool returns the only tool exposed to the model in this increment. The
// callback records a proposal; it never creates a ticket.
func (s *Store) Tool() agent.Tool {
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
		Propose: s.propose,
	}
}

func (s *Store) propose(ctx context.Context, proposal agent.ToolProposal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if proposal.Name != createTicketToolName {
		return fmt.Errorf("unsupported tool %q", proposal.Name)
	}
	if strings.TrimSpace(proposal.ThreadID) == "" || strings.TrimSpace(proposal.RunID) == "" || strings.TrimSpace(proposal.ToolCallID) == "" {
		return errors.New("tool proposal correlation IDs are required")
	}

	// json 字符串转为go struct
	var wireInput struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	decoder := json.NewDecoder(bytes.NewReader(proposal.Arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wireInput); err != nil {
		return fmt.Errorf("invalid create_ticket arguments: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return fmt.Errorf("invalid create_ticket arguments: %w", err)
	}
	// 必选参数
	if wireInput.Title == nil || wireInput.Description == nil {
		return errors.New("ticket title and description are required")
	}
	input := CreateTicketInput{
		Title:       strings.TrimSpace(*wireInput.Title),
		Description: strings.TrimSpace(*wireInput.Description),
	}
	if input.Title == "" {
		return errors.New("ticket title is required")
	}
	if utf8.RuneCountInString(input.Title) > maxTitleLength {
		return fmt.Errorf("ticket title exceeds %d characters", maxTitleLength)
	}
	if utf8.RuneCountInString(input.Description) > maxDescriptionLength {
		return fmt.Errorf("ticket description exceeds %d characters", maxDescriptionLength)
	}

	id, err := newID("approval")
	if err != nil {
		return err
	}
	approval := &Approval{
		ID:         id,
		ThreadID:   proposal.ThreadID,
		RunID:      proposal.RunID,
		ToolCallID: proposal.ToolCallID,
		ToolName:   proposal.Name,
		Arguments:  input,
		Status:     StatusPending,
		CreatedAt:  s.now().UTC(),
	}
	s.mu.Lock()
	s.approvals[id] = approval
	s.mu.Unlock()
	return nil
}

func (s *Store) List() []Approval {
	s.mu.RLock()
	approvals := make([]Approval, 0, len(s.approvals))
	for _, approval := range s.approvals {
		approvals = append(approvals, *approval)
	}
	s.mu.RUnlock()
	sort.Slice(approvals, func(i, j int) bool {
		if approvals[i].CreatedAt.Equal(approvals[j].CreatedAt) {
			return approvals[i].ID < approvals[j].ID
		}
		return approvals[i].CreatedAt.After(approvals[j].CreatedAt)
	})
	return approvals
}

func (s *Store) Approve(ctx context.Context, id string) (Approval, error) {
	if err := ctx.Err(); err != nil {
		return Approval{}, err
	}
	ticketID, err := newID("ticket")
	if err != nil {
		return Approval{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	approval, err := s.pendingApproval(id)
	if err != nil {
		return Approval{}, err
	}
	now := s.now().UTC()
	s.tickets[ticketID] = Ticket{
		ID:          ticketID,
		Title:       approval.Arguments.Title,
		Description: approval.Arguments.Description,
		Status:      "open",
		CreatedAt:   now,
	}
	approval.Status = StatusApproved
	approval.TicketID = ticketID
	approval.DecidedAt = &now
	return *approval, nil
}

func (s *Store) Reject(ctx context.Context, id string) (Approval, error) {
	if err := ctx.Err(); err != nil {
		return Approval{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, err := s.pendingApproval(id)
	if err != nil {
		return Approval{}, err
	}
	now := s.now().UTC()
	approval.Status = StatusRejected
	approval.DecidedAt = &now
	return *approval, nil
}

func (s *Store) pendingApproval(id string) (*Approval, error) {
	approval, ok := s.approvals[id]
	if !ok {
		return nil, ErrNotFound
	}
	if approval.Status != StatusPending {
		return nil, ErrAlreadyDecided
	}
	return approval, nil
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
