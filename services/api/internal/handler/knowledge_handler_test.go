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
	"nexusai/services/api/internal/data/ent/chunk"
	entschema "nexusai/services/api/internal/data/ent/schema"
	"nexusai/services/api/internal/knowledge"
	"nexusai/services/api/internal/svc"

	"entgo.io/ent/dialect"
)

type knowledgeServiceFunc func(ctx context.Context, title, content, tenantID string) (string, error)

func (f knowledgeServiceFunc) IngestDocument(ctx context.Context, title, content, tenantID string) (string, error) {
	return f(ctx, title, content, tenantID)
}

func TestIngestDocumentHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var title, content string
		service := knowledgeServiceFunc(func(_ context.Context, gotTitle, gotContent, gotTenantID string) (string, error) {
			title, content = gotTitle, gotContent
			return "document-1", nil
		})
		ctx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, service, nil, nil, nil, nil, nil)
		request := httptest.NewRequest(http.MethodPost, "/knowledge/documents", strings.NewReader(`{"title":"运维手册","content":"故障排查"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		ingestDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusOK || title != "运维手册" || content != "故障排查" {
			t.Fatalf("unexpected ingestion response or input: status=%d title=%q content=%q", response.Code, title, content)
		}
		var body struct {
			DocumentID string `json:"documentId"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.DocumentID != "document-1" {
			t.Fatalf("unexpected ingestion body: %+v err=%v", body, err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		ctx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, knowledgeServiceFunc(func(context.Context, string, string, string) (string, error) {
			t.Fatal("invalid input must not reach the knowledge service")
			return "", nil
		}), nil, nil, nil, nil, nil)
		request := httptest.NewRequest(http.MethodPost, "/knowledge/documents", strings.NewReader(`{"title":"","content":"故障排查"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		ingestDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400, got %d", response.Code)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		ctx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, nil, nil, nil, nil, nil, nil)
		request := httptest.NewRequest(http.MethodPost, "/knowledge/documents", strings.NewReader(`{"title":"运维手册","content":"故障排查"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		ingestDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", response.Code)
		}
	})

	t.Run("service failure", func(t *testing.T) {
		service := knowledgeServiceFunc(func(context.Context, string, string, string) (string, error) {
			return "", errors.New("sensitive provider detail")
		})
		ctx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, service, nil, nil, nil, nil, nil)
		request := httptest.NewRequest(http.MethodPost, "/knowledge/documents", strings.NewReader(`{"title":"运维手册","content":"故障排查"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		ingestDocumentHandler(ctx).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "sensitive") {
			t.Fatalf("unexpected service failure response: status=%d body=%q", response.Code, response.Body.String())
		}
	})
}

func TestIngestDocumentHandlerPersistsEmbeddedDocument(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("unexpected embedding path: %s", r.URL.Path)
		}
		var request struct {
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		if request.Dimensions != entschema.EmbeddingDimensions || len(request.Input) == 0 {
			t.Errorf("unexpected embedding request: %+v", request)
		}
		data := make([]map[string]any, len(request.Input))
		for index := range request.Input {
			values := make([]float32, entschema.EmbeddingDimensions)
			values[0] = 1
			data[index] = map[string]any{"index": index, "embedding": values}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer provider.Close()
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:knowledge_handler_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()
	embedder, err := agent.NewOpenAIEmbedder(agent.OpenAIConfig{
		BaseURL: provider.URL,
		Model:   "embedding-model",
	}, provider.Client(), entschema.EmbeddingDimensions)
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}
	service, err := knowledge.NewService(embedder, data.NewKnowledgeStore(client, dialect.SQLite), 0, 0)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}
	ctx := svc.NewServiceContext(&config.Config{Auth: config.AuthConfig{AccessSecret: "test-secret-123"}}, agent.MockRunner{}, nil, service, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/knowledge/documents", strings.NewReader(`{"title":"运维手册","content":"故障排查"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	ingestDocumentHandler(ctx).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		DocumentID string `json:"documentId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.DocumentID == "" {
		t.Fatalf("unexpected ingestion body: %+v err=%v", body, err)
	}
	chunks, err := client.Chunk.Query().Where(chunk.DocumentID(body.DocumentID)).All(context.Background())
	if err != nil || len(chunks) != 1 || chunks[0].Embedding == nil {
		t.Fatalf("embedded document was not persisted: chunks=%+v err=%v", chunks, err)
	}
}
