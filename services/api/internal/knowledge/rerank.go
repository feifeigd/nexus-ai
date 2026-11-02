package knowledge

import (
	"sort"
	"unicode"

	"nexusai/services/api/internal/data"
)

const (
	// SearchRecallFactor 是召回时放大请求 limit 的倍数，为重排序提供候选池。
	SearchRecallFactor = 4

	// KeywordRelevanceWeight 控制关键词覆盖率在重排序分数中能抵消的余弦距离量：
	// 分数越低越靠前，每 1 单位覆盖率可抵消这么多单位的余弦距离。
	KeywordRelevanceWeight = 0.15
)

// searchRecallLimit 返回重排序前从 pgvector 召回的候选数量，即 min(limit×4, 50)，
// 不会超过存储层的最大召回上限。
func searchRecallLimit(limit int) int {
	if recall := limit * SearchRecallFactor; recall < data.MaxKnowledgeSearchLimit {
		return recall
	}
	return data.MaxKnowledgeSearchLimit
}

// rerank 把通过距离阈值过滤后的分块按「余弦距离 + 关键词覆盖率」的混合分数重新排序，
// 再截断返回前 limit 条。排序是确定的：同分依次按 distance、ordinal 打破平局。
func rerank(query string, matches []data.KnowledgeMatch, limit int) []data.KnowledgeMatch {
	if limit < 1 || len(matches) == 0 {
		return nil
	}
	scored := make([]scoredKnowledgeMatch, len(matches))
	for i, match := range matches {
		scored[i] = scoredKnowledgeMatch{
			match:    match,
			coverage: keywordCoverage(query, match.Content),
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		left, right := scored[i], scored[j]
		if left.score() != right.score() {
			return left.score() < right.score()
		}
		if left.match.Distance != right.match.Distance {
			return left.match.Distance < right.match.Distance
		}
		return left.match.Ordinal < right.match.Ordinal
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	reordered := make([]data.KnowledgeMatch, len(scored))
	for i, item := range scored {
		reordered[i] = item.match
	}
	return reordered
}

type scoredKnowledgeMatch struct {
	match    data.KnowledgeMatch
	coverage float64
}

// score 将余弦距离与关键词覆盖率合并成重排序分数：分数越低越靠前，
// 覆盖率可抵消最多 KeywordRelevanceWeight 的余弦距离。
func (s scoredKnowledgeMatch) score() float64 {
	return s.match.Distance - KeywordRelevanceWeight*s.coverage
}

// keywordCoverage 返回查询中「有意义」的去重字符（忽略空白、标点和符号）
// 在正文里出现的比例。它是重排序用的轻量词法信号，是余弦距离的补充而非替代。
func keywordCoverage(query, content string) float64 {
	queryRunes := significantRunes(query)
	if len(queryRunes) == 0 {
		return 0
	}
	contentRunes := significantRunes(content)
	matched := 0
	for r := range queryRunes {
		if _, ok := contentRunes[r]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(queryRunes))
}

// significantRunes 返回承载词法信号的字符集合，丢弃空白、标点和符号。
func significantRunes(s string) map[rune]struct{} {
	set := make(map[rune]struct{})
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		set[r] = struct{}{}
	}
	return set
}
