package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/movieimages"
)

func imageVersion(info os.FileInfo) string {
	return strconv.FormatInt(info.ModTime().UTC().UnixNano(), 10) + "-" + strconv.FormatInt(info.Size(), 10)
}

func (h *Handler) readyMovieVariants(movieID, kind, version string) []contracts.ImageVariantDTO {
	if h.movieImages == nil || version == "" {
		return nil
	}
	source := posterLocalAPIPath(movieID, kind, version)
	h.movieImages.Ensure(source, func(ctx context.Context) (*os.File, error) {
		var f *os.File
		var err error
		if strings.HasPrefix(kind, "preview/") {
			seq, _ := strconv.Atoi(strings.TrimPrefix(kind, "preview/"))
			f, err = h.store.OpenMoviePreviewImageFile(ctx, movieID, seq, h.cfg.CacheDir)
		} else {
			f, err = h.store.OpenMovieAssetFile(ctx, movieID, kind, h.cfg.CacheDir)
		}
		if err != nil {
			return nil, err
		}
		info, err := f.Stat()
		if err != nil || imageVersion(info) != version {
			_ = f.Close()
			return nil, fmt.Errorf("image source revision changed")
		}
		return f, nil
	})
	var variants []contracts.ImageVariantDTO
	for _, width := range movieimages.Widths {
		if _, ready := h.movieImages.Ready(source, width); ready {
			variants = append(variants, contracts.ImageVariantDTO{Width: width, URL: source + "&width=" + strconv.Itoa(width)})
		}
	}
	return variants
}

func (h *Handler) enrichPosterVariants(item *contracts.MovieListItemDTO) {
	for _, asset := range []struct {
		url    string
		target *[]contracts.ImageVariantDTO
	}{
		{item.CoverURL, &item.CoverVariants}, {item.ThumbURL, &item.ThumbVariants},
	} {
		u, err := url.Parse(asset.url)
		if err != nil || !strings.HasPrefix(u.Path, "/api/library/movies/") {
			continue
		}
		kind := u.Path[strings.LastIndex(u.Path, "/")+1:]
		*asset.target = h.readyMovieVariants(item.ID, kind, u.Query().Get("v"))
	}
}

// Missing derivatives redirect without caching the original under a variant URL.
// Source files have already passed the normal asset permission checks.
func (h *Handler) serveMovieImageVariant(w http.ResponseWriter, r *http.Request, movieID, kind string, info os.FileInfo) bool {
	raw, present := r.URL.Query()["width"]
	if !present {
		return false
	}
	width := 0
	if len(raw) == 1 {
		width, _ = strconv.Atoi(raw[0])
	}
	if !movieimages.ValidWidth(width) {
		writeAppError(w, 400, contracts.ErrorCodeBadRequest, "unsupported image width")
		return true
	}
	version := imageVersion(info)
	source := posterLocalAPIPath(movieID, kind, version)
	h.readyMovieVariants(movieID, kind, version)
	if h.movieImages != nil {
		if path, ready := h.movieImages.Ready(source, width); ready {
			f, err := os.Open(path)
			if err == nil {
				defer f.Close()
				st, err := f.Stat()
				if err == nil {
					setReusableLocalImageCacheHeaders(w)
					if r.URL.Query().Get("v") != version {
						w.Header().Set("Cache-Control", "private, no-store")
					}
					w.Header().Set("X-Curated-Image-Cache", "hit")
					http.ServeContent(w, r, "derived.img", st.ModTime(), f)
					h.movieImages.Touch(path)
					return true
				}
			}
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Curated-Image-Cache", "pending")
	http.Redirect(w, r, source, http.StatusTemporaryRedirect)
	return true
}
