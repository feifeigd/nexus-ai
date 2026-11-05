package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/handler"

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
	if secret := strings.TrimSpace(os.Getenv("NEXUS_AUTH_SECRET")); secret != "" {
		c.Auth.AccessSecret = secret
	}
	if v := strings.TrimSpace(os.Getenv("NEXUS_GITHUB_CLIENT_ID")); v != "" {
		c.GitHub.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv("NEXUS_GITHUB_CLIENT_SECRET")); v != "" {
		c.GitHub.ClientSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("NEXUS_GITHUB_REDIRECT_URI")); v != "" {
		c.GitHub.RedirectURI = v
	}
	if v := strings.TrimSpace(os.Getenv("NEXUS_WEB_BASE_URL")); v != "" {
		c.GitHub.WebBaseURL = v
	}

	ctx, cleanup, err := wireContext(&c)
	if err != nil {
		return fmt.Errorf("wire service context: %w", err)
	}
	defer cleanup()

	if err := seedAdminUser(context.Background(), ctx); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	server, err := rest.NewServer(c.RestConf)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}
	defer server.Stop()

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
