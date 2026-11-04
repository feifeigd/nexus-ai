package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaffOnlyMiddleware(t *testing.T) {
	m := NewStaffOnlyMiddleware()

	t.Run("staff passes", func(t *testing.T) {
		called := false
		handler := m.Handle(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		r := httptest.NewRequest(http.MethodGet, "/approvals", nil)
		r = r.WithContext(context.WithValue(r.Context(), "role", "staff"))
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != http.StatusOK || !called {
			t.Fatalf("expected staff to pass, got code=%d called=%v", w.Code, called)
		}
	})

	t.Run("user forbidden", func(t *testing.T) {
		handler := m.Handle(func(http.ResponseWriter, *http.Request) {
			t.Fatal("next should not be called for a non-staff role")
		})
		r := httptest.NewRequest(http.MethodGet, "/approvals", nil)
		r = r.WithContext(context.WithValue(r.Context(), "role", "user"))
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("missing role forbidden", func(t *testing.T) {
		handler := m.Handle(func(http.ResponseWriter, *http.Request) {
			t.Fatal("next should not be called without a role claim")
		})
		r := httptest.NewRequest(http.MethodGet, "/approvals", nil)
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}
