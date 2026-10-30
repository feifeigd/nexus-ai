package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

	approvals := approval.NewStore()
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
