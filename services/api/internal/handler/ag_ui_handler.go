package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/types"
)

const maxAGUIRequestSize = 1 << 20

func AGUIHandler(runner agent.Runner) http.HandlerFunc {
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
			return
		}
		if r.Context().Err() != nil {
			return
		}

		log.Printf("AG-UI run %s failed: %v", input.RunID, err)
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
