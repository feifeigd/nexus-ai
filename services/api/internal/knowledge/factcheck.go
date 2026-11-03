package knowledge

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

const (
	// minSentenceRunes is the minimum number of significant runes a sentence
	// must contain after stripping document IDs before its source support is
	// checked. Shorter fragments ("好的", "综上所述") are too weak to judge.
	minSentenceRunes = 8

	// sentenceSupportThreshold is the minimum bigram overlap a sentence must
	// have with at least one retrieved chunk to be considered supported.
	sentenceSupportThreshold = 0.3

	// maxUnsupportedSentences caps how many unsupported sentences are listed in
	// the visible note, keeping it readable.
	maxUnsupportedSentences = 3
)

// answerSupportNote splits the answer into sentences and checks each against
// the retrieved chunks for source support, returning a note listing sentences
// that are not sufficiently supported. When judge is set it defers to the
// semantic (NLI/LLM) verdicts; otherwise it falls back to the lexical bigram
// overlap heuristic. It returns "" when every judged sentence is supported.
func answerSupportNote(ctx context.Context, judge SupportJudge, answer string, matches []searchKnowledgeMatch) (string, error) {
	if len(matches) == 0 {
		return "", nil // nothing to compare against
	}
	sentences := make([]string, 0)
	cleaned := make([]string, 0)
	for _, sentence := range splitSentences(answer) {
		stripped := searchKnowledgeDocumentIDPattern.ReplaceAllString(sentence, "")
		if len(significantRunes(stripped)) < minSentenceRunes {
			continue
		}
		if isOrganizationalSentence(stripped) {
			continue
		}
		sentences = append(sentences, sentence)
		cleaned = append(cleaned, stripped)
	}
	if len(sentences) == 0 {
		return "", nil
	}

	supported := make([]bool, len(sentences))
	if judge != nil {
		// 字面 bigram 先行：字面重叠足够的句子直接判支撑，避免语义判定器对
		// “原文照抄/高度重叠”句子的误报；只有字面重叠不足的句子才交给 LLM
		// 语义判定（用于识别同义改写或编造）。
		var pending []int
		for i, stripped := range cleaned {
			if supportedByAnyChunk(stripped, matches) {
				supported[i] = true
			} else {
				pending = append(pending, i)
			}
		}
		if len(pending) > 0 {
			chunks := make([]string, len(matches))
			for i, match := range matches {
				chunks[i] = match.Content
			}
			pendingSentences := make([]string, len(pending))
			for i, idx := range pending {
				pendingSentences[i] = cleaned[idx]
			}
			verdicts, err := judge.Judge(ctx, pendingSentences, chunks)
			if err != nil {
				return "", err
			}
			if len(verdicts) != len(pending) {
				return "", fmt.Errorf("support judge returned %d verdicts for %d sentences", len(verdicts), len(pending))
			}
			for i, idx := range pending {
				supported[idx] = verdicts[i]
			}
		}
	} else {
		for i, stripped := range cleaned {
			supported[i] = supportedByAnyChunk(stripped, matches)
		}
	}

	unsupported := make([]string, 0, maxUnsupportedSentences)
	for i, ok := range supported {
		if ok {
			continue
		}
		unsupported = append(unsupported, strings.TrimSpace(sentences[i]))
	}
	if len(unsupported) == 0 {
		return "", nil
	}
	// 截断了
	truncated := len(unsupported) > maxUnsupportedSentences
	if truncated {
		unsupported = unsupported[:maxUnsupportedSentences]
	}
	note := "以下回答未能被本次检索结果充分支撑，建议人工核实：" + strings.Join(unsupported, "；")
	if truncated {
		note += "；……"
	}
	return note, nil
}

// supportedByAnyChunk 是语义判定器未配置时的字面 bigram 启发式兜底：判断某句
// 是否被至少一个检索分块「字面」支撑。
//
// 三个要素：
//   - bigram（二元组）：句子去掉空白、标点、符号后，每相邻两个字组成一对，
//     例如「蓝雀告警」→「蓝雀」「雀告」「告警」；跨中英文的相邻字不成对，
//     所以「API服务」得到「AP」「PI」「服务」，不会出现「I服」。
//   - 字面：只比对字符序列是否相同，不关心语义。某句的 bigram 有多少比例出现在
//     某个分块的 bigram 集合里，就是它与该分块的重叠率（bigramOverlap）。
//   - 启发式：重叠率达到 sentenceSupportThreshold（当前 0.3）即认定「有支撑」，
//     否则在答案末尾追加人工核实告警。这是经验规则而非严谨判定：原文照抄重叠率
//     高、判为支撑；同义改写会改变字符对、重叠率骤降，可能被误标为无支撑。
//     这正是引入语义判定器（SupportJudge / OpenAISupportJudge）的原因。
func supportedByAnyChunk(sentence string, matches []searchKnowledgeMatch) bool {
	sentenceBigrams := significantBigrams(sentence)
	if len(sentenceBigrams) == 0 {
		return true // nothing meaningful to check
	}
	for _, match := range matches {
		if bigramOverlap(sentenceBigrams, match.Content) >= sentenceSupportThreshold {
			return true
		}
	}
	return false
}

// 计算重叠率
func bigramOverlap(sentence map[[2]rune]struct{}, content string) float64 {
	contentBigrams := significantBigrams(content)
	if len(contentBigrams) == 0 {
		return 0
	}
	matched := 0
	for bigram := range sentence {
		if _, ok := contentBigrams[bigram]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(sentence))
}

// splitSentences splits on sentence-ending punctuation, returning trimmed
// non-empty fragments.
// 按 。！？；!?;\n 拆句
func splitSentences(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case '。', '！', '？', '；', '!', '?', ';', '\n', '\r':
			return true
		}
		return false
	})
}

// isOrganizationalSentence reports whether a sentence is organizational
// language — a markdown heading or a colon-ended lead-in such as
// “它可以分两种情况处理：” — rather than a knowledge claim worth fact-checking.
// Such sentences are not source-backed by design, so flagging them only adds
// noise to the answer-validation note.
func isOrganizationalSentence(s string) bool {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "#") {
		return true // markdown 标题
	}
	if strings.HasSuffix(trimmed, "：") || strings.HasSuffix(trimmed, ":") {
		return true // 引导/连接语
	}
	return false
}

// 启发式二元组
// significantBigrams returns the set of two-rune bigrams formed by consecutive
// significant runes, never bridging an ASCII letter/digit with a non-ASCII
// rune (so "API服务" yields "AP", "PI", "服务" but not "I服").
func significantBigrams(s string) map[[2]rune]struct{} {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		runes = append(runes, r)
	}
	// 相邻两个字组成一个二元组；跨中英文的相邻字不成对（避免 "I服"），
	// 因此 "API服务" 得到 "AP"、"PI"、"服务"。
	bigrams := make(map[[2]rune]struct{})
	for i := 0; i+1 < len(runes); i++ {
		left, right := runes[i], runes[i+1]
		if isASCIIAlnum(left) != isASCIIAlnum(right) {
			continue
		}
		bigrams[[2]rune{left, right}] = struct{}{}
	}
	return bigrams
}

func isASCIIAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
