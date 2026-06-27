package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

const eventStreamHeartbeatInterval = 20 * time.Second

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	if h.tasks == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "task event runtime not available")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "event streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	_, events, unsubscribe := h.tasks.Subscribe(64)
	defer unsubscribe()

	if err := writeSSEJSON(w, "hello", map[string]string{"type": "hello"}); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(eventStreamHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSEJSON(w, event.Type, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSEJSON(w http.ResponseWriter, eventName string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", eventName); err != nil {
		return err
	}
	for _, line := range strings.Split(string(encoded), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}
