package knowledge

import (
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
// the retrieved chunks for lexical support, returning a note listing sentences
// whose wording has low overlap with the sources. It returns "" when every
// judged sentence is sufficiently supported.
//
// This is a lexical heuristic, not semantic fact verification: a sentence that
// paraphrases the source with different wording may still be flagged, so the
// note asks for manual review rather than asserting fabrication.
func answerSupportNote(answer string, matches []searchKnowledgeMatch) string {
	if len(matches) == 0 {
		return "" // nothing to compare against
	}
	unsupported := make([]string, 0, maxUnsupportedSentences)
	for _, sentence := range splitSentences(answer) {
		cleaned := searchKnowledgeDocumentIDPattern.ReplaceAllString(sentence, "")
		if len(significantRunes(cleaned)) < minSentenceRunes {
			continue
		}
		if supportedByAnyChunk(cleaned, matches) {
			continue
		}
		unsupported = append(unsupported, strings.TrimSpace(sentence))
	}
	if len(unsupported) == 0 {
		return ""
	}
	// 截断了
	truncated := len(unsupported) > maxUnsupportedSentences
	if truncated {
		unsupported = unsupported[:maxUnsupportedSentences]
	}
	note := "以下回答与本次检索结果的字面重叠较低，建议人工核实：" + strings.Join(unsupported, "；")
	if truncated {
		note += "；……"
	}
	return note
}

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
