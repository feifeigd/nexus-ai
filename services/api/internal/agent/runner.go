package agent

import (
	"context"

	"nexusai/services/api/internal/types"
)

type EmitFunc func(types.AGUIEvent) error

// AG-UI Runner
type Runner interface {
	Run(context.Context, types.AGUIRunInput, EmitFunc) error
}

// 最新的用户输入消息
func LastUserText(messages []types.AGUIMessage) (string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		return messages[i].Content, true
	}
	return "", false
}
