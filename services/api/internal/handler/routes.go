package handler

import (
	"net/http"

	"nexusai/services/api/internal/agent"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, runner agent.Runner) {
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/healthz",
		Handler: HealthHandler(),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/ag-ui",
		Handler: AGUIHandler(runner),
	})
}
