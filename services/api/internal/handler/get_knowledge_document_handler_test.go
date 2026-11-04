package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
)

type knowledgeDocumentDetailStoreFunc func(ctx context.Context, id string) (data.KnowledgeDocumentDetail, error)

func (f knowledgeDocumentDetailStoreFunc) ListDocuments(context.Context) ([]data.KnowledgeDocument, error) {
	return nil, errors.New("unexpected ListDocuments call")
}

func (f knowledgeDocumentDetailStoreFunc) GetDocument(ctx context.Context, id string) (data.KnowledgeDocumentDetail, error) {
	return f(ctx, id)
}

func (f knowledgeDocumentDetailStoreFunc) DeleteDocument(context.Context, string) error {
	return errors.New("unexpected DeleteDocument call")
}

func TestGetKnowledgeDocumentHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		createdAt := time.Date(2026, time.November, 2, 4, 0, 0, 0, time.UTC)
		var requestedID string
		store := knowledgeDocumentDetailStoreFunc(func(_ context.Context, id string) (data.KnowledgeDocumentDetail, error) {
			requestedID = id
			return data.KnowledgeDocumentDetail{
				KnowledgeDocument: data.KnowledgeDocument{
					DocumentID: id,
					Title:      "运维手册",
					CreatedAt:  createdAt,
					UpdatedAt:  createdAt,
				},
				Content: "第一章：故障排查。\n第二章：恢复验证。",
			}, nil
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil)
		request := withPathID(httptest.NewRequest(http.MethodGet, "/knowledge/documents/document-1", nil), "document-1")
		response := httptest.NewRecorder()
		getKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusOK || requestedID != "document-1" {
			t.Fatalf("unexpected detail response: status=%d id=%q body=%q", response.Code, requestedID, response.Body.String())
		}
		var body struct {
			DocumentID string `json:"documentId"`
			Title      string `json:"title"`
			Content    string `json:"content"`
			CreatedAt  string `json:"createdAt"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode document detail: %v", err)
		}
		if body.DocumentID != "document-1" || body.Title != "运维手册" ||
			body.Content != "第一章：故障排查。\n第二章：恢复验证。" || body.CreatedAt != "2026-11-02T04:00:00Z" {
			t.Fatalf("unexpected document detail: %+v", body)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := knowledgeDocumentDetailStoreFunc(func(context.Context, string) (data.KnowledgeDocumentDetail, error) {
			return data.KnowledgeDocumentDetail{}, data.ErrKnowledgeDocumentNotFound
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil)
		request := withPathID(httptest.NewRequest(http.MethodGet, "/knowledge/documents/missing", nil), "missing")
		response := httptest.NewRecorder()
		getKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("expected HTTP 404, got %d: %s", response.Code, response.Body.String())
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := knowledgeDocumentDetailStoreFunc(func(context.Context, string) (data.KnowledgeDocumentDetail, error) {
			return data.KnowledgeDocumentDetail{}, errors.New("sensitive database detail")
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil)
		request := withPathID(httptest.NewRequest(http.MethodGet, "/knowledge/documents/document-1", nil), "document-1")
		response := httptest.NewRecorder()
		getKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "sensitive") {
			t.Fatalf("unexpected store failure response: status=%d body=%q", response.Code, response.Body.String())
		}
	})
}
