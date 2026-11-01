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

## 数据源与迁移

`NewData` 是进程唯一的数据库连接所有者，负责连接池、Ping、迁移和关闭；Approval Store 复用同一个 Ent Client，不单独打开或关闭连接。

PostgreSQL 启用 `AutoMigrate` 时执行 `internal/data/migrations` 中经过评审的版本化 SQL。SQLite 启用 `AutoMigrate` 时使用 Ent `Schema.Create`；默认开发配置为共享内存数据库，进程退出后数据清空。SQLite 使用纯 Go `modernc.org/sqlite` 驱动，配置中的 Ent dialect 为 `sqlite3`，由 `NewData` 内部映射到底层 SQL 驱动名 `sqlite`。
