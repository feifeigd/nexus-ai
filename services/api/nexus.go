package main

import (
	"flag"
	"fmt"

	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/handler"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/nexus-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server)

	fmt.Printf("Starting %s at %s:%d...\n", c.Name, c.Host, c.Port)
	server.Start()
}
