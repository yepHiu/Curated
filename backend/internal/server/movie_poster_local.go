package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func posterLocalAPIPath(movieID, kind string) string {
	return "/api/library/movies/" + url.PathEscape(movieID) + "/asset/" + kind
}

func applyLocalPosterURLs(item *contracts.MovieListItemDTO, flags storage.MoviePosterLocalFlags) {
	if flags.Thumb {
		item.ThumbURL = posterLocalAPIPath(item.ID, "thumb")
	} else if flags.Cover {
		item.ThumbURL = posterLocalAPIPath(item.ID, "cover")
	}

	if flags.Cover {
		item.CoverURL = posterLocalAPIPath(item.ID, "cover")
	} else if flags.Thumb {
		item.CoverURL = posterLocalAPIPath(item.ID, "thumb")
	}
}

func (h *Handler) enrichMovieListItemsLocalPosters(ctx context.Context, items []contracts.MovieListItemDTO) {
	if len(items) == 0 {
		return
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	ready, err := h.store.BatchMoviePosterLocalReady(ctx, ids, h.cfg.CacheDir)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("batch local poster probe failed", zap.Error(err))
		}
		return
	}
	for i := range items {
		flags := ready[items[i].ID]
		if !flags.Cover && !flags.Thumb {
			continue
		}
		applyLocalPosterURLs(&items[i], flags)
	}
}

func (h *Handler) enrichMovieDetailLocalPosters(ctx context.Context, movie *contracts.MovieDetailDTO) {
	flags, err := h.store.BatchMoviePosterLocalReady(ctx, []string{movie.ID}, h.cfg.CacheDir)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("local poster probe failed", zap.String("movieId", movie.ID), zap.Error(err))
		}
	} else {
		f := flags[movie.ID]
		if f.Cover || f.Thumb {
			applyLocalPosterURLs(&movie.MovieListItemDTO, f)
		}
	}
	if len(movie.PreviewImages) > 0 {
		movie.PreviewImages = h.store.RewritePreviewImageURLsPreferLocal(ctx, movie.ID, h.cfg.CacheDir, movie.PreviewImages)
	}
	h.enrichMovieDetailLocalActorAvatars(ctx, movie)
}

func (h *Handler) handleGetMovieAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	kind := strings.TrimSpace(strings.ToLower(r.PathValue("kind")))
	if movieID == "" || (kind != "cover" && kind != "thumb") {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid movie asset request")
		return
	}

	f, err := h.store.OpenMovieAssetFile(r.Context(), movieID, kind, h.cfg.CacheDir)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieAssetNotFound), errors.Is(err, storage.ErrMovieAssetForbidden):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "asset not found")
		default:
			if h.logger != nil {
				h.logger.Warn("open movie asset failed", zap.Error(err), zap.String("movieId", movieID), zap.String("kind", kind))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open asset")
		}
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("stat movie asset failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read asset")
		return
	}

	name := kind + pickImageExtFromPath(f.Name())
	// Same URL after rescrape must revalidate: browsers otherwise keep stale bytes (paths are stable).
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, name, st.ModTime(), f)
}

func (h *Handler) handleGetMoviePreviewAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	movieID := strings.TrimSpace(r.PathValue("movieId"))
	idxStr := strings.TrimSpace(r.PathValue("index"))
	if movieID == "" || idxStr == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid preview asset request")
		return
	}
	seq, err := strconv.Atoi(idxStr)
	if err != nil || seq < 1 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid preview index")
		return
	}

	f, err := h.store.OpenMoviePreviewImageFile(r.Context(), movieID, seq, h.cfg.CacheDir)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrMovieAssetNotFound), errors.Is(err, storage.ErrMovieAssetForbidden):
			if h.proxyMoviePreviewAsset(w, r, movieID, seq) {
				return
			}
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "asset not found")
		default:
			if h.logger != nil {
				h.logger.Warn("open movie preview asset failed", zap.Error(err), zap.String("movieId", movieID), zap.Int("seq", seq))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to open asset")
		}
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("stat movie preview asset failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to read asset")
		return
	}

	name := "preview-" + strconv.Itoa(seq) + pickImageExtFromPath(f.Name())
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, name, st.ModTime(), f)
}

func (h *Handler) proxyMoviePreviewAsset(w http.ResponseWriter, r *http.Request, movieID string, seq int) bool {
	sourceURL, refererURL, err := h.store.GetMoviePreviewSourceURL(r.Context(), movieID, seq)
	if err != nil || strings.TrimSpace(sourceURL) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	if strings.TrimSpace(refererURL) != "" {
		req.Header.Set("Referer", refererURL)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	w.Header().Set("Cache-Control", "private, no-cache")
	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
	return true
}

func pickImageExtFromPath(p string) string {
	p = strings.ToLower(p)
	for _, ext := range []string{".webp", ".jpg", ".jpeg", ".png", ".gif"} {
		if strings.HasSuffix(p, ext) {
			return ext
		}
	}
	return ""
}
