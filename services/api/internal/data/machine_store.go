package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"nexusai/services/api/internal/data/ent"
	entmachine "nexusai/services/api/internal/data/ent/machine"
)

// MachineRecord 是一台运营中的裸机/服务器（平台级基础设施）。
type MachineRecord struct {
	ID             string
	Name           string
	PrivateAddress string
	PublicAddress  string
	Region         string
	RootPublicKey  string
	JoinedK8s      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ErrMachineNotFound 表示机器不存在。
var ErrMachineNotFound = fmt.Errorf("machine not found")

// MachineStore 持久化机器清单。
type MachineStore struct {
	client *ent.Client
}

func NewMachineStore(client *ent.Client) *MachineStore {
	return &MachineStore{client: client}
}

// Create 创建一台机器。可选字段（name/公网地址/区域/root 公钥）留空表示不设置。
func (s *MachineStore) Create(ctx context.Context, m MachineRecord) (MachineRecord, error) {
	if s == nil || s.client == nil {
		return MachineRecord{}, fmt.Errorf("machine store client is required")
	}
	id, err := newMachineID()
	if err != nil {
		return MachineRecord{}, err
	}
	builder := s.client.Machine.Create().
		SetID(id).
		SetPrivateAddress(m.PrivateAddress).
		SetJoinedK8s(m.JoinedK8s)
	applyOptional(builder, m)
	row, err := builder.Save(ctx)
	if err != nil {
		return MachineRecord{}, fmt.Errorf("create machine: %w", err)
	}
	return machineFromEnt(row), nil
}

// List 列出所有机器，按创建时间倒序。
func (s *MachineStore) List(ctx context.Context) ([]MachineRecord, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("machine store client is required")
	}
	rows, err := s.client.Machine.Query().
		Order(ent.Desc(entmachine.FieldCreatedAt), ent.Asc(entmachine.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}
	records := make([]MachineRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, machineFromEnt(row))
	}
	return records, nil
}

// GetByID 按 id 查机器。不存在时返回 ErrMachineNotFound。
func (s *MachineStore) GetByID(ctx context.Context, id string) (MachineRecord, error) {
	if s == nil || s.client == nil {
		return MachineRecord{}, fmt.Errorf("machine store client is required")
	}
	row, err := s.client.Machine.Get(ctx, id)
	if ent.IsNotFound(err) {
		return MachineRecord{}, ErrMachineNotFound
	}
	if err != nil {
		return MachineRecord{}, fmt.Errorf("get machine: %w", err)
	}
	return machineFromEnt(row), nil
}

// Update 全量更新一台机器。可选字段留空表示清空该字段。
func (s *MachineStore) Update(ctx context.Context, m MachineRecord) (MachineRecord, error) {
	if s == nil || s.client == nil {
		return MachineRecord{}, fmt.Errorf("machine store client is required")
	}
	if _, err := s.GetByID(ctx, m.ID); err != nil {
		return MachineRecord{}, err
	}
	update := s.client.Machine.UpdateOneID(m.ID).
		SetPrivateAddress(m.PrivateAddress).
		SetJoinedK8s(m.JoinedK8s)
	update = applyOptionalUpdate(update, m)
	if _, err := update.Save(ctx); err != nil {
		return MachineRecord{}, fmt.Errorf("update machine: %w", err)
	}
	return s.GetByID(ctx, m.ID)
}

// Delete 删除一台机器。不存在时返回 ErrMachineNotFound。
func (s *MachineStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("machine store client is required")
	}
	deleted, err := s.client.Machine.Delete().Where(entmachine.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}
	if deleted == 0 {
		return ErrMachineNotFound
	}
	return nil
}

// applyOptional 把可选字段设置到 Create builder。
func applyOptional(builder *ent.MachineCreate, m MachineRecord) {
	if m.Name != "" {
		builder.SetName(m.Name)
	}
	if m.PublicAddress != "" {
		builder.SetPublicAddress(m.PublicAddress)
	}
	if m.Region != "" {
		builder.SetRegion(m.Region)
	}
	if m.RootPublicKey != "" {
		builder.SetRootPublicKey(m.RootPublicKey)
	}
}

// applyOptionalUpdate 把可选字段设置（或清空）到 UpdateOne builder。
func applyOptionalUpdate(builder *ent.MachineUpdateOne, m MachineRecord) *ent.MachineUpdateOne {
	if m.Name != "" {
		builder.SetName(m.Name)
	} else {
		builder.ClearName()
	}
	if m.PublicAddress != "" {
		builder.SetPublicAddress(m.PublicAddress)
	} else {
		builder.ClearPublicAddress()
	}
	if m.Region != "" {
		builder.SetRegion(m.Region)
	} else {
		builder.ClearRegion()
	}
	if m.RootPublicKey != "" {
		builder.SetRootPublicKey(m.RootPublicKey)
	} else {
		builder.ClearRootPublicKey()
	}
	return builder
}

func machineFromEnt(row *ent.Machine) MachineRecord {
	return MachineRecord{
		ID:             row.ID,
		Name:           derefOrEmpty(row.Name),
		PrivateAddress: row.PrivateAddress,
		PublicAddress:  derefOrEmpty(row.PublicAddress),
		Region:         derefOrEmpty(row.Region),
		RootPublicKey:  derefOrEmpty(row.RootPublicKey),
		JoinedK8s:      row.JoinedK8s,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func newMachineID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate machine ID: %w", err)
	}
	return "machine_" + hex.EncodeToString(value[:]), nil
}
