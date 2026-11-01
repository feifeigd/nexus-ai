package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nexusai/services/api/internal/agent"
	dataent "nexusai/services/api/internal/data/ent"
	entapproval "nexusai/services/api/internal/data/ent/approval"
	entticket "nexusai/services/api/internal/data/ent/ticket"

	"entgo.io/ent/dialect"
)

// EntStore provides approval and ticket persistence using a process-owned Ent
// client. The client lifecycle and database migrations belong to internal/data.
type EntStore struct {
	client    *dataent.Client
	now       func() time.Time
	forUpdate bool
}

func NewEntStore(client *dataent.Client, driver string) *EntStore {
	return &EntStore{
		client:    client,
		now:       time.Now,
		forUpdate: driver == dialect.Postgres,
	}
}

func (s *EntStore) Tool() agent.Tool {
	return ticketTool(s.propose)
}

func (s *EntStore) propose(ctx context.Context, proposal agent.ToolProposal) error {
	approval, err := newPendingApproval(ctx, proposal, s.now().UTC())
	if err != nil {
		return err
	}
	err = s.client.Approval.Create().
		SetID(approval.ID).
		SetThreadID(approval.ThreadID).
		SetRunID(approval.RunID).
		SetToolCallID(approval.ToolCallID).
		SetToolName(approval.ToolName).
		SetArguments(map[string]any{
			"title":       approval.Arguments.Title,
			"description": approval.Arguments.Description,
		}).
		SetStatus(string(approval.Status)).
		SetCreatedAt(approval.CreatedAt).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert approval: %w", err)
	}
	return nil
}

func (s *EntStore) List(ctx context.Context) ([]Approval, error) {
	rows, err := s.client.Approval.Query().
		Order(dataent.Desc(entapproval.FieldCreatedAt), dataent.Asc(entapproval.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	approvals := make([]Approval, 0, len(rows))
	for _, row := range rows {
		approval, err := approvalFromEnt(row)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	return approvals, nil
}

func (s *EntStore) ListTickets(ctx context.Context) ([]Ticket, error) {
	rows, err := s.client.Ticket.Query().
		Order(dataent.Desc(entticket.FieldCreatedAt), dataent.Asc(entticket.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	tickets := make([]Ticket, 0, len(rows))
	for _, row := range rows {
		tickets = append(tickets, ticketFromEnt(row))
	}
	return tickets, nil
}

func (s *EntStore) UpdateTicketStatus(ctx context.Context, id string, status TicketStatus) (Ticket, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("begin ticket status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ticket, err := s.lockedTicketByID(ctx, tx.Client(), id)
	if err != nil {
		return Ticket{}, err
	}
	if err := validateTicketTransition(ticket.Status, status); err != nil {
		return Ticket{}, err
	}
	if err := tx.Client().Ticket.UpdateOneID(id).
		SetStatus(string(status)).
		Exec(ctx); err != nil {
		return Ticket{}, fmt.Errorf("update ticket status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket status transaction: %w", err)
	}
	ticket.Status = status
	return ticket, nil
}

func (s *EntStore) Approve(ctx context.Context, id string) (Approval, error) {
	ticketID, err := newID("ticket")
	if err != nil {
		return Approval{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Approval{}, fmt.Errorf("begin approval transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	approval, err := s.lockedApprovalByID(ctx, tx.Client(), id)
	if err != nil {
		return Approval{}, err
	}
	if approval.Status != StatusPending {
		return Approval{}, ErrAlreadyDecided
	}
	now := s.now().UTC()
	if err := tx.Client().Ticket.Create().
		SetID(ticketID).
		SetApprovalID(approval.ID).
		SetTitle(approval.Arguments.Title).
		SetDescription(approval.Arguments.Description).
		SetStatus(string(TicketStatusOpen)).
		SetCreatedAt(now).
		Exec(ctx); err != nil {
		return Approval{}, fmt.Errorf("create approved ticket: %w", err)
	}
	if err := tx.Client().Approval.UpdateOneID(id).
		SetStatus(string(StatusApproved)).
		SetTicketID(ticketID).
		SetDecidedAt(now).
		Exec(ctx); err != nil {
		return Approval{}, fmt.Errorf("approve proposal: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Approval{}, fmt.Errorf("commit approval transaction: %w", err)
	}
	approval.Status = StatusApproved
	approval.TicketID = ticketID
	approval.DecidedAt = &now
	return approval, nil
}

func (s *EntStore) Reject(ctx context.Context, id string) (Approval, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Approval{}, fmt.Errorf("begin rejection transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	approval, err := s.lockedApprovalByID(ctx, tx.Client(), id)
	if err != nil {
		return Approval{}, err
	}
	if approval.Status != StatusPending {
		return Approval{}, ErrAlreadyDecided
	}
	now := s.now().UTC()
	if err := tx.Client().Approval.UpdateOneID(id).
		SetStatus(string(StatusRejected)).
		SetDecidedAt(now).
		Exec(ctx); err != nil {
		return Approval{}, fmt.Errorf("reject proposal: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Approval{}, fmt.Errorf("commit rejection transaction: %w", err)
	}
	approval.Status = StatusRejected
	approval.DecidedAt = &now
	return approval, nil
}

func (s *EntStore) lockedApprovalByID(ctx context.Context, client *dataent.Client, id string) (Approval, error) {
	query := client.Approval.Query().Where(entapproval.ID(id))
	if s.forUpdate {
		query = query.ForUpdate()
	}
	row, err := query.Only(ctx)
	if dataent.IsNotFound(err) {
		return Approval{}, ErrNotFound
	}
	if err != nil {
		return Approval{}, fmt.Errorf("query approval: %w", err)
	}
	return approvalFromEnt(row)
}

func (s *EntStore) lockedTicketByID(ctx context.Context, client *dataent.Client, id string) (Ticket, error) {
	query := client.Ticket.Query().Where(entticket.ID(id))
	if s.forUpdate {
		query = query.ForUpdate()
	}
	row, err := query.Only(ctx)
	if dataent.IsNotFound(err) {
		return Ticket{}, ErrTicketNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("query ticket: %w", err)
	}
	return ticketFromEnt(row), nil
}

func ticketFromEnt(row *dataent.Ticket) Ticket {
	return Ticket{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Status:      TicketStatus(row.Status),
		CreatedAt:   row.CreatedAt.UTC(),
	}
}

func approvalFromEnt(row *dataent.Approval) (Approval, error) {
	arguments, err := json.Marshal(row.Arguments)
	if err != nil {
		return Approval{}, fmt.Errorf("encode approval arguments: %w", err)
	}
	approval := Approval{
		ID:         row.ID,
		ThreadID:   row.ThreadID,
		RunID:      row.RunID,
		ToolCallID: row.ToolCallID,
		ToolName:   row.ToolName,
		Status:     Status(row.Status),
		CreatedAt:  row.CreatedAt.UTC(),
	}
	if err := json.Unmarshal(arguments, &approval.Arguments); err != nil {
		return Approval{}, fmt.Errorf("decode approval arguments: %w", err)
	}
	if row.TicketID != nil {
		approval.TicketID = *row.TicketID
	}
	if row.DecidedAt != nil {
		value := row.DecidedAt.UTC()
		approval.DecidedAt = &value
	}
	return approval, nil
}
