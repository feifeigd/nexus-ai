package knowledge

import (
	"math"
	"sort"
	"unicode"

	"nexusai/services/api/internal/data"
)

const (
	// SearchRecallFactor 是召回时放大请求 limit 的倍数，为重排序提供候选池。
	SearchRecallFactor = 4

	// KeywordRelevanceWeight 控制关键词分数在混合分数中能抵消的余弦距离量：
	// 分数越低越靠前，归一化 BM25 关键词分数每 1 单位可抵消这么多单位的余弦距离。
	KeywordRelevanceWeight = 0.15

	// bm25K1 与 bm25B 是 BM25 的词频饱和与文档长度归一化参数，沿用业界常用默认值。
	bm25K1 = 1.2 // k1 控制饱和速度
	bm25B  = 0.75
)

// searchRecallLimit 返回重排序前从 pgvector 召回的候选数量，即 min(limit×4, 50)，
// 不会超过存储层的最大召回上限。
func searchRecallLimit(limit int) int {
	if recall := limit * SearchRecallFactor; recall < data.MaxKnowledgeSearchLimit {
		return recall
	}
	return data.MaxKnowledgeSearchLimit
}

// rerank 把通过距离阈值过滤后的分块按「余弦距离 + BM25 关键词分数」的混合分数重新排序，
// 再截断返回前 limit 条。关键词分数基于 bigram 词元、在候选池上统计文档频率并归一化到
// [0,1]，是余弦距离的补充而非替代。排序是确定的：同分依次按 distance、ordinal 打破平局。
func rerank(query string, matches []data.KnowledgeMatch, limit int) []data.KnowledgeMatch {
	if limit < 1 || len(matches) == 0 {
		return nil
	}
	keyword := bm25Scores(query, matches)
	scored := make([]scoredKnowledgeMatch, len(matches))
	for i, match := range matches {
		scored[i] = scoredKnowledgeMatch{match: match, keyword: keyword[i]}
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
	match   data.KnowledgeMatch
	keyword float64 // 归一化到 [0,1] 的 BM25 关键词分数
}

// score 将余弦距离与关键词分数合并成重排序分数：分数越低越靠前，
// 关键词分数可抵消最多 KeywordRelevanceWeight 的余弦距离。
func (s scoredKnowledgeMatch) score() float64 {
	return s.match.Distance - KeywordRelevanceWeight*s.keyword
}

// bm25Scores 对候选池中的每个分块计算 BM25 关键词分数，并按池内最大值归一化到 [0,1]。
// 词元是 bigram（相邻两个有意义的字符，跨中英文不成对），IDF 与平均分块长度都只在候选池
// 内统计——这是「局部」BM25，避免维护全局倒排索引，同时保留词频与逆文档频率两类信号：
// 出现次数更多、或在候选池中更罕见（区分度更高）的词元，会给分块带来更高的关键词分数。
// 一句话总结：BM25 = 词频 × 罕见度 ÷ 长度惩罚，是对「关键词命中质量」的精细刻画。
func bm25Scores(query string, matches []data.KnowledgeMatch) []float64 {
	if len(matches) == 0 {
		return nil
	}
	queryTerms := significantBigramCounts(query)
	scores := make([]float64, len(matches))
	if len(queryTerms) == 0 {
		return scores
	}

	n := len(matches)
	// 词频
	chunkTF := make([]map[[2]rune]int, n)
	docLen := make([]int, n)
	totalLen := 0
	dfs := make(map[[2]rune]int, len(queryTerms))
	for i, match := range matches {
		tf := significantBigramCounts(match.Content)
		chunkTF[i] = tf
		length := 0
		for term, count := range tf {
			length += count
			if _, isQueryTerm := queryTerms[term]; isQueryTerm {
				dfs[term]++
			}
		}
		docLen[i] = length
		totalLen += length
	}

	// 逆文档频率 idf：越罕见的词越值钱
	// df 越小（越罕见）→ idf 越大。
	//   N = 候选池分块总数
	//   df = 候选池里包含这个词的分块数
	// idf = ln( 1 + (N − df + 0.5) / (df + 0.5) )
	idf := make(map[[2]rune]float64, len(dfs))
	for term, df := range dfs {
		idf[term] = math.Log(1 + (float64(n)-float64(df)+0.5)/(float64(df)+0.5))
	}
	avgdl := float64(totalLen) / float64(n)
	maxScore := 0.0
	for i := range matches {
		score := 0.0
		for term, idf := range idf {
			tf := float64(chunkTF[i][term])
			if tf == 0 {
				continue
			}
			denom := tf + bm25K1*(1-bm25B+bm25B*float64(docLen[i])/avgdl)
			score += idf * (tf * (bm25K1 + 1)) / denom
		}
		scores[i] = score
		if score > maxScore {
			maxScore = score
		}
	}
	if maxScore > 0 {
		for i := range scores {
			scores[i] /= maxScore
		}
	}
	return scores
}

// significantBigramCounts 返回相邻两个有意义字符组成的 bigram 及其出现次数，跨中英文的
// 相邻字不成对（与 factcheck 的 significantBigrams 同一规则，这里额外保留词频）。
func significantBigramCounts(s string) map[[2]rune]int {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		runes = append(runes, r)
	}
	counts := make(map[[2]rune]int)
	for i := 0; i+1 < len(runes); i++ {
		left, right := runes[i], runes[i+1]
		if isASCIIAlnum(left) != isASCIIAlnum(right) {
			continue
		}
		counts[[2]rune{left, right}]++
	}
	return counts
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
