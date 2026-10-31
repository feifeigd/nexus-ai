# 这里是数据库模型的定义

## ent 文档

[Ent 官方文档](https://entgo.io/docs/getting-started/)

## 新建模型
```shell
ent new 模型
```

## 外键

参考 `ent/schema/approval.go` 和 `ent/schema/ticket.go`。

## 代码生成

在 `services/api` 目录执行：

```shell
go generate ./internal/data/ent
```

生成代码按 `.gitignore` 规则保留在本地，不要手动修改。修改 Schema 后必须重新生成。

## 数据库迁移

现有 `internal/approval/migrations` 中的版本化 SQL 仍是数据库结构迁移的权威来源；Ent 负责数据访问，不调用 `Schema.Create` 自动迁移现有数据库。
