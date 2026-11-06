// 这行注释是一个构建约束，它告诉 go build 只有在满足条件 wireinject 的情况下才应该构建这个文件。
// wireinject 是一个特殊的标签，用于指示 wire 工具处理这个文件。
//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

// 执行命令生成 wire_gen.go
// wire ./...
// 或者
// go generate ./...

package main

import (
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"

	"github.com/google/wire"
)

func wireContext(*config.Config) (*svc.ServiceContext, func(), error) {
	panic(wire.Build(
		provideDataConfig,
		provideAgentConfig,
		data.DriverName,
		data.NewData,
		approval.NewEntStore,
		wire.Bind(new(approval.Store), new(*approval.EntStore)),
		newRunner,
		newKnowledgeStore,
		wire.Bind(new(svc.KnowledgeDocumentStore), new(*data.KnowledgeStore)),
		newKnowledgeService,
		newSSHRunner,
		data.ProviderSet,
		// ServiceContextDeps 用 wire.Struct 聚合所有业务依赖，
		// 再由 NewServiceContext 注入到 ServiceContext。
		wire.Struct(new(svc.ServiceContextDeps), "*"),
		svc.NewServiceContext,
	))
}
