package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"nexusai/services/api/internal/data/ent"
	entauditlog "nexusai/services/api/internal/data/ent/auditlog"
)

// AuditRecord 是一条待写入的审计记录。字段尽量精简，避免记录敏感内容。
type AuditRecord struct {
	TenantID      string
	ActorID       string
	ActorUsername string
	Action        string // approve_approval / update_ticket_status / create_tenant 等
	ResourceType  string // approval / ticket / document / tenant 等
	ResourceID    string
	Detail        string // 可选：状态变更等补充描述
}

// AuditLogEntry 是查询返回的审计记录。
type AuditLogEntry struct {
	ID            string
	TenantID      string
	ActorID       string
	ActorUsername string
	Action        string
	ResourceType  string
	ResourceID    string
	Detail        string
	CreatedAt     time.Time
}

// AuditStore 追加并查询审计日志。审计日志只追加、不可修改或删除。
type AuditStore struct {
	client *ent.Client
}

func NewAuditStore(client *ent.Client) *AuditStore {
	return &AuditStore{client: client}
}

// Create 追加一条审计记录。
func (s *AuditStore) Create(ctx context.Context, record AuditRecord) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("audit store client is required")
	}
	id, err := newAuditID()
	if err != nil {
		return err
	}
	builder := s.client.AuditLog.Create().
		SetID(id).
		SetTenantID(record.TenantID).
		SetActorID(record.ActorID).
		SetActorUsername(record.ActorUsername).
		SetAction(record.Action).
		SetResourceType(record.ResourceType).
		SetResourceID(record.ResourceID)
	if record.Detail != "" {
		builder = builder.SetDetail(record.Detail)
	}
	if _, err := builder.Save(ctx); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// List 查询审计记录，按时间倒序，最多 limit 条。tenantID 为空表示不过滤（staff 看全部）。
func (s *AuditStore) List(ctx context.Context, tenantID string, limit int) ([]AuditLogEntry, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("audit store client is required")
	}
	query := s.client.AuditLog.Query()
	if tenantID != "" {
		query = query.Where(entauditlog.TenantIDEQ(tenantID))
	}
	rows, err := query.
		Order(ent.Desc(entauditlog.FieldCreatedAt), ent.Asc(entauditlog.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	entries := make([]AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		detail := ""
		if row.Detail != nil {
			detail = *row.Detail
		}
		entries = append(entries, AuditLogEntry{
			ID:            row.ID,
			TenantID:      row.TenantID,
			ActorID:       row.ActorID,
			ActorUsername: row.ActorUsername,
			Action:        row.Action,
			ResourceType:  row.ResourceType,
			ResourceID:    row.ResourceID,
			Detail:        detail,
			CreatedAt:     row.CreatedAt,
		})
	}
	return entries, nil
}

func newAuditID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate audit ID: %w", err)
	}
	return "audit_" + hex.EncodeToString(value[:]), nil
}
