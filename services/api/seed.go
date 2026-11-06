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

// seedStaffUsers 在启动时确保存在 admin/auditor/readonly 三个内部账号。密码分别来自
// NEXUS_ADMIN_PASSWORD / NEXUS_AUDITOR_PASSWORD / NEXUS_READONLY_PASSWORD；未设置则跳过，
// 已存在则幂等跳过。
func seedStaffUsers(ctx context.Context, svcCtx *svc.ServiceContext) error {
	if svcCtx.Users == nil {
		return errors.New("user store is not configured for seeding")
	}
	for _, spec := range []struct {
		username string
		role     string
		envKey   string
	}{
		{username: "admin", role: auth.RoleAdmin, envKey: "NEXUS_ADMIN_PASSWORD"},
		{username: "auditor", role: auth.RoleAuditor, envKey: "NEXUS_AUDITOR_PASSWORD"},
		{username: "readonly", role: auth.RoleReadonly, envKey: "NEXUS_READONLY_PASSWORD"},
	} {
		if err := seedStaffUser(ctx, svcCtx, spec.username, spec.role, spec.envKey); err != nil {
			return err
		}
	}
	return nil
}

func seedStaffUser(ctx context.Context, svcCtx *svc.ServiceContext, username, role, envKey string) error {
	password := strings.TrimSpace(os.Getenv(envKey))
	if password == "" {
		return nil
	}
	if _, err := svcCtx.Users.GetByUsername(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, data.ErrUserNotFound) {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := svcCtx.Users.Create(ctx, username, hash, role, ""); err != nil {
		return err
	}
	logx.Infow("seeded staff user", logx.Field("username", username), logx.Field("role", role))
	return nil
}
