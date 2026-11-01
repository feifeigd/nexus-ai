package logic

import (
	"time"

	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/types"
)

func approvalResponses(approvals []approval.Approval) []types.Approval {
	responses := make([]types.Approval, len(approvals))
	for i, item := range approvals {
		responses[i] = approvalResponse(item)
	}
	return responses
}

func approvalResponse(item approval.Approval) types.Approval {
	response := types.Approval{
		ID:         item.ID,
		ThreadID:   item.ThreadID,
		RunID:      item.RunID,
		ToolCallID: item.ToolCallID,
		ToolName:   item.ToolName,
		Arguments: types.CreateTicketInput{
			Title:       item.Arguments.Title,
			Description: item.Arguments.Description,
		},
		Status:    string(item.Status),
		TicketID:  item.TicketID,
		CreatedAt: formatTime(item.CreatedAt),
	}
	if item.DecidedAt != nil {
		response.DecidedAt = formatTime(*item.DecidedAt)
	}
	return response
}

func ticketResponses(tickets []approval.Ticket) []types.Ticket {
	responses := make([]types.Ticket, len(tickets))
	for i, item := range tickets {
		responses[i] = ticketResponse(item)
	}
	return responses
}

func ticketResponse(item approval.Ticket) types.Ticket {
	return types.Ticket{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Status:      string(item.Status),
		CreatedAt:   formatTime(item.CreatedAt),
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
