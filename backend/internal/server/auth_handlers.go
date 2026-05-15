package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const (
	authCookieName             = "curated_auth"
	trustedForeverCookieMaxAge = 10 * 365 * 24 * 60 * 60
)

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	dto, err := h.authStatusForRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read auth status")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleSetupPIN(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "security storage is not available")
		return
	}

	var body contracts.SetupPINRequest
	if err := decodeAuthJSONBody(r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	body.PIN = strings.TrimSpace(body.PIN)
	body.ConfirmPIN = strings.TrimSpace(body.ConfirmPIN)
	if err := validatePIN(body.PIN); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	if body.PIN != body.ConfirmPIN {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "pin confirmation does not match")
		return
	}

	settings, err := h.store.GetAppSecuritySettings(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read security settings")
		return
	}
	if settings.PINEnabled {
		if _, ok, err := h.authSessionFromRequest(r); err != nil {
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate auth session")
			return
		} else if !ok {
			writeAppError(w, http.StatusLocked, contracts.ErrorCodeAuthLocked, "Curated is locked")
			return
		}
	}

	if err := h.store.SetAppPIN(r.Context(), body.PIN); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to set pin")
		return
	}
	patch := storage.AppSecuritySettingsPatch{
		SessionTTLMinutes: body.SessionTTLMinutes,
		LANRequiresPIN:    body.LANRequiresPIN,
		LockOnRestart:     body.LockOnRestart,
	}
	settings, err = h.store.PatchAppSecuritySettings(r.Context(), patch)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update security settings")
		return
	}

	session, err := h.createAuthSessionForRequest(r, settings, body.TrustedForever)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to create auth session")
		return
	}
	writeAuthCookie(w, session)
	writeJSON(w, http.StatusOK, authStatusFromSettings(settings, &session))
}

func (h *Handler) handleUnlockPIN(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "security storage is not available")
		return
	}

	var body contracts.UnlockPINRequest
	if err := decodeAuthJSONBody(r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	ok, err := h.store.VerifyAppPIN(r.Context(), body.PIN)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to verify pin")
		return
	}
	if !ok {
		writeAppError(w, http.StatusUnauthorized, contracts.ErrorCodeAuthInvalidPIN, "PIN is incorrect")
		return
	}
	settings, err := h.store.GetAppSecuritySettings(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read security settings")
		return
	}
	session, err := h.createAuthSessionForRequest(r, settings, body.TrustedForever)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to create auth session")
		return
	}
	writeAuthCookie(w, session)
	writeJSON(w, http.StatusOK, authStatusFromSettings(settings, &session))
}

func (h *Handler) handleChangePIN(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "security storage is not available")
		return
	}

	var body contracts.ChangePINRequest
	if err := decodeAuthJSONBody(r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	body.CurrentPIN = strings.TrimSpace(body.CurrentPIN)
	body.NewPIN = strings.TrimSpace(body.NewPIN)
	body.ConfirmPIN = strings.TrimSpace(body.ConfirmPIN)
	if err := validatePIN(body.NewPIN); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	if body.NewPIN != body.ConfirmPIN {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "pin confirmation does not match")
		return
	}
	ok, err := h.store.VerifyAppPIN(r.Context(), body.CurrentPIN)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to verify pin")
		return
	}
	if !ok {
		writeAppError(w, http.StatusUnauthorized, contracts.ErrorCodeAuthInvalidPIN, "Current PIN is incorrect")
		return
	}
	if err := h.store.SetAppPIN(r.Context(), body.NewPIN); err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to set pin")
		return
	}
	settings, err := h.store.GetAppSecuritySettings(r.Context())
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read security settings")
		return
	}
	session, _, err := h.authSessionFromRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate auth session")
		return
	}
	writeJSON(w, http.StatusOK, authStatusFromSettings(settings, &session))
}

func (h *Handler) handleLockPIN(w http.ResponseWriter, r *http.Request) {
	if h.store != nil {
		if cookie, err := r.Cookie(authCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			_ = h.store.RevokeAuthSession(r.Context(), cookie.Value)
		}
	}
	clearAuthCookie(w)
	dto, err := h.authStatusForRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read auth status")
		return
	}
	dto.Unlocked = !dto.PINEnabled
	dto.SessionExpiresAt = ""
	dto.TrustedForever = false
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePatchAuthSettings(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "security storage is not available")
		return
	}
	var body contracts.PatchAuthSettingsRequest
	if err := decodeAuthJSONBody(r, &body); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	settings, err := h.store.PatchAppSecuritySettings(r.Context(), storage.AppSecuritySettingsPatch{
		PINEnabled:        body.PINEnabled,
		SessionTTLMinutes: body.SessionTTLMinutes,
		LANRequiresPIN:    body.LANRequiresPIN,
		LockOnRestart:     body.LockOnRestart,
	})
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update auth settings")
		return
	}
	session, _, err := h.authSessionFromRequest(r)
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to validate auth session")
		return
	}
	writeJSON(w, http.StatusOK, authStatusFromSettings(settings, &session))
}

func (h *Handler) authStatusForRequest(r *http.Request) (contracts.AuthStatusDTO, error) {
	if h.store == nil {
		return contracts.AuthStatusDTO{
			PINEnabled:        false,
			Unlocked:          true,
			SetupRequired:     true,
			PINLength:         0,
			SessionTTLMinutes: 60,
			LANRequiresPIN:    true,
			LockOnRestart:     true,
		}, nil
	}
	settings, err := h.store.GetAppSecuritySettings(r.Context())
	if err != nil {
		return contracts.AuthStatusDTO{}, err
	}
	session, ok, err := h.authSessionFromRequest(r)
	if err != nil {
		return contracts.AuthStatusDTO{}, err
	}
	if !ok {
		return authStatusFromSettings(settings, nil), nil
	}
	return authStatusFromSettings(settings, &session), nil
}

func (h *Handler) authSessionFromRequest(r *http.Request) (storage.AuthSession, bool, error) {
	if h.store == nil || r == nil {
		return storage.AuthSession{}, false, nil
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return storage.AuthSession{}, false, nil
	}
	return h.store.GetValidAuthSession(r.Context(), strings.TrimSpace(cookie.Value), time.Now().UTC())
}

func (h *Handler) createAuthSessionForRequest(r *http.Request, settings storage.AppSecuritySettings, trustedForever bool) (storage.AuthSession, error) {
	clientKey, ip := authClientKey(r)
	return h.store.CreateAuthSession(r.Context(), storage.CreateAuthSessionInput{
		ClientKey:      clientKey,
		UserAgent:      r.UserAgent(),
		IP:             ip,
		TTLMinutes:     settings.SessionTTLMinutes,
		TrustedForever: trustedForever,
		Now:            time.Now().UTC(),
	})
}

func authStatusFromSettings(settings storage.AppSecuritySettings, session *storage.AuthSession) contracts.AuthStatusDTO {
	unlocked := !settings.PINEnabled || session != nil
	dto := contracts.AuthStatusDTO{
		PINEnabled:        settings.PINEnabled,
		Unlocked:          unlocked,
		SetupRequired:     !settings.PINEnabled,
		PINLength:         authStatusPINLength(settings),
		SessionTTLMinutes: settings.SessionTTLMinutes,
		LANRequiresPIN:    settings.LANRequiresPIN,
		LockOnRestart:     settings.LockOnRestart,
	}
	if session != nil {
		dto.SessionExpiresAt = session.ExpiresAt
		dto.TrustedForever = session.TrustedForever
	}
	return dto
}

func authStatusPINLength(settings storage.AppSecuritySettings) int {
	if !settings.PINEnabled || settings.PINLength < 4 || settings.PINLength > 8 {
		return 0
	}
	return settings.PINLength
}

func decodeAuthJSONBody(r *http.Request, out any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(out)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func validatePIN(pin string) error {
	if len(pin) < 4 || len(pin) > 8 {
		return errors.New("pin must be 4-8 digits")
	}
	for _, ch := range pin {
		if ch < '0' || ch > '9' {
			return errors.New("pin must be numeric")
		}
	}
	return nil
}

func authClientKey(r *http.Request) (clientKey string, ip string) {
	if r == nil {
		return "unknown", ""
	}
	ip = strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = strings.Trim(host, "[]")
	} else {
		ip = strings.Trim(ip, "[]")
	}
	ua := strings.TrimSpace(r.UserAgent())
	if ua == "" {
		ua = "Unknown Client"
	}
	sum := sha256.Sum256([]byte(ip + "\x00" + ua))
	return hex.EncodeToString(sum[:16]), ip
}

func writeAuthCookie(w http.ResponseWriter, session storage.AuthSession) {
	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if session.TrustedForever {
		cookie.MaxAge = trustedForeverCookieMaxAge
		cookie.Expires = time.Now().UTC().Add(time.Duration(trustedForeverCookieMaxAge) * time.Second)
	}
	http.SetCookie(w, cookie)
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}
