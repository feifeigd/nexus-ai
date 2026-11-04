package knowledge

import (
	"context"
	"math"
	"testing"

	"github.com/pgvector/pgvector-go"

	"nexusai/services/api/internal/data"
)

func TestBM25ScoresRewardsTermFrequency(t *testing.T) {
	// 两个分块长度相同，含查询词元次数更多的分块应得分更高。
	frequent := data.KnowledgeMatch{ChunkID: "frequent", Content: "蓝雀蓝雀", Distance: 0.40}
	single := data.KnowledgeMatch{ChunkID: "single", Content: "蓝雀无关", Distance: 0.30}
	scores := bm25Scores("蓝雀", []data.KnowledgeMatch{frequent, single})
	if scores[0] <= scores[1] {
		t.Fatalf("expected higher term frequency to score higher, got %v", scores)
	}
}

func TestBM25ScoresRewardsRareTerms(t *testing.T) {
	// 查询同时含常见词元「蓝雀」与罕见词元「告警」；含罕见词元的分块应高于只含常见词元的分块。
	commonOnly := data.KnowledgeMatch{ChunkID: "common", Content: "蓝雀"}
	rareHit := data.KnowledgeMatch{ChunkID: "rare", Content: "告警"}
	common1 := data.KnowledgeMatch{ChunkID: "common1", Content: "蓝雀"}
	common2 := data.KnowledgeMatch{ChunkID: "common2", Content: "蓝雀"}
	scores := bm25Scores("蓝雀告警", []data.KnowledgeMatch{commonOnly, rareHit, common1, common2})
	if scores[1] <= scores[0] {
		t.Fatalf("expected rare-term hit to outrank common-term-only hit, got %v", scores)
	}
}

func TestBM25ScoresNormalizesTopScoreToOne(t *testing.T) {
	matches := []data.KnowledgeMatch{
		{ChunkID: "a", Content: "蓝雀"},
		{ChunkID: "rare", Content: "告警"},
		{ChunkID: "b", Content: "蓝雀"},
	}
	scores := bm25Scores("蓝雀告警", matches)
	if len(scores) != len(matches) {
		t.Fatalf("expected %d scores, got %d", len(matches), len(scores))
	}
	if math.Abs(scores[1]-1.0) > 1e-9 {
		t.Fatalf("expected top score to normalize to 1.0, got %v", scores)
	}
	for _, score := range scores {
		if score < 0 || score > 1 {
			t.Fatalf("expected scores in [0,1], got %v", scores)
		}
	}
}

func TestBM25ScoresEmptyQueryOrPool(t *testing.T) {
	if got := bm25Scores("蓝雀", nil); got != nil {
		t.Fatalf("expected nil for empty pool, got %v", got)
	}
	matches := []data.KnowledgeMatch{{ChunkID: "a", Content: "蓝雀"}, {ChunkID: "b", Content: "蓝雀"}}
	for _, query := range []string{"", "。", "蓝"} {
		got := bm25Scores(query, matches)
		if len(got) != len(matches) {
			t.Fatalf("expected %d scores for query %q, got %d", len(matches), query, len(got))
		}
		for _, score := range got {
			if score != 0 {
				t.Fatalf("expected zero scores for query %q, got %v", query, got)
			}
		}
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

func TestSearchReranksByBM25KeywordScore(t *testing.T) {
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
	matches, err := service.Search(context.Background(), "蓝雀告警", 1, "")
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
