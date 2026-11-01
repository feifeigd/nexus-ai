package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/handler"
	"nexusai/services/api/internal/knowledge"
	"nexusai/services/api/internal/svc"

	entschema "nexusai/services/api/internal/data/ent/schema"

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

	// Preserve the existing environment-based PostgreSQL configuration while
	// keeping NewData as the only database owner in the process.
	if dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN")); dsn != "" {
		c.Data.Database.Driver = "postgres"
		c.Data.Database.Source = dsn
		c.Data.Database.AutoMigrate = true
	}
	client, closeData, err := data.NewData(&c.Data)
	if err != nil {
		return fmt.Errorf("configure data source: %w", err)
	}
	defer closeData()
	approvals := approval.NewEntStore(client, c.Data.Database.Driver)
	agentConfig := agent.ConfigFromEnv(c.Agent)
	runner, err := agent.NewRunner(agentConfig, approvals.Tool())
	if err != nil {
		return fmt.Errorf("configure agent: %w", err)
	}

	var knowledgeService svc.KnowledgeService
	if agentConfig.OpenAI.EmbeddingModel != "" {
		embedder, err := agent.NewOpenAIEmbedder(agentConfig.OpenAI, nil, entschema.EmbeddingDimensions)
		if err != nil {
			return fmt.Errorf("configure knowledge embedder: %w", err)
		}
		knowledgeService, err = knowledge.NewService(
			embedder,
			data.NewKnowledgeStore(client, c.Data.Database.Driver),
			0,
			0,
		)
		if err != nil {
			return fmt.Errorf("configure knowledge service: %w", err)
		}
	}

	server, err := rest.NewServer(c.RestConf)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}
	defer server.Stop()

	ctx := svc.NewServiceContext(c, runner, approvals, knowledgeService)
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
