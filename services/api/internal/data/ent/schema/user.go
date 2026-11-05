package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User 是登录账号。role 区分 staff（内部工作人员，可登录 Admin）与 user（普通用户，登录 Web）。
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("username").Unique().Immutable(),
		field.String("password_hash").Immutable(),
		// oauth_provider + oauth_subject 表示一个三方登录身份（provider 如
		// github/gitee，subject 是对应平台上的用户 id）。相比每个平台单独加一列
		// （github_id/gitee_id/…），泛化成两列更可扩展。
		field.String("oauth_provider").Optional().Nillable(),
		field.String("oauth_subject").Optional().Nillable(),
		field.String("role").Immutable(),
		// tenant_id 表示用户当前所属租户；加入组织时会切换，因此不设 Immutable。
		field.String("tenant_id"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at").Annotations(entsql.DescColumns("created_at")),
		// 同一平台下的 subject 唯一，跨平台可重复（同一个 subject 在 github 与
		// gitee 可能是不同的人）。
		index.Fields("oauth_provider", "oauth_subject").Unique(),
	}
}

// Annotations of the User.
func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		&entsql.Annotation{
			Checks: map[string]string{
				"users_role_check": "role IN ('staff', 'user')",
			},
		},
	}
}
