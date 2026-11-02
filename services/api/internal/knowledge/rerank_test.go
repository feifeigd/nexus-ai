package knowledge

import (
	"context"
	"testing"

	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data"
)

func TestKeywordCoverage(t *testing.T) {
	for _, test := range []struct {
		name    string
		query   string
		content string
		want    float64
	}{
		{name: "empty query", query: "", content: "蓝雀告警", want: 0},
		{name: "punctuation only query", query: "，。？！", content: "蓝雀告警", want: 0},
		{name: "full coverage", query: "蓝雀告警", content: "蓝雀告警需要检查 API", want: 1},
		{name: "partial coverage", query: "蓝雀告警", content: "蓝雀", want: 0.5},
		{name: "no overlap", query: "蓝雀告警", content: "完全无关", want: 0},
		{name: "duplicate query runes counted once", query: "蓝蓝蓝", content: "蓝", want: 1},
		{name: "ignores punctuation and whitespace", query: "蓝雀，告警 ？", content: "蓝雀告警", want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := keywordCoverage(test.query, test.content); got != test.want {
				t.Fatalf("keywordCoverage(%q, %q) = %v, want %v", test.query, test.content, got, test.want)
			}
		})
	}
}

func TestSearchRecallLimit(t *testing.T) {
	if got := searchRecallLimit(1); got != SearchRecallFactor {
		t.Fatalf("searchRecallLimit(1) = %d, want %d", got, SearchRecallFactor)
	}
	if got := searchRecallLimit(DefaultSearchLimit); got != DefaultSearchLimit*SearchRecallFactor {
		t.Fatalf("searchRecallLimit(%d) = %d, want %d", DefaultSearchLimit, got, DefaultSearchLimit*SearchRecallFactor)
	}
	if got := searchRecallLimit(13); got != data.MaxKnowledgeSearchLimit {
		t.Fatalf("searchRecallLimit(13) = %d, want capped at %d", got, data.MaxKnowledgeSearchLimit)
	}
}

func TestRerankPromotesKeywordMatch(t *testing.T) {
	semantic := data.KnowledgeMatch{ChunkID: "semantic", Content: "完全无关内容", Distance: 0.30, Ordinal: 0}
	keyword := data.KnowledgeMatch{ChunkID: "keyword", Content: "蓝雀告警排查步骤", Distance: 0.40, Ordinal: 1}

	got := rerank("蓝雀告警", []data.KnowledgeMatch{semantic, keyword}, 2)
	if len(got) != 2 || got[0].ChunkID != "keyword" || got[1].ChunkID != "semantic" {
		t.Fatalf("expected keyword match to rank first, got %+v", got)
	}
}

func TestRerankTruncatesToLimit(t *testing.T) {
	matches := []data.KnowledgeMatch{
		{ChunkID: "a", Content: "蓝", Distance: 0.20, Ordinal: 0},
		{ChunkID: "b", Content: "蓝", Distance: 0.25, Ordinal: 1},
		{ChunkID: "c", Content: "蓝", Distance: 0.30, Ordinal: 2},
	}
	got := rerank("蓝", matches, 2)
	if len(got) != 2 || got[0].ChunkID != "a" || got[1].ChunkID != "b" {
		t.Fatalf("expected top 2 by score, got %+v", got)
	}
}

func TestRerankTieBreaksByDistanceThenOrdinal(t *testing.T) {
	later := data.KnowledgeMatch{ChunkID: "later", Content: "无关", Distance: 0.30, Ordinal: 1}
	earlier := data.KnowledgeMatch{ChunkID: "earlier", Content: "无关", Distance: 0.30, Ordinal: 0}

	got := rerank("蓝雀告警", []data.KnowledgeMatch{later, earlier}, 2)
	if len(got) != 2 || got[0].ChunkID != "earlier" || got[1].ChunkID != "later" {
		t.Fatalf("expected equal scores to break by ordinal, got %+v", got)
	}
}

func TestRerankEmptyOrInvalidInput(t *testing.T) {
	if got := rerank("蓝雀告警", nil, 2); got != nil {
		t.Fatalf("expected nil for empty matches, got %+v", got)
	}
	if got := rerank("蓝雀告警", []data.KnowledgeMatch{{ChunkID: "a", Content: "蓝"}}, 0); got != nil {
		t.Fatalf("expected nil for invalid limit, got %+v", got)
	}
}

func TestSearchReranksByKeywordCoverage(t *testing.T) {
	store := &knowledgeSearchStore{matches: []data.KnowledgeMatch{
		{ChunkID: "semantic", Content: "完全无关的内容段落", Distance: 0.30, Ordinal: 0},
		{ChunkID: "keyword", Content: "蓝雀告警需要先检查 API 服务。", Distance: 0.40, Ordinal: 1},
	}}
	service, err := NewService(embedderFunc(func(context.Context, []string) ([]pgvector.Vector, error) {
		return []pgvector.Vector{knowledgeServiceTestVector(1)}, nil
	}), store, 5, 1)
	if err != nil {
		t.Fatalf("create knowledge service: %v", err)
	}
	matches, err := service.Search(context.Background(), "蓝雀告警", 1)
	if err != nil {
		t.Fatalf("search knowledge: %v", err)
	}
	if store.limit != searchRecallLimit(1) {
		t.Fatalf("expected recall limit %d, got %d", searchRecallLimit(1), store.limit)
	}
	if len(matches) != 1 || matches[0].ChunkID != "keyword" {
		t.Fatalf("expected keyword-matching chunk to win top-1 after re-ranking, got %+v", matches)
	}
}
