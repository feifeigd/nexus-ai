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

type knowledgeDocumentStoreFunc func(ctx context.Context) ([]data.KnowledgeDocument, error)

func (f knowledgeDocumentStoreFunc) ListDocuments(ctx context.Context) ([]data.KnowledgeDocument, error) {
	return f(ctx)
}

func (f knowledgeDocumentStoreFunc) GetDocument(context.Context, string) (data.KnowledgeDocumentDetail, error) {
	return data.KnowledgeDocumentDetail{}, errors.New("unexpected GetDocument call")
}

func (f knowledgeDocumentStoreFunc) DeleteDocument(context.Context, string) error {
	return errors.New("unexpected DeleteDocument call")
}

func TestListKnowledgeDocumentsHandler(t *testing.T) {
	t.Run("success without an embedding service", func(t *testing.T) {
		createdAt := time.Date(2026, time.November, 2, 4, 0, 0, 0, time.UTC)
		store := knowledgeDocumentStoreFunc(func(context.Context) ([]data.KnowledgeDocument, error) {
			return []data.KnowledgeDocument{{
				DocumentID: "document-1",
				Title:      "运维手册",
				CreatedAt:  createdAt,
				UpdatedAt:  createdAt.Add(time.Minute),
			}}, nil
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil)
		request := httptest.NewRequest(http.MethodGet, "/knowledge/documents", nil)
		response := httptest.NewRecorder()
		listKnowledgeDocumentsHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d: %s", response.Code, response.Body.String())
		}
		var body struct {
			Documents []struct {
				DocumentID string `json:"documentId"`
				Title      string `json:"title"`
				CreatedAt  string `json:"createdAt"`
				UpdatedAt  string `json:"updatedAt"`
			} `json:"documents"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode document list: %v", err)
		}
		if len(body.Documents) != 1 || body.Documents[0].DocumentID != "document-1" ||
			body.Documents[0].Title != "运维手册" || body.Documents[0].CreatedAt != "2026-11-02T04:00:00Z" ||
			body.Documents[0].UpdatedAt != "2026-11-02T04:01:00Z" {
			t.Fatalf("unexpected document list: %+v", body.Documents)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := knowledgeDocumentStoreFunc(func(context.Context) ([]data.KnowledgeDocument, error) {
			return nil, errors.New("sensitive database detail")
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil)
		request := httptest.NewRequest(http.MethodGet, "/knowledge/documents", nil)
		response := httptest.NewRecorder()
		listKnowledgeDocumentsHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "sensitive") {
			t.Fatalf("unexpected store failure response: status=%d body=%q", response.Code, response.Body.String())
		}
	})
}
