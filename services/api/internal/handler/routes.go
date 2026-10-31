package handler

import (
	"net/http"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, runner agent.Runner, approvals approval.Store) {
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
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/approvals",
		Handler: ListApprovalsHandler(approvals),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/tickets",
		Handler: ListTicketsHandler(approvals),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPatch,
		Path:    "/tickets/:id/status",
		Handler: UpdateTicketStatusHandler(approvals),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/approvals/:id/approve",
		Handler: ApproveApprovalHandler(approvals),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/approvals/:id/reject",
		Handler: RejectApprovalHandler(approvals),
	})
}
