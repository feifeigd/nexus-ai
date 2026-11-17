package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"nexusai/services/api/internal/data/ent"
	entmachine "nexusai/services/api/internal/data/ent/machine"
	"nexusai/services/api/internal/machinekey"
)

// MachineRecord 是一台运营中的裸机/服务器（平台级基础设施）。
// SSHPrivateKey 是 root 登录私钥，只写不读：仅由 Create/Update 写入、由 Exec 逻辑
// 内部读取，绝不通过 API 响应或前端列表返回给调用方。
type MachineRecord struct {
	ID                 string
	Name               string
	PrivateAddress     string
	PublicAddress      string
	Region             string
	RootPublicKey      string
	SSHPrivateKey      string
	ClearSSHPrivateKey bool
	JoinedK8s          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ErrMachineNotFound 表示机器不存在。
var ErrMachineNotFound = fmt.Errorf("machine not found")

// MachineStore 持久化机器清单。SSH 私钥只在写入前加密、内部读取后解密，
// 业务逻辑拿到的仍是明文；新写入数据库和 Ent 原始记录的私钥只保存密文。
type MachineStore struct {
	client *ent.Client
	keys   *machinekey.Box
}

func NewMachineStore(client *ent.Client, keys *machinekey.Box) *MachineStore {
	return &MachineStore{client: client, keys: keys}
}

// Create 创建一台机器。可选字段（name/公网地址/区域/root 公钥）留空表示不设置。
func (s *MachineStore) Create(ctx context.Context, m MachineRecord) (MachineRecord, error) {
	if s == nil || s.client == nil {
		return MachineRecord{}, fmt.Errorf("machine store client is required")
	}
	m, err := s.sealPrivateKey(m)
	if err != nil {
		return MachineRecord{}, err
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
	return s.openPrivateKey(machineFromEnt(row))
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
		record := machineFromEnt(row)
		// 列表不需要私钥：直接抹除，既避免无意义解密，也保证认证密钥更换后
		// 旧密文失效不会阻塞机器管理和重新录入新私钥。
		record.SSHPrivateKey = ""
		records = append(records, record)
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
	return s.openPrivateKey(machineFromEnt(row))
}

// Update 全量更新一台机器。可选字段留空表示清空该字段。
func (s *MachineStore) Update(ctx context.Context, m MachineRecord) (MachineRecord, error) {
	if s == nil || s.client == nil {
		return MachineRecord{}, fmt.Errorf("machine store client is required")
	}
	// 只检查存在性，不解密旧私钥。这样认证密钥更换后仍能重新录入新私钥。
	exists, err := s.client.Machine.Query().Where(entmachine.IDEQ(m.ID)).Exist(ctx)
	if err != nil {
		return MachineRecord{}, fmt.Errorf("get machine: %w", err)
	}
	if !exists {
		return MachineRecord{}, ErrMachineNotFound
	}
	m, err = s.sealPrivateKey(m)
	if err != nil {
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

func (s *MachineStore) sealPrivateKey(m MachineRecord) (MachineRecord, error) {
	if m.SSHPrivateKey == "" {
		return m, nil
	}
	sealed, err := s.keys.Seal(m.SSHPrivateKey)
	if err != nil {
		return MachineRecord{}, fmt.Errorf("encrypt machine SSH private key: %w", err)
	}
	m.SSHPrivateKey = sealed
	return m, nil
}

func (s *MachineStore) openPrivateKey(m MachineRecord) (MachineRecord, error) {
	// 升级前保存的历史明文继续可读；下次写入新私钥时自动转为密文。
	if m.SSHPrivateKey == "" || !machinekey.IsEncrypted(m.SSHPrivateKey) {
		return m, nil
	}
	opened, err := s.keys.Open(m.SSHPrivateKey)
	if err != nil {
		return MachineRecord{}, fmt.Errorf("decrypt machine %s SSH private key: %w", m.ID, err)
	}
	m.SSHPrivateKey = opened
	return m, nil
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
	if m.SSHPrivateKey != "" {
		builder.SetSSHPrivateKey(m.SSHPrivateKey)
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
	// 私钥只写不读：留空表示「保持不变」，而不是清空。只有显式设置
	// ClearSSHPrivateKey 才清除，避免编辑其它字段时意外抹掉已配置私钥。
	if m.ClearSSHPrivateKey {
		builder.ClearSSHPrivateKey()
	} else if m.SSHPrivateKey != "" {
		builder.SetSSHPrivateKey(m.SSHPrivateKey)
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
		SSHPrivateKey:  derefOrEmpty(row.SSHPrivateKey),
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
