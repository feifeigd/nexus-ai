package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexusai/services/api/internal/types"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	HealthHandler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var got types.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Status != "ok" {
		t.Errorf("expected status %q, got %q", "ok", got.Status)
	}
	if got.Service != "nexus-api" {
		t.Errorf("expected service %q, got %q", "nexus-api", got.Service)
	}
}
