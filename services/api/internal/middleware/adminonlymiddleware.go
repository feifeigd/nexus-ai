// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"net/http"

	"nexusai/services/api/internal/auth"
)

type AdminOnlyMiddleware struct {
}

func NewAdminOnlyMiddleware() *AdminOnlyMiddleware {
	return &AdminOnlyMiddleware{}
}

func (m *AdminOnlyMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return RequireRole(auth.IsAdmin)(next)
}
