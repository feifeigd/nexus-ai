package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
func (s *UserStore) Create(ctx context.Context, username, passwordHash, role string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, fmt.Errorf("user store client is required")
	}
	id, err := newUserID()
	if err != nil {
		return UserRecord{}, err
	}
	if _, err := s.client.User.Create().
		SetID(id).
		SetUsername(username).
		SetPasswordHash(passwordHash).
		SetRole(role).
		Save(ctx); err != nil {
		return UserRecord{}, fmt.Errorf("create user: %w", err)
	}
	return UserRecord{ID: id, Username: username, PasswordHash: passwordHash, Role: role}, nil
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
		CreatedAt:    row.CreatedAt,
	}
}

func newUserID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate user ID: %w", err)
	}
	return "user_" + hex.EncodeToString(value[:]), nil
}
