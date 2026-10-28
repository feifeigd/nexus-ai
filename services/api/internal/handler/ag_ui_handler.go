package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nexusai/services/api/internal/types"
)

const maxAGUIRequestSize = 1 << 20

func AGUIHandler() http.HandlerFunc {
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

		prompt, ok := lastUserText(input.Messages)
		if !ok {
			http.Error(w, "a user message with plain text content is required", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming is not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

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
			if r.Context().Err() != nil {
				return
			}
			if err := writeAGUIEvent(w, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func lastUserText(messages []types.AGUIMessage) (string, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		text, ok := messages[i].Content.(string)
		return text, ok
	}
	return "", false
}

func writeAGUIEvent(w io.Writer, event types.AGUIEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}
