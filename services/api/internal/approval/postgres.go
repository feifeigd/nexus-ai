package approval

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"nexusai/services/api/internal/agent"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/001_approvals.sql
var migrationSQL string

//go:embed migrations/002_ticket_status.sql
var ticketStatusMigrationSQL string

const approvalColumns = `id, thread_id, run_id, tool_call_id, tool_name,
	arguments, status, ticket_id, created_at, decided_at`

type PostgresStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	store := &PostgresStore{db: db, now: time.Now}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := store.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	for _, migration := range []string{migrationSQL, ticketStatusMigrationSQL} {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migrate approval store: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) Tool() agent.Tool {
	return ticketTool(s.propose)
}

func (s *PostgresStore) propose(ctx context.Context, proposal agent.ToolProposal) error {
	approval, err := newPendingApproval(ctx, proposal, s.now().UTC())
	if err != nil {
		return err
	}
	arguments, err := json.Marshal(approval.Arguments)
	if err != nil {
		return fmt.Errorf("encode approval arguments: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO approvals (
		id, thread_id, run_id, tool_call_id, tool_name, arguments, status, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		approval.ID, approval.ThreadID, approval.RunID, approval.ToolCallID,
		approval.ToolName, arguments, approval.Status, approval.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert approval: %w", err)
	}
	return nil
}

func (s *PostgresStore) List(ctx context.Context) ([]Approval, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+approvalColumns+`
		FROM approvals ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	defer rows.Close()

	approvals := make([]Approval, 0)
	for rows.Next() {
		approval, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	return approvals, nil
}

func (s *PostgresStore) ListTickets(ctx context.Context) ([]Ticket, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, description, status, created_at
		FROM tickets ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()

	tickets := make([]Ticket, 0)
	for rows.Next() {
		ticket, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return tickets, nil
}

func (s *PostgresStore) UpdateTicketStatus(ctx context.Context, id string, status TicketStatus) (Ticket, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Ticket{}, fmt.Errorf("begin ticket status transaction: %w", err)
	}
	defer tx.Rollback()

	ticket, err := ticketByID(ctx, tx, id, true)
	if err != nil {
		return Ticket{}, err
	}
	if err := validateTicketTransition(ticket.Status, status); err != nil {
		return Ticket{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tickets SET status = $2 WHERE id = $1`, id, status); err != nil {
		return Ticket{}, fmt.Errorf("update ticket status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket status transaction: %w", err)
	}
	ticket.Status = status
	return ticket, nil
}

func (s *PostgresStore) Approve(ctx context.Context, id string) (Approval, error) {
	ticketID, err := newID("ticket")
	if err != nil {
		return Approval{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Approval{}, fmt.Errorf("begin approval transaction: %w", err)
	}
	defer tx.Rollback()

	approval, err := approvalByID(ctx, tx, id, true)
	if err != nil {
		return Approval{}, err
	}
	if approval.Status != StatusPending {
		return Approval{}, ErrAlreadyDecided
	}
	now := s.now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO tickets (
		id, approval_id, title, description, status, created_at
	) VALUES ($1, $2, $3, $4, 'open', $5)`,
		ticketID, approval.ID, approval.Arguments.Title, approval.Arguments.Description, now,
	); err != nil {
		return Approval{}, fmt.Errorf("create approved ticket: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE approvals
		SET status = $2, ticket_id = $3, decided_at = $4 WHERE id = $1`,
		id, StatusApproved, ticketID, now,
	); err != nil {
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

func (s *PostgresStore) Reject(ctx context.Context, id string) (Approval, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Approval{}, fmt.Errorf("begin rejection transaction: %w", err)
	}
	defer tx.Rollback()

	approval, err := approvalByID(ctx, tx, id, true)
	if err != nil {
		return Approval{}, err
	}
	if approval.Status != StatusPending {
		return Approval{}, ErrAlreadyDecided
	}
	now := s.now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE approvals
		SET status = $2, decided_at = $3 WHERE id = $1`,
		id, StatusRejected, now,
	); err != nil {
		return Approval{}, fmt.Errorf("reject proposal: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Approval{}, fmt.Errorf("commit rejection transaction: %w", err)
	}
	approval.Status = StatusRejected
	approval.DecidedAt = &now
	return approval, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func approvalByID(ctx context.Context, db queryRower, id string, lock bool) (Approval, error) {
	query := `SELECT ` + approvalColumns + ` FROM approvals WHERE id = $1`
	if lock {
		query += ` FOR UPDATE`
	}
	approval, err := scanApproval(db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Approval{}, ErrNotFound
	}
	return approval, err
}

func ticketByID(ctx context.Context, db queryRower, id string, lock bool) (Ticket, error) {
	query := `SELECT id, title, description, status, created_at FROM tickets WHERE id = $1`
	if lock {
		query += ` FOR UPDATE`
	}
	ticket, err := scanTicket(db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Ticket{}, ErrTicketNotFound
	}
	return ticket, err
}

type scanner interface {
	Scan(...any) error
}

func scanTicket(row scanner) (Ticket, error) {
	var ticket Ticket
	if err := row.Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status, &ticket.CreatedAt,
	); err != nil {
		return Ticket{}, fmt.Errorf("scan ticket: %w", err)
	}
	ticket.CreatedAt = ticket.CreatedAt.UTC()
	return ticket, nil
}

func scanApproval(row scanner) (Approval, error) {
	var approval Approval
	var arguments []byte
	var ticketID sql.NullString
	var decidedAt sql.NullTime
	if err := row.Scan(
		&approval.ID, &approval.ThreadID, &approval.RunID, &approval.ToolCallID,
		&approval.ToolName, &arguments, &approval.Status, &ticketID,
		&approval.CreatedAt, &decidedAt,
	); err != nil {
		return Approval{}, fmt.Errorf("scan approval: %w", err)
	}
	if err := json.Unmarshal(arguments, &approval.Arguments); err != nil {
		return Approval{}, fmt.Errorf("decode approval arguments: %w", err)
	}
	approval.TicketID = ticketID.String
	if decidedAt.Valid {
		value := decidedAt.Time.UTC()
		approval.DecidedAt = &value
	}
	approval.CreatedAt = approval.CreatedAt.UTC()
	return approval, nil
}
