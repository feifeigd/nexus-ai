package knowledge

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"
)

type knowledgeSearchStore struct {
	matches   []data.KnowledgeMatch
	embedding pgvector.Vector
	limit     int
}

func (s *knowledgeSearchStore) CreateDocument(context.Context, string, string, []data.KnowledgeChunkInput) (string, error) {
	return "", nil
}

func (s *knowledgeSearchStore) SearchSimilar(_ context.Context, embedding pgvector.Vector, limit int) ([]data.KnowledgeMatch, error) {
	s.embedding = embedding
	s.limit = limit
	return s.matches, nil
}

func TestServiceSearchEmbedsQueryAndReturnsMatches(t *testing.T) {
	store := &knowledgeSearchStore{matches: []data.KnowledgeMatch{{
		DocumentID: "document-1",
		ChunkID:    "chunk-1",
		Content:    "蓝雀告警需要先检查 API 服务。",
		Distance:   0.125,
	}}}
	var embeddedTexts []string
	service, err := NewService(embedderFunc(func(_ context.Context, texts []string) ([]pgvector.Vector, error) {
		embeddedTexts = append([]string(nil), texts...)
		return []pgvector.Vector{knowledgeServiceTestVector(1)}, nil
	}), store, 5, 1)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}
	matches, err := service.Search(context.Background(), " 蓝雀告警 ", 0)
	if err != nil {
		t.Fatalf("search knowledge: %v", err)
	}
	if !reflect.DeepEqual(embeddedTexts, []string{"蓝雀告警"}) {
		t.Fatalf("unexpected embedded query: %v", embeddedTexts)
	}
	if store.limit != searchRecallLimit(DefaultSearchLimit) || store.embedding.Slice()[0] != 1 {
		t.Fatalf("unexpected vector search input: limit=%d embedding=%v", store.limit, store.embedding)
	}
	if !reflect.DeepEqual(matches, store.matches) {
		t.Fatalf("unexpected matches: got %+v want %+v", matches, store.matches)
	}
	for _, test := range []struct {
		query string
		limit int
	}{
		{query: " ", limit: 1},
		{query: strings.Repeat("问", MaxSearchQueryLength+1), limit: 1},
		{query: "有效问题", limit: -1},
		{query: "有效问题", limit: MaxSearchLimit + 1},
	} {
		if _, err := service.Search(context.Background(), test.query, test.limit); err == nil {
			t.Fatalf("expected invalid search input to fail: %+v", test)
		}
	}
}

func TestSearchKnowledgeToolReturnsStructuredMatches(t *testing.T) {
	store := &knowledgeSearchStore{matches: []data.KnowledgeMatch{{
		DocumentID: "document-1",
		ChunkID:    "chunk-1",
		Ordinal:    2,
		Content:    "先检查 API 服务状态。",
		Distance:   0.25,
	}}}
	service, err := NewService(embedderFunc(func(context.Context, []string) ([]pgvector.Vector, error) {
		return []pgvector.Vector{knowledgeServiceTestVector(1)}, nil
	}), store, 5, 1)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}
	tool := service.Tool()
	if tool.Name != searchKnowledgeToolName || tool.Execute == nil || tool.Propose != nil {
		t.Fatalf("unexpected read-only tool definition: %+v", tool)
	}
	content, err := tool.Execute(context.Background(), agent.ToolProposal{
		Name:      searchKnowledgeToolName,
		Arguments: json.RawMessage(`{"query":"蓝雀告警","limit":3}`),
	})
	if err != nil {
		t.Fatalf("execute knowledge search: %v", err)
	}
	var result searchKnowledgeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if store.limit != searchRecallLimit(3) || len(result.Matches) != 1 || result.Matches[0].DocumentID != "document-1" ||
		result.Matches[0].Content != "先检查 API 服务状态。" {
		t.Fatalf("unexpected structured tool result: %s", content)
	}
	if _, err := tool.Execute(context.Background(), agent.ToolProposal{
		Name:      searchKnowledgeToolName,
		Arguments: json.RawMessage(`{"query":"蓝雀告警","limit":11}`),
	}); err == nil {
		t.Fatal("expected the server-side limit validation to reject the tool call")
	}
}

func TestSearchFiltersByDistanceThreshold(t *testing.T) {
	for _, test := range []struct {
		name    string
		matches []data.KnowledgeMatch
		want    []string
	}{
		{
			name: "keeps chunks at or below the threshold",
			matches: []data.KnowledgeMatch{
				{ChunkID: "at-threshold", Distance: SearchDistanceThreshold},
				{ChunkID: "below", Distance: 0.1},
				{ChunkID: "above", Distance: 0.5000001},
				{ChunkID: "far", Distance: 1.2},
			},
			want: []string{"below", "at-threshold"},
		},
		{
			name: "empty result when every chunk exceeds the threshold",
			matches: []data.KnowledgeMatch{
				{ChunkID: "far", Distance: 0.9},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &knowledgeSearchStore{matches: test.matches}
			service, err := NewService(embedderFunc(func(context.Context, []string) ([]pgvector.Vector, error) {
				return []pgvector.Vector{knowledgeServiceTestVector(1)}, nil
			}), store, 5, 1)
			if err != nil {
				t.Fatalf("create knowledge service: %v", err)
			}
			matches, err := service.Search(context.Background(), "阈值过滤", len(test.matches))
			if err != nil {
				t.Fatalf("search knowledge: %v", err)
			}
			var got []string
			for _, match := range matches {
				got = append(got, match.ChunkID)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("unexpected threshold filtering: got %v want %v", got, test.want)
			}
		})
	}
}

func TestValidateSearchAnswer(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	fabricated := "document_" + strings.Repeat("f", 32)
	toolResult := `{"matches":[{"documentId":"` + valid + `"}]}`

	for _, test := range []struct {
		name       string
		answer     string
		toolResult string
		wantErr    bool
		wantListed string
		wantAbsent string
	}{
		{
			name:       "clean answer citing only retrieved source",
			answer:     "根据 " + valid + " 处理。",
			toolResult: toolResult,
		},
		{
			name:       "empty answer",
			answer:     "",
			toolResult: toolResult,
		},
		{
			name:       "fabricated citation is reported",
			answer:     "根据 " + fabricated + " 处理。",
			toolResult: toolResult,
			wantListed: fabricated,
		},
		{
			name:       "mixed citations list only the fabricated one",
			answer:     "参见 " + valid + " 与 " + fabricated + "。",
			toolResult: toolResult,
			wantListed: fabricated,
			wantAbsent: valid,
		},
		{
			name:       "duplicate fabricated citation listed once",
			answer:     fabricated + " 和 " + fabricated,
			toolResult: toolResult,
			wantListed: fabricated,
		},
		{
			name:       "malformed tool result",
			answer:     "任意回答",
			toolResult: `not-json`,
			wantErr:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			note, err := validateSearchAnswer(context.Background(), nil, test.answer, test.toolResult)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got note %q", note)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate answer: %v", err)
			}
			if test.wantListed == "" {
				if note != "" {
					t.Fatalf("expected clean answer, got note %q", note)
				}
				return
			}
			if !strings.Contains(note, test.wantListed) {
				t.Fatalf("expected note to list %q, got %q", test.wantListed, note)
			}
			if test.wantAbsent != "" && strings.Contains(note, test.wantAbsent) {
				t.Fatalf("note should not list the valid source %q, got %q", test.wantAbsent, note)
			}
		})
	}
}
