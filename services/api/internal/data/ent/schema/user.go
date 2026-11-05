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
		field.String("github_id").Optional().Nillable().Unique().Immutable(),
		field.String("role").Immutable(),
		field.String("tenant_id").Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at").Annotations(entsql.DescColumns("created_at")),
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
