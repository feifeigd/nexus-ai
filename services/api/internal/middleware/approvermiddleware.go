// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"net/http"

	"nexusai/services/api/internal/auth"
)

type ApproverMiddleware struct {
}

func NewApproverMiddleware() *ApproverMiddleware {
	return &ApproverMiddleware{}
}

func (m *ApproverMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return RequireRole(auth.CanApprove)(next)
}
