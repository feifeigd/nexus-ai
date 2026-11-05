package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"nexusai/services/api/internal/data/ent"
	entuser "nexusai/services/api/internal/data/ent/user"
)

// UserRecord 是一个登录账号。
type UserRecord struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
	TenantID     string
	CreatedAt    time.Time
}

// ErrUserNotFound 表示用户名或 ID 不存在。
var ErrUserNotFound = fmt.Errorf("user not found")

// UserStore 持久化登录账号。role 区分 staff（登录 Admin）与 user（登录 Web）。
type UserStore struct {
	client *ent.Client
}

func NewUserStore(client *ent.Client) *UserStore {
	return &UserStore{client: client}
}

// Create 创建用户，密码为已哈希的 passwordHash。username 已存在时返回数据库唯一约束错误。
// tenantID 为空时自动为这个用户创建一个「个人租户」（owner 是自己）。
func (s *UserStore) Create(ctx context.Context, username, passwordHash, role, tenantID string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	id, err := newUserID()
	if err != nil {
		return UserRecord{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return UserRecord{}, fmt.Errorf("begin user creation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if tenantID == "" {
		tenantID, err = createPersonalTenant(ctx, tx.Client(), username, id)
		if err != nil {
			return UserRecord{}, err
		}
	}
	if _, err := tx.Client().User.Create().
		SetID(id).
		SetUsername(username).
		SetPasswordHash(passwordHash).
		SetRole(role).
		SetTenantID(tenantID).
		Save(ctx); err != nil {
		return UserRecord{}, fmt.Errorf("create user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return UserRecord{}, fmt.Errorf("commit user creation: %w", err)
	}
	return UserRecord{ID: id, Username: username, PasswordHash: passwordHash, Role: role, TenantID: tenantID}, nil
}

// CreateOAuth 创建通过三方登录（无密码）的用户。provider/subject 标识一个 OAuth 身份
// （如 provider=github、subject=该 GitHub 用户的 id），跨平台 subject 可重复。
// 同时为该用户创建个人租户（owner 是自己）。
func (s *UserStore) CreateOAuth(ctx context.Context, provider, subject, username, role string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	id, err := newUserID()
	if err != nil {
		return UserRecord{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return UserRecord{}, fmt.Errorf("begin oauth user creation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tenantID, err := createPersonalTenant(ctx, tx.Client(), username, id)
	if err != nil {
		return UserRecord{}, err
	}
	if _, err := tx.Client().User.Create().
		SetID(id).
		SetUsername(username).
		SetPasswordHash("").
		SetOauthProvider(provider).
		SetOauthSubject(subject).
		SetRole(role).
		SetTenantID(tenantID).
		Save(ctx); err != nil {
		return UserRecord{}, fmt.Errorf("create oauth user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return UserRecord{}, fmt.Errorf("commit oauth user creation: %w", err)
	}
	return UserRecord{ID: id, Username: username, Role: role, TenantID: tenantID}, nil
}

// GetByUsername 按用户名查。不存在时返回 ErrUserNotFound。
func (s *UserStore) GetByUsername(ctx context.Context, username string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	row, err := s.client.User.Query().Where(entuser.UsernameEQ(username)).Only(ctx)
	if ent.IsNotFound(err) {
		return UserRecord{}, ErrUserNotFound
	}
	if err != nil {
		return UserRecord{}, fmt.Errorf("get user by username: %w", err)
	}
	return toUserRecord(row), nil
}

// GetByOAuthSubject 按 provider + subject 查三方登录用户。不存在时返回 ErrUserNotFound。
func (s *UserStore) GetByOAuthSubject(ctx context.Context, provider, subject string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	row, err := s.client.User.Query().
		Where(entuser.OauthProviderEQ(provider), entuser.OauthSubjectEQ(subject)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return UserRecord{}, ErrUserNotFound
	}
	if err != nil {
		return UserRecord{}, fmt.Errorf("get user by oauth subject: %w", err)
	}
	return toUserRecord(row), nil
}

// GetByID 按 ID 查。不存在时返回 ErrUserNotFound。
func (s *UserStore) GetByID(ctx context.Context, id string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	row, err := s.client.User.Get(ctx, id)
	if ent.IsNotFound(err) {
		return UserRecord{}, ErrUserNotFound
	}
	if err != nil {
		return UserRecord{}, fmt.Errorf("get user by id: %w", err)
	}
	return toUserRecord(row), nil
}

func toUserRecord(row *ent.User) UserRecord {
	return UserRecord{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		TenantID:     row.TenantID,
		CreatedAt:    row.CreatedAt,
	}
}

// UpdateTenantID 把用户的 tenant_id 切到指定租户（如加入组织）。用户不存在时返回
// ErrUserNotFound。
func (s *UserStore) UpdateTenantID(ctx context.Context, userID, tenantID string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	if _, err := s.GetByID(ctx, userID); err != nil {
		return UserRecord{}, err
	}
	if _, err := s.client.User.UpdateOneID(userID).SetTenantID(tenantID).Save(ctx); err != nil {
		return UserRecord{}, fmt.Errorf("update user tenant: %w", err)
	}
	return s.GetByID(ctx, userID)
}

func newUserID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate user ID: %w", err)
	}
	return "user_" + hex.EncodeToString(value[:]), nil
}

func newTenantID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate tenant ID: %w", err)
	}
	return "tenant_" + hex.EncodeToString(value[:]), nil
}

// createPersonalTenant 在给定 client（可为事务 client）上为 ownerID 用户创建一个
// 个人租户，返回租户 id。个人租户的名称用用户名拼接，便于列表展示。
func createPersonalTenant(ctx context.Context, client *ent.Client, username, ownerID string) (string, error) {
	id, err := newTenantID()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(username) + " 的个人空间"
	if _, err := client.Tenant.Create().
		SetID(id).
		SetName(name).
		SetOwnerID(ownerID).
		Save(ctx); err != nil {
		return "", fmt.Errorf("create personal tenant: %w", err)
	}
	return id, nil
}
