package knowledge

import (
	"context"
	"strings"
	"testing"
)

func TestSplitSentences(t *testing.T) {
	got := splitSentences("第一句。第二句！第三句？\n第四句；第五句;第六句")
	want := []string{"第一句", "第二句", "第三句", "第四句", "第五句", "第六句"}
	if len(got) != len(want) {
		t.Fatalf("splitSentences = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitSentences[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSignificantBigramsBridgeNoScripts(t *testing.T) {
	got := significantBigrams("API服务")
	for _, want := range [][2]rune{{'A', 'P'}, {'P', 'I'}, {'服', '务'}} {
		if _, ok := got[want]; !ok {
			t.Fatalf("expected bigram %q in %v", want, got)
		}
	}
	if _, ok := got[[2]rune{'I', '服'}]; ok {
		t.Fatalf("should not bridge ASCII and non-ASCII runes: %v", got)
	}
	if len(significantBigrams("一")) != 0 || len(significantBigrams("")) != 0 {
		t.Fatalf("single-rune and empty input should have no bigrams")
	}
}

func TestAnswerSupportNoteSupported(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	if note := answerSupportNote("蓝雀告警需要先检查 API 服务状态。", matches); note != "" {
		t.Fatalf("expected supported answer to produce no note, got %q", note)
	}
}

func TestAnswerSupportNoteFlagsUnsupported(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	note := answerSupportNote("数据库迁移需要停机维护窗口。", matches)
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected note to list the unsupported sentence, got %q", note)
	}
}

func TestAnswerSupportNoteSkipsShortSentences(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	if note := answerSupportNote("好的。综上所述。", matches); note != "" {
		t.Fatalf("expected short sentences to be skipped, got %q", note)
	}
}

func TestAnswerSupportNoteSkipsCitationOnlySentences(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	matches := []searchKnowledgeMatch{{DocumentID: valid, Content: "蓝雀告警需要先检查 API 服务状态"}}
	if note := answerSupportNote("根据 "+valid+" 处理。", matches); note != "" {
		t.Fatalf("expected citation-only sentence to be skipped, got %q", note)
	}
}

func TestAnswerSupportNoteStripsDocumentIDAndChecksContent(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	matches := []searchKnowledgeMatch{{DocumentID: valid, Content: "蓝雀告警需要先检查 API 服务状态"}}
	note := answerSupportNote("根据 "+valid+"，数据库迁移需要停机维护窗口。", matches)
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected content after the document ID to be checked, got %q", note)
	}
}

func TestAnswerSupportNoteTruncatesLongList(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	answer := "数据库迁移需要停机维护窗口。容器编排使用声明式配置。缓存策略采用淘汰算法。消息队列保证可靠投递。"
	note := answerSupportNote(answer, matches)
	for _, want := range []string{"数据库迁移", "容器编排", "缓存策略", "……"} {
		if !strings.Contains(note, want) {
			t.Fatalf("expected note to contain %q, got %q", want, note)
		}
	}
	if strings.Contains(note, "消息队列") {
		t.Fatalf("expected the fourth sentence to be truncated, got %q", note)
	}
}

func TestAnswerSupportNoteEmptyMatches(t *testing.T) {
	if note := answerSupportNote("数据库迁移需要停机维护窗口。", nil); note != "" {
		t.Fatalf("expected empty matches to skip support checking, got %q", note)
	}
}

func TestValidateSearchAnswerCombinesCitationAndSupport(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	fabricated := "document_" + strings.Repeat("f", 32)
	toolResult := `{"matches":[{"documentId":"` + valid + `","content":"蓝雀告警需要先检查 API 服务状态"}]}`
	answer := "根据 " + fabricated + "，数据库迁移需要停机维护窗口。"
	note, err := validateSearchAnswer(context.Background(), answer, toolResult)
	if err != nil {
		t.Fatalf("validate answer: %v", err)
	}
	if !strings.Contains(note, fabricated) {
		t.Fatalf("expected fabricated citation to be reported, got %q", note)
	}
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected unsupported sentence to be reported, got %q", note)
	}
}
