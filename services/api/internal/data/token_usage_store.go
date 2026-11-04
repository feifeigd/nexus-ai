package data

import (
	"context"
	"fmt"
	"time"

	"nexusai/services/api/internal/data/ent"
	enttokenusage "nexusai/services/api/internal/data/ent/tokenusage"
)

// TokenUsageRecord 是一次对话（run）的 token 消耗明细。
type TokenUsageRecord struct {
	ThreadID         string
	RunID            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	JudgeTokens      int
	CreatedAt        time.Time
}

// TokenUsageStore 持久化对话 token 消耗明细。当前按 thread/run 记录，
// 不归属到用户（用户归属待认证/租户隔离接入）。
type TokenUsageStore struct {
	client *ent.Client
}

func NewTokenUsageStore(client *ent.Client) *TokenUsageStore {
	return &TokenUsageStore{client: client}
}

// Create 记录一次对话的 token 消耗。CreatedAt 为零值时使用当前时间。
func (s *TokenUsageStore) Create(ctx context.Context, record TokenUsageRecord) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("token usage store client is required")
	}
	builder := s.client.TokenUsage.Create().
		SetThreadID(record.ThreadID).
		SetRunID(record.RunID).
		SetPromptTokens(record.PromptTokens).
		SetCompletionTokens(record.CompletionTokens).
		SetTotalTokens(record.TotalTokens).
		SetJudgeTokens(record.JudgeTokens)
	if !record.CreatedAt.IsZero() {
		builder = builder.SetCreatedAt(record.CreatedAt)
	}
	if _, err := builder.Save(ctx); err != nil {
		return fmt.Errorf("create token usage: %w", err)
	}
	return nil
}

// List 返回最近 limit 条消耗明细（按时间倒序）。limit <= 0 时使用默认值。
func (s *TokenUsageStore) List(ctx context.Context, limit int) ([]TokenUsageRecord, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("token usage store client is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.client.TokenUsage.Query().
		Order(ent.Desc(enttokenusage.FieldCreatedAt), ent.Desc(enttokenusage.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list token usage: %w", err)
	}
	records := make([]TokenUsageRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, TokenUsageRecord{
			ThreadID:         row.ThreadID,
			RunID:            row.RunID,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			TotalTokens:      row.TotalTokens,
			JudgeTokens:      row.JudgeTokens,
			CreatedAt:        row.CreatedAt,
		})
	}
	return records, nil
}
