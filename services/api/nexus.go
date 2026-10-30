package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/handler"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/nexus-api.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}

	var c config.Config
	conf.MustLoad(*configFile, &c)

	approvals, closeStore := approvalStore()
	defer closeStore()
	runner, err := agent.NewRunner(agent.ConfigFromEnv(c.Agent), approvals.Tool())
	if err != nil {
		log.Fatalf("configure agent: %v", err)
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, runner, approvals)

	fmt.Printf("Starting %s at %s:%d...\n", c.Name, c.Host, c.Port)
	server.Start()
}

func approvalStore() (approval.Store, func()) {
	dsn := strings.TrimSpace(os.Getenv("NEXUS_POSTGRES_DSN"))
	if dsn == "" {
		log.Print("approval store: memory")
		return approval.NewStore(), func() {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := approval.NewPostgresStore(ctx, dsn)
	if err != nil {
		log.Fatalf("configure postgres approval store: %v", err)
	}
	log.Print("approval store: postgres")
	return store, func() {
		if err := store.Close(); err != nil {
			log.Printf("close postgres approval store: %v", err)
		}
	}
}
