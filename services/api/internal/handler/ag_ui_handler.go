package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxAGUIRequestSize = 1 << 20

func AGUIHandler(runner agent.Runner, tokenUsage *data.TokenUsageStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input types.AGUIRunInput
		body := http.MaxBytesReader(w, r.Body, maxAGUIRequestSize)
		if err := json.NewDecoder(body).Decode(&input); err != nil {
			http.Error(w, "invalid AG-UI request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.ThreadID) == "" || strings.TrimSpace(input.RunID) == "" {
			http.Error(w, "threadId and runId are required", http.StatusBadRequest)
			return
		}

		if _, ok := agent.LastUserText(input.Messages); !ok {
			http.Error(w, "a user message with plain text content is required", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming is not supported", http.StatusInternalServerError)
			return
		}

		started := false
		err := runner.Run(r.Context(), input, func(event types.AGUIEvent) error {
			if !started {
				w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("X-Accel-Buffering", "no")
				w.WriteHeader(http.StatusOK)
				started = true
			}
			if err := writeAGUIEvent(w, event); err != nil {
				return err
			}
			flusher.Flush()
			return nil
		})
		if err == nil {
			if !started {
				http.Error(w, "agent returned an empty run", http.StatusBadGateway)
			}
			recordTokenUsage(r, runner, input, tokenUsage)
			return
		}
		if r.Context().Err() != nil {
			return
		}

		logx.WithContext(r.Context()).Errorw(
			"AG-UI run failed",
			logx.Field("threadId", input.ThreadID),
			logx.Field("runId", input.RunID),
			logx.Field("error", err),
		)
		if !started {
			http.Error(w, "agent run failed", http.StatusBadGateway)
			return
		}
		_ = writeAGUIEvent(w, types.AGUIEvent{
			Type:     "RUN_ERROR",
			ThreadID: input.ThreadID,
			RunID:    input.RunID,
			Message:  "agent run failed",
		})
		flusher.Flush()
	}
}

func writeAGUIEvent(w io.Writer, event types.AGUIEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

// recordTokenUsage 在对话成功后，若 runner 报告了 token 消耗，则按 thread/run 持久化。
// 记录失败只记日志，不影响对话响应。
func recordTokenUsage(r *http.Request, runner agent.Runner, input types.AGUIRunInput, tokenUsage *data.TokenUsageStore) {
	if tokenUsage == nil {
		return
	}
	reporter, ok := runner.(interface{ Usage() agent.Usage })
	if !ok {
		return
	}
	usage := reporter.Usage()
	if usage.TotalTokens <= 0 {
		return
	}
	record := data.TokenUsageRecord{
		ThreadID:         input.ThreadID,
		RunID:            input.RunID,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
	if err := tokenUsage.Create(r.Context(), record); err != nil {
		logx.WithContext(r.Context()).Errorw(
			"record token usage",
			logx.Field("threadId", input.ThreadID),
			logx.Field("runId", input.RunID),
			logx.Field("error", err),
		)
	}
}
