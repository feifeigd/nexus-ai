package logic

import (
	"os"
	"strings"
	"time"

	"nexusai/services/api/internal/config"
)

// authSecret 返回 JWT 签名密钥：config 优先，环境变量 NEXUS_AUTH_SECRET 兜底。
func authSecret(cfg config.AuthConfig) string {
	if cfg.AccessSecret != "" {
		return cfg.AccessSecret
	}
	return strings.TrimSpace(os.Getenv("NEXUS_AUTH_SECRET"))
}

// authTTL 返回 token 有效期：config 优先，默认 24 小时。
func authTTL(cfg config.AuthConfig) time.Duration {
	if cfg.AccessExpire > 0 {
		return time.Duration(cfg.AccessExpire) * time.Second
	}
	return 24 * time.Hour
}
