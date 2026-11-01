package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/handler"
	"nexusai/services/api/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/nexus-api.yaml", "the config file")

func main() {
	flag.Parse()

	if err := run(); err != nil {
		logx.Errorw(
			"nexus api exited with error",
			logx.Field("error", err),
		)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	var c config.Config
	if err := conf.Load(*configFile, &c); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	approvals, closeStore, err := approvalStore()
	if err != nil {
		return err
	}
	defer closeStore()

	runner, err := agent.NewRunner(agent.ConfigFromEnv(c.Agent), approvals.Tool())
	if err != nil {
		return fmt.Errorf("configure agent: %w", err)
	}

	server, err := rest.NewServer(c.RestConf)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}
	defer server.Stop()

	ctx := svc.NewServiceContext(c, runner, approvals)
	handler.RegisterHandlers(server, ctx)

	logx.Infow(
		"nexus api starting",
		logx.Field("service", c.Name),
		logx.Field("host", c.Host),
		logx.Field("port", c.Port),
	)
	server.Start()
	return nil
}

func approvalStore() (approval.Store, func(), error) {
	dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN"))
	if dsn == "" {
		logx.Info("approval store configured as memory")
		return approval.NewStore(), func() {}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := approval.NewPostgresStore(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("configure postgres approval store: %w", err)
	}

	logx.Info("approval store configured as postgres")

	return store, func() {
		logx.Info("closing postgres approval store")

		if err := store.Close(); err != nil {
			logx.Errorw(
				"close postgres approval store failed",
				logx.Field("error", err),
			)
			return
		}

		logx.Info("postgres approval store closed")
	}, nil
}
