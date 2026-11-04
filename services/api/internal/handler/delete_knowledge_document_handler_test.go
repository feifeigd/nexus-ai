package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
)

type knowledgeDocumentDeleteStoreFunc func(ctx context.Context, id string) error

func (f knowledgeDocumentDeleteStoreFunc) ListDocuments(context.Context) ([]data.KnowledgeDocument, error) {
	return nil, errors.New("unexpected ListDocuments call")
}

func (f knowledgeDocumentDeleteStoreFunc) GetDocument(context.Context, string) (data.KnowledgeDocumentDetail, error) {
	return data.KnowledgeDocumentDetail{}, errors.New("unexpected GetDocument call")
}

func (f knowledgeDocumentDeleteStoreFunc) DeleteDocument(ctx context.Context, id string) error {
	return f(ctx, id)
}

func TestDeleteKnowledgeDocumentHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var requestedID string
		store := knowledgeDocumentDeleteStoreFunc(func(_ context.Context, id string) error {
			requestedID = id
			return nil
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil, nil)
		request := withPathID(httptest.NewRequest(http.MethodDelete, "/knowledge/documents/document-1", nil), "document-1")
		response := httptest.NewRecorder()
		deleteKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusOK || requestedID != "document-1" {
			t.Fatalf("unexpected deletion response: status=%d id=%q body=%q", response.Code, requestedID, response.Body.String())
		}
		var body struct {
			DocumentID string `json:"documentId"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode deletion response: %v", err)
		}
		if body.DocumentID != "document-1" {
			t.Fatalf("unexpected deleted document ID: %q", body.DocumentID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := knowledgeDocumentDeleteStoreFunc(func(context.Context, string) error {
			return data.ErrKnowledgeDocumentNotFound
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil, nil)
		request := withPathID(httptest.NewRequest(http.MethodDelete, "/knowledge/documents/missing", nil), "missing")
		response := httptest.NewRecorder()
		deleteKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("expected HTTP 404, got %d: %s", response.Code, response.Body.String())
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := knowledgeDocumentDeleteStoreFunc(func(context.Context, string) error {
			return errors.New("sensitive database detail")
		})
		ctx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, store, nil, nil)
		request := withPathID(httptest.NewRequest(http.MethodDelete, "/knowledge/documents/document-1", nil), "document-1")
		response := httptest.NewRecorder()
		deleteKnowledgeDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "sensitive") {
			t.Fatalf("unexpected store failure response: status=%d body=%q", response.Code, response.Body.String())
		}
	})
}
