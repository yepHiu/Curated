package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func actorAvatarLocalAPIPath(name string) string {
	return "/api/library/actors/" + url.PathEscape(name) + "/asset/avatar"
}

func (h *Handler) enrichActorProfileLocalAvatar(ctx context.Context, profile *contracts.ActorProfileDTO) {
	if profile == nil || strings.TrimSpace(profile.Name) == "" {
		return
	}
	ready, err := h.store.BatchActorAvatarLocalReady(ctx, []string{profile.Name}, h.cfg.CacheDir)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("actor avatar probe failed", zap.Error(err), zap.String("actorName", profile.Name))
		}
		return
	}
	if ready[profile.Name] {
		profile.HasLocalAvatar = true
		profile.AvatarLocalURL = actorAvatarLocalAPIPath(profile.Name)
		profile.AvatarURL = profile.AvatarLocalURL
	}
}

func (h *Handler) enrichActorListLocalAvatars(ctx context.Context, items []contracts.ActorListItemDTO) {
	if len(items) == 0 {
		return
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}
	ready, err := h.store.BatchActorAvatarLocalReady(ctx, names, h.cfg.CacheDir)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("batch actor avatar probe failed", zap.Error(err))
		}
		return
	}
	for i := range items {
		if !ready[items[i].Name] {
			continue
		}
		items[i].HasLocalAvatar = true
		items[i].AvatarLocalURL = actorAvatarLocalAPIPath(items[i].Name)
		items[i].AvatarURL = items[i].AvatarLocalURL
	}
}

func (h *Handler) enrichMovieDetailLocalActorAvatars(ctx context.Context, movie *contracts.MovieDetailDTO) {
	if movie == nil || len(movie.ActorAvatarURLs) == 0 {
		return
	}
	ready, err := h.store.BatchActorAvatarLocalReady(ctx, movie.Actors, h.cfg.CacheDir)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("movie actor avatar probe failed", zap.Error(err), zap.String("movieId", movie.ID))
		}
		return
	}
	for name := range movie.ActorAvatarURLs {
		if ready[name] {
			movie.ActorAvatarURLs[name] = actorAvatarLocalAPIPath(name)
		}
	}
}

func (h *Handler) handleGetActorAvatarAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid actor avatar request")
		return
	}
	f, err := h.store.OpenActorAvatarFile(r.Context(), name, h.cfg.CacheDir)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieAssetNotFound), errors.Is(err, storage.ErrMovieAssetForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "asset not found")
		default:
			if h.logger != nil {
				h.logger.Warn("open actor avatar failed", zap.Error(err), zap.String("actorName", name))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open asset")
		}
		return
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read asset")
		return
	}
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, "avatar"+pickImageExtFromPath(f.Name()), st.ModTime(), f)
}
