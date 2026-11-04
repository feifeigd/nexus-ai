package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// seedAdminUser 在启动时确保存在一个 staff 账号。密码来自环境变量
// NEXUS_ADMIN_PASSWORD，未设置时跳过（不创建默认账号）。已存在时幂等跳过。
func seedAdminUser(ctx context.Context, svcCtx *svc.ServiceContext) error {
	password := strings.TrimSpace(os.Getenv("NEXUS_ADMIN_PASSWORD"))
	if password == "" {
		return nil
	}
	if svcCtx.Users == nil {
		return errors.New("user store is not configured for seeding")
	}
	const username = "admin"
	if _, err := svcCtx.Users.GetByUsername(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, data.ErrUserNotFound) {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := svcCtx.Users.Create(ctx, username, hash, "staff", ""); err != nil {
		return err
	}
	logx.Infow("seeded admin staff user", logx.Field("username", username))
	return nil
}
