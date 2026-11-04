// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginInput) (resp *types.LoginResponse, err error) {
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}
	if l.svcCtx.Users == nil {
		return nil, errors.New("user store is not configured")
	}
	user, err := l.svcCtx.Users.GetByUsername(l.ctx, username)
	if errors.Is(err, data.ErrUserNotFound) {
		return nil, errors.New("用户名或密码错误")
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("用户名或密码错误")
	}
	secret := l.svcCtx.Config.Auth.Secret
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("NEXUS_AUTH_SECRET"))
	}
	if secret == "" {
		return nil, errors.New("认证密钥未配置，请设置 NEXUS_AUTH_SECRET")
	}
	token, err := auth.SignToken(secret, user.ID, user.Username, user.Role, time.Now(), 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &types.LoginResponse{Token: token, Username: user.Username, Role: user.Role}, nil
}
