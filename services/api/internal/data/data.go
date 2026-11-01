package data

import (
	"context"
	stdsql "database/sql"

	// "kratos-todo-service/internal/conf"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	"nexusai/services/api/internal/data/ent"

	// _ "github.com/go-sql-driver/mysql" // 导入数据库驱动

	// PostgreSQL 驱动，注册名为 "postgres"
	_ "github.com/lib/pq"

	// SQLite 驱动，注册名为 "sqlite3"
	_ "github.com/mattn/go-sqlite3"

	"github.com/zeromicro/go-zero/core/logx"
	// "github.com/google/wire"
)

// var ProviderSet = wire.NewSet(NewData, NewTodoRepo)

// NewData opens the database client and returns it with a cleanup function.
// 创建数据库连接
// 构建函数的返回值:
// 1: *
// 2: *, error
// 3: *, cleanup func(), error
func NewData(c *Data) (*ent.Client, func(), error) {
	dc := c.Database
	sqlDB, err := stdsql.Open(dc.Driver, dc.Source)
	if err != nil {
		return nil, nil, err
	}
	// 连接池配置:database/sql 默认 MaxIdleConns=2,空闲连接很快被回收,
	// 每个请求都新建连接。本机 MySQL 建连要几百毫秒(反向 DNS),
	// 不配池子会直接把请求拖过 deadline。
	// 配置里留 0 则回落到下面的默认值。
	maxOpenConns := dc.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 20
	}
	maxIdleConns := dc.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	connMaxLifetime := time.Hour
	if d := dc.ConnMaxLifetime; d != nil {
		connMaxLifetime = d.AsDuration()
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	// 寿命要小于 MySQL wait_timeout(默认 28800s),避免服务端掐连接后
	// 客户端拿到死连接报 driver: bad connection。
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// 用配好池子的 *sql.DB 构建 ent client
	db := ent.NewClient(ent.Driver(entsql.OpenDB(dc.Driver, sqlDB)))

	if dc.Debug {
		db = db.Debug()
	}

	// Auto migration is a convenience for local development.
	// In production, apply schema changes as a separate reviewed step instead.
	if dc.AutoMigrate {
		if err := db.Schema.Create(context.Background()); err != nil {
			db.Close()
			return nil, nil, err
		}
	}
	// 析构函数
	cleanup := func() {
		logx.Info("Closing the data resources.")
		if err := db.Close(); err != nil {
			logx.Error("failed closing the database", "err", err)
		}
	}
	return db, cleanup, nil
}
