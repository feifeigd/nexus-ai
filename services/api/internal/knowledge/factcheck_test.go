package knowledge

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type supportJudgeFunc func(ctx context.Context, sentences []string, chunks []string) ([]bool, error)

func (f supportJudgeFunc) Judge(ctx context.Context, sentences []string, chunks []string) ([]bool, error) {
	return f(ctx, sentences, chunks)
}

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
	note, err := answerSupportNote(context.Background(), nil, "蓝雀告警需要先检查 API 服务状态。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected supported answer to produce no note, got %q", note)
	}
}

func TestAnswerSupportNoteFlagsUnsupported(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	note, err := answerSupportNote(context.Background(), nil, "数据库迁移需要停机维护窗口。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected note to list the unsupported sentence, got %q", note)
	}
}

func TestAnswerSupportNoteSkipsShortSentences(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	note, err := answerSupportNote(context.Background(), nil, "好的。综上所述。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected short sentences to be skipped, got %q", note)
	}
}

func TestAnswerSupportNoteSkipsCitationOnlySentences(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	matches := []searchKnowledgeMatch{{DocumentID: valid, Content: "蓝雀告警需要先检查 API 服务状态"}}
	note, err := answerSupportNote(context.Background(), nil, "根据 "+valid+" 处理。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected citation-only sentence to be skipped, got %q", note)
	}
}

func TestAnswerSupportNoteStripsDocumentIDAndChecksContent(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	matches := []searchKnowledgeMatch{{DocumentID: valid, Content: "蓝雀告警需要先检查 API 服务状态"}}
	note, err := answerSupportNote(context.Background(), nil, "根据 "+valid+"，数据库迁移需要停机维护窗口。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected content after the document ID to be checked, got %q", note)
	}
}

func TestAnswerSupportNoteTruncatesLongList(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	answer := "数据库迁移需要停机维护窗口。容器编排使用声明式配置。缓存策略采用淘汰算法。消息队列保证可靠投递。"
	note, err := answerSupportNote(context.Background(), nil, answer, matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
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
	note, err := answerSupportNote(context.Background(), nil, "数据库迁移需要停机维护窗口。", nil)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected empty matches to skip support checking, got %q", note)
	}
}

func TestAnswerSupportNoteUsesJudgeVerdicts(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	judge := supportJudgeFunc(func(_ context.Context, sentences []string, _ []string) ([]bool, error) {
		return []bool{true}, nil // 语义改写后仍判定有支撑
	})
	note, err := answerSupportNote(context.Background(), judge, "数据库迁移需要停机维护窗口。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected the judge to clear the lexical mismatch, got %q", note)
	}
}

func TestAnswerSupportNoteFlagsJudgeUnsupported(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	judge := supportJudgeFunc(func(_ context.Context, sentences []string, _ []string) ([]bool, error) {
		return []bool{false}, nil
	})
	note, err := answerSupportNote(context.Background(), judge, "数据库迁移需要停机维护窗口。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if !strings.Contains(note, "数据库迁移") {
		t.Fatalf("expected the unsupported sentence to be reported, got %q", note)
	}
}

func TestAnswerSupportNotePropagatesJudgeError(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	judge := supportJudgeFunc(func(context.Context, []string, []string) ([]bool, error) {
		return nil, errors.New("judge unavailable")
	})
	if _, err := answerSupportNote(context.Background(), judge, "数据库迁移需要停机维护窗口。", matches); err == nil {
		t.Fatal("expected judge error to propagate")
	}
}

func TestAnswerSupportNoteSkipsJudgeForLiteralMatches(t *testing.T) {
	matches := []searchKnowledgeMatch{{Content: "蓝雀告警需要先检查 API 服务状态"}}
	called := false
	judge := supportJudgeFunc(func(_ context.Context, sentences []string, _ []string) ([]bool, error) {
		called = true
		return []bool{false}, nil // 即使语义判定误判无支撑，字面重叠足够时也不应被标记
	})
	note, err := answerSupportNote(context.Background(), judge, "蓝雀告警需要先检查 API 服务状态。", matches)
	if err != nil {
		t.Fatalf("answer support note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected literal match to be supported without judge, got %q", note)
	}
	if called {
		t.Fatalf("expected judge not to be called for literal matches")
	}
}

func TestValidateSearchAnswerCombinesCitationAndSupport(t *testing.T) {
	valid := "document_" + strings.Repeat("a", 32)
	fabricated := "document_" + strings.Repeat("f", 32)
	toolResult := `{"matches":[{"documentId":"` + valid + `","content":"蓝雀告警需要先检查 API 服务状态"}]}`
	answer := "根据 " + fabricated + "，数据库迁移需要停机维护窗口。"
	note, err := validateSearchAnswer(context.Background(), nil, answer, toolResult)
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
