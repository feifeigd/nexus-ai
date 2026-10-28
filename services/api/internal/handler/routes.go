package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server) {
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/healthz",
		Handler: HealthHandler(),
	})
}
