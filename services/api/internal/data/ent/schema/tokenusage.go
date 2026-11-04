package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TokenUsage 记录一次 AG-UI 对话（run）的 token 消耗明细。
// 当前按 thread/run 记录，不归属到用户（用户归属待认证/租户隔离接入）。
type TokenUsage struct {
	ent.Schema
}

// Fields of the TokenUsage.
func (TokenUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("thread_id").Immutable(),
		field.String("run_id").Immutable(),
		field.Int("prompt_tokens").Immutable(),
		field.Int("completion_tokens").Immutable(),
		field.Int("total_tokens").Immutable(),
		field.Int("judge_tokens").Default(0).Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the TokenUsage.
func (TokenUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at").Annotations(entsql.DescColumns("created_at")),
		index.Fields("thread_id", "created_at"),
	}
}
