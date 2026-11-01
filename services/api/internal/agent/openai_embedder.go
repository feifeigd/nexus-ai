package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pgvector/pgvector-go"
)

const (
	maxEmbeddingBatchSize    = 64
	maxEmbeddingResponseSize = 8 << 20
)

type OpenAIEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

type embeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error json.RawMessage `json:"error"`
}

func NewOpenAIEmbedder(config OpenAIConfig, client *http.Client, dimensions int) (*OpenAIEmbedder, error) {
	model := strings.TrimSpace(config.EmbeddingModel)
	if model == "" {
		return nil, fmt.Errorf("NEXUS_OPENAI_EMBEDDING_MODEL is required for knowledge ingestion")
	}
	if dimensions <= 0 {
		return nil, fmt.Errorf("embedding dimensions must be positive")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	if client == nil {
		client = newHTTPClient()
	}
	return &OpenAIEmbedder{
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(config.APIKey),
		model:      model,
		dimensions: dimensions,
		client:     client,
	}, nil
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	vectors := make([]pgvector.Vector, len(texts))
	for start := 0; start < len(texts); start += maxEmbeddingBatchSize {
		end := min(start+maxEmbeddingBatchSize, len(texts))
		batch, err := e.embedBatch(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		copy(vectors[start:], batch)
	}
	return vectors, nil
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("embedding input must not be empty")
		}
	}
	payload, err := json.Marshal(embeddingRequest{Model: e.model, Input: texts, Dimensions: e.dimensions})
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embedding provider returned HTTP %d", resp.StatusCode)
	}
	var decoded embeddingResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxEmbeddingResponseSize)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Error) > 0 && string(decoded.Error) != "null" {
		return nil, fmt.Errorf("embedding provider returned an error response")
	}
	if len(decoded.Data) != len(texts) {
		return nil, fmt.Errorf("embedding provider returned %d vectors for %d inputs", len(decoded.Data), len(texts))
	}
	vectors := make([]pgvector.Vector, len(texts))
	seen := make([]bool, len(texts))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(texts) || seen[item.Index] {
			return nil, fmt.Errorf("embedding provider returned an invalid or duplicate index %d", item.Index)
		}
		if len(item.Embedding) != e.dimensions {
			return nil, fmt.Errorf("embedding provider returned %d dimensions, expected %d", len(item.Embedding), e.dimensions)
		}
		seen[item.Index] = true
		vectors[item.Index] = pgvector.NewVector(item.Embedding)
	}
	return vectors, nil
}
