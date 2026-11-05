package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"nexusai/services/api/internal/data/ent"
	enttenant "nexusai/services/api/internal/data/ent/tenant"
	enttenantmember "nexusai/services/api/internal/data/ent/tenantmember"
)

// TenantRecord 是一个租户（组织或个人空间）。
type TenantRecord struct {
	ID        string
	Name      string
	OwnerID   string
	CreatedAt time.Time
}

// ErrTenantNotFound 表示租户不存在。
var ErrTenantNotFound = fmt.Errorf("tenant not found")

// TenantStore 持久化租户。
type TenantStore struct {
	client *ent.Client
}

func NewTenantStore(client *ent.Client) *TenantStore {
	return &TenantStore{client: client}
}

// Create 创建一个租户，ownerID 是所有者用户 id。
func (s *TenantStore) Create(ctx context.Context, name, ownerID string) (TenantRecord, error) {
	if s == nil || s.client == nil {
		return TenantRecord{}, fmt.Errorf("tenant store client is required")
	}
	id, err := newTenantID()
	if err != nil {
		return TenantRecord{}, err
	}
	if _, err := s.client.Tenant.Create().
		SetID(id).
		SetName(name).
		SetOwnerID(ownerID).
		Save(ctx); err != nil {
		return TenantRecord{}, fmt.Errorf("create tenant: %w", err)
	}
	return TenantRecord{ID: id, Name: name, OwnerID: ownerID}, nil
}

// GetByID 按 id 查租户。不存在时返回 ErrTenantNotFound。
func (s *TenantStore) GetByID(ctx context.Context, id string) (TenantRecord, error) {
	if s == nil || s.client == nil {
		return TenantRecord{}, fmt.Errorf("tenant store client is required")
	}
	row, err := s.client.Tenant.Get(ctx, id)
	if ent.IsNotFound(err) {
		return TenantRecord{}, ErrTenantNotFound
	}
	if err != nil {
		return TenantRecord{}, fmt.Errorf("get tenant: %w", err)
	}
	return TenantRecord{ID: row.ID, Name: row.Name, OwnerID: row.OwnerID, CreatedAt: row.CreatedAt}, nil
}

// ListByOwner 列出某用户拥有的租户，按创建时间倒序。
func (s *TenantStore) ListByOwner(ctx context.Context, ownerID string) ([]TenantRecord, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("tenant store client is required")
	}
	rows, err := s.client.Tenant.Query().
		Where(enttenant.OwnerIDEQ(ownerID)).
		Order(ent.Desc(enttenant.FieldCreatedAt), ent.Asc(enttenant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	records := make([]TenantRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, TenantRecord{ID: row.ID, Name: row.Name, OwnerID: row.OwnerID, CreatedAt: row.CreatedAt})
	}
	return records, nil
}

// AddMember 添加一条成员关系（role 为 owner 或 member）。重复添加由唯一索引拒绝。
func (s *TenantStore) AddMember(ctx context.Context, tenantID, userID, role string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("tenant store client is required")
	}
	id, err := newTenantMemberID()
	if err != nil {
		return err
	}
	if _, err := s.client.TenantMember.Create().
		SetID(id).
		SetTenantID(tenantID).
		SetUserID(userID).
		SetRole(role).
		Save(ctx); err != nil {
		return fmt.Errorf("add tenant member: %w", err)
	}
	return nil
}

// ListByMember 列出 userID 是成员的所有租户，按创建时间倒序。
func (s *TenantStore) ListByMember(ctx context.Context, userID string) ([]TenantRecord, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("tenant store client is required")
	}
	members, err := s.client.TenantMember.Query().
		Where(enttenantmember.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tenant members: %w", err)
	}
	if len(members) == 0 {
		return []TenantRecord{}, nil
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.TenantID)
	}
	rows, err := s.client.Tenant.Query().
		Where(enttenant.IDIn(ids...)).
		Order(ent.Desc(enttenant.FieldCreatedAt), ent.Asc(enttenant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tenant records: %w", err)
	}
	records := make([]TenantRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, TenantRecord{ID: row.ID, Name: row.Name, OwnerID: row.OwnerID, CreatedAt: row.CreatedAt})
	}
	return records, nil
}

// MemberRole 返回 userID 在 tenantID 里的角色；不是成员时返回空字符串。
func (s *TenantStore) MemberRole(ctx context.Context, tenantID, userID string) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("tenant store client is required")
	}
	row, err := s.client.TenantMember.Query().
		Where(enttenantmember.TenantIDEQ(tenantID), enttenantmember.UserIDEQ(userID)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get tenant member: %w", err)
	}
	return row.Role, nil
}

func newTenantMemberID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate tenant member ID: %w", err)
	}
	return "tm_" + hex.EncodeToString(value[:]), nil
}
