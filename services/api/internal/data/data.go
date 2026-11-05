package data

import (
	"context"
	stdsql "database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	_ "modernc.org/sqlite"

	"nexusai/services/api/internal/data/ent"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewTokenUsageStore,
	NewUserStore,
	NewTenantStore,
)

//go:embed migrations/001_approvals.sql
var postgresApprovalMigration string

//go:embed migrations/002_ticket_status.sql
var postgresTicketStatusMigration string

//go:embed migrations/003_knowledge_base.sql
var postgresKnowledgeBaseMigration string

//go:embed migrations/004_knowledge_embeddings.sql
var postgresKnowledgeEmbeddingsMigration string

//go:embed migrations/005_token_usage.sql
var postgresTokenUsageMigration string

//go:embed migrations/006_users.sql
var postgresUsersMigration string

//go:embed migrations/007_tenant_isolation.sql
var postgresTenantIsolationMigration string

//go:embed migrations/008_github_oauth.sql
var postgresGithubOAuthMigration string

//go:embed migrations/009_oauth_identity.sql
var postgresOAuthIdentityMigration string

//go:embed migrations/010_tenants.sql
var postgresTenantsMigration string

// DriverName returns the effective dialect after applying environment overrides.
func DriverName(c *Data) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(resolveDatabase(c).Driver)
}

func resolveDatabase(c *Data) Database {
	dc := c.Database
	if dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN")); dsn != "" {
		dc.Driver = "postgres"
		dc.Source = dsn
		dc.AutoMigrate = true
	}
	return dc
}

// NewData opens the process-owned database client. Stores share this client;
// only the returned cleanup function may close it.
func NewData(c *Data) (*ent.Client, func(), error) {
	if c == nil {
		return nil, nil, fmt.Errorf("database config is required")
	}

	dc := resolveDatabase(c)
	driverName := strings.TrimSpace(dc.Driver)
	source := strings.TrimSpace(dc.Source)
	if driverName != dialect.Postgres && driverName != dialect.SQLite {
		return nil, nil, fmt.Errorf("unsupported database driver %q", driverName)
	}
	if source == "" {
		return nil, nil, fmt.Errorf("database source is required")
	}

	sqlDriverName := driverName
	if driverName == dialect.SQLite {
		// modernc registers "sqlite"; Ent's SQLite dialect is named "sqlite3".
		sqlDriverName = "sqlite"
	}
	sqlDB, err := stdsql.Open(sqlDriverName, source)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s database: %w", driverName, err)
	}
	maxOpenConns := dc.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 20
		if driverName == dialect.SQLite {
			maxOpenConns = 1
		}
	}
	maxIdleConns := dc.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
		if driverName == dialect.SQLite {
			maxIdleConns = 1
		}
	}
	// Zero keeps connections alive indefinitely. Negative values use the
	// one-hour default; in-memory SQLite should normally leave this at zero.
	connMaxLifetime := dc.ConnMaxLifetime
	if connMaxLifetime < 0 {
		connMaxLifetime = time.Hour
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("connect %s database: %w", driverName, err)
	}

	client := ent.NewClient(ent.Driver(entsql.OpenDB(driverName, sqlDB)))
	if dc.Debug {
		client = client.Debug()
	}
	if dc.AutoMigrate {
		if err := migrate(ctx, driverName, sqlDB, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
	}

	logx.Infow("database connected", logx.Field("driver", driverName))
	cleanup := func() {
		logx.Infow("closing database", logx.Field("driver", driverName))
		if err := client.Close(); err != nil {
			logx.Errorw(
				"close database failed",
				logx.Field("driver", driverName),
				logx.Field("error", err),
			)
			return
		}
		logx.Infow("database closed", logx.Field("driver", driverName))
	}
	return client, cleanup, nil
}

func migrate(ctx context.Context, driverName string, db *stdsql.DB, client *ent.Client) error {
	if driverName == dialect.SQLite {
		if err := client.Schema.Create(ctx); err != nil {
			return fmt.Errorf("migrate sqlite database: %w", err)
		}
		return nil
	}

	// PostgreSQL keeps the reviewed, versioned SQL migrations authoritative.
	for _, migration := range []string{postgresApprovalMigration, postgresTicketStatusMigration, postgresKnowledgeBaseMigration, postgresKnowledgeEmbeddingsMigration, postgresTokenUsageMigration, postgresUsersMigration, postgresTenantIsolationMigration, postgresGithubOAuthMigration, postgresOAuthIdentityMigration, postgresTenantsMigration} {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migrate postgres database: %w", err)
		}
	}
	return nil
}
