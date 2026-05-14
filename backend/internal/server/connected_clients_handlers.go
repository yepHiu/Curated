package server

import (
	"net/http"
	"time"

	"curated-backend/internal/clienttracker"
	"curated-backend/internal/contracts"
)

func withClientTracking(next http.Handler, tracker *clienttracker.Tracker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tracker != nil {
			tracker.Record(r)
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) handleConnectedClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	snapshots := h.clientTracker.Snapshot()
	dto := contracts.ConnectedClientsDTO{
		Clients:   make([]contracts.ConnectedClientDTO, 0, len(snapshots)),
		Total:     len(snapshots),
		SampledAt: time.Now().UTC().Format(time.RFC3339),
	}
	for _, snapshot := range snapshots {
		if snapshot.AccessKind == clienttracker.AccessKindLocal {
			dto.LocalCount += 1
		} else {
			dto.RemoteCount += 1
		}
		dto.Clients = append(dto.Clients, contracts.ConnectedClientDTO{
			Key:            snapshot.Key,
			IP:             snapshot.IP,
			Port:           snapshot.Port,
			Hostname:       snapshot.Hostname,
			UserAgent:      snapshot.UserAgent,
			Browser:        snapshot.Browser,
			BrowserVersion: snapshot.BrowserVersion,
			OS:             snapshot.OS,
			OSVersion:      snapshot.OSVersion,
			DeviceType:     string(snapshot.DeviceType),
			AccessKind:     string(snapshot.AccessKind),
			IsLocalMachine: snapshot.IsLocalMachine,
			FirstSeen:      snapshot.FirstSeen.UTC().Format(time.RFC3339Nano),
			LastSeen:       snapshot.LastSeen.UTC().Format(time.RFC3339Nano),
			RequestCount:   snapshot.RequestCount,
		})
	}
	writeJSON(w, http.StatusOK, dto)
}
