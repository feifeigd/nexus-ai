package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Machine 是一台运营中的裸机/服务器，属于平台级基础设施（不做租户隔离）。它记录
// 机器的网络地址、所在区域、root 公钥，以及是否已加入 Kubernetes 集群。
type Machine struct {
	ent.Schema
}

// Fields of the Machine.
func (Machine) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name").Optional().Nillable().MaxLen(200),
		field.String("private_address").NotEmpty(),
		field.String("public_address").Optional().Nillable(),
		field.String("region").Optional().Nillable(),
		field.String("root_public_key").Optional().Nillable(),
		field.Bool("joined_k8s").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the Machine.
func (Machine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("private_address").Unique(),
	}
}
