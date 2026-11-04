package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims 是登录令牌的声明。
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// SignToken 签发一个带过期时间的 JWT。
func SignToken(secret string, userID, username, role, tenantID string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并校验令牌。
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// TenantIDFromContext 从请求 context 取 JWT 中间件注入的 tenant_id claim。
// 兼容旧 token（无 tenant_id claim）：每用户一个租户，回退到用户 id（uid）。
func TenantIDFromContext(ctx context.Context) string {
	tenantID, _ := ctx.Value("tenant_id").(string)
	if tenantID == "" {
		tenantID, _ = ctx.Value("uid").(string)
	}
	return tenantID
}
