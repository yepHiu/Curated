package server

import (
	"errors"
	"net/http"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (h *Handler) handleListAuthSessions(w http.ResponseWriter, r *http.Request) {
	current, _, err := h.authSessionFromRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate current auth session")
		return
	}
	dto, err := h.authSessionsDTO(r, current.ID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list trusted sessions")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleRevokeAuthSession(w http.ResponseWriter, r *http.Request) {
	publicID := strings.TrimSpace(r.PathValue("publicId"))
	if publicID == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "publicId is required")
		return
	}
	current, _, err := h.authSessionFromRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate current auth session")
		return
	}
	revoked, err := h.store.RevokeTrustedAuthSessionByPublicID(r.Context(), publicID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to revoke trusted session")
		return
	}
	if !revoked {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "trusted session not found")
		return
	}
	if current.PublicID == publicID {
		clearAuthCookie(w)
	}
	dto, err := h.authSessionsDTO(r, current.ID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list trusted sessions")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleRevokeOtherAuthSessions(w http.ResponseWriter, r *http.Request) {
	current, _, err := h.authSessionFromRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate current auth session")
		return
	}
	if _, err := h.store.RevokeOtherTrustedAuthSessions(r.Context(), current.ID); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to revoke other trusted sessions")
		return
	}
	dto, err := h.authSessionsDTO(r, current.ID)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to list trusted sessions")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) authSessionsDTO(r *http.Request, currentSessionID string) (contracts.AuthSessionsDTO, error) {
	if h == nil || h.store == nil {
		return contracts.AuthSessionsDTO{}, errors.New("security storage is not available")
	}
	sessions, err := h.store.ListTrustedAuthSessions(r.Context())
	if err != nil {
		return contracts.AuthSessionsDTO{}, err
	}
	items := make([]contracts.AuthSessionDTO, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, authSessionDTO(session, currentSessionID))
	}
	return contracts.AuthSessionsDTO{Items: items}, nil
}

func authSessionDTO(session storage.AuthSession, currentSessionID string) contracts.AuthSessionDTO {
	return contracts.AuthSessionDTO{
		PublicID:       session.PublicID,
		UserAgent:      session.UserAgent,
		IP:             session.IP,
		CreatedAt:      session.CreatedAt,
		LastSeenAt:     session.LastSeenAt,
		TrustedForever: session.TrustedForever,
		Current:        session.ID != "" && session.ID == currentSessionID,
	}
}
