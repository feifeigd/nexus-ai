// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterInput) (resp *types.LoginResponse, err error) {
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}
	if len(username) > 64 || len(req.Password) > 128 {
		return nil, errors.New("用户名或密码过长")
	}
	if l.svcCtx.Users == nil {
		return nil, errors.New("user store is not configured")
	}
	if _, err := l.svcCtx.Users.GetByUsername(l.ctx, username); err == nil {
		return nil, errors.New("用户名已存在")
	} else if !errors.Is(err, data.ErrUserNotFound) {
		return nil, err
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user, err := l.svcCtx.Users.Create(l.ctx, username, hash, "user")
	if err != nil {
		return nil, err
	}
	secret := authSecret(l.svcCtx.Config.Auth)
	if secret == "" {
		return nil, errors.New("认证密钥未配置，请设置 NEXUS_AUTH_SECRET")
	}
	token, err := auth.SignToken(secret, user.ID, user.Username, user.Role, time.Now(), authTTL(l.svcCtx.Config.Auth))
	if err != nil {
		return nil, err
	}
	return &types.LoginResponse{Token: token, Username: user.Username, Role: user.Role}, nil
}
