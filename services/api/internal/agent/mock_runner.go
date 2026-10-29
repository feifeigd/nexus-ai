package agent

import (
	"context"
	"fmt"

	"nexusai/services/api/internal/types"
)

type MockRunner struct{}

func (MockRunner) Run(ctx context.Context, input types.AGUIRunInput, emit EmitFunc) error {
	prompt, ok := LastUserText(input.Messages)
	if !ok {
		return fmt.Errorf("a user message with plain text content is required")
	}

	messageID := input.RunID + "-assistant"
	events := []types.AGUIEvent{
		{Type: "RUN_STARTED", ThreadID: input.ThreadID, RunID: input.RunID},
		{Type: "TEXT_MESSAGE_START", MessageID: messageID, Role: "assistant"},
		{Type: "TEXT_MESSAGE_CONTENT", MessageID: messageID, Delta: "NexusAI Mock Agent 已收到："},
		{Type: "TEXT_MESSAGE_CONTENT", MessageID: messageID, Delta: prompt},
		{Type: "TEXT_MESSAGE_END", MessageID: messageID},
		{Type: "RUN_FINISHED", ThreadID: input.ThreadID, RunID: input.RunID},
	}
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(event); err != nil {
			return err
		}
	}
	return nil
}
