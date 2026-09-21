package app

import (
	"context"
	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
	"curated-backend/internal/wishlistassets"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// startWishlistWorker 创建单个后台处理器，关闭时取消且等待其退出。
func (a *App) startWishlistWorker() {
	ctx, cancel := context.WithCancel(a.appCtx)
	a.wishlistCancel = cancel
	a.wishlistDone = make(chan struct{})
	go func() { // 独立生命周期持续消费持久队列，不依赖插件或 HTTP 连接。
		defer close(a.wishlistDone)
		timer := time.NewTicker(2 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				item, attempt, e := a.store.ClaimWishlistJob(ctx)
				if errors.Is(e, sql.ErrNoRows) {
					continue
				}
				if e != nil {
					a.logger.Warn("wishlist queue", zap.Error(e))
					continue
				}
				a.enrichWishlist(ctx, item, attempt)
			}
		}
	}()
}

// wishlistMetadata 从刮削结果提取可展示事实，保持用户备注独立。
func wishlistMetadata(m scraper.Metadata) contracts.WishlistMetadata {
	return contracts.WishlistMetadata{Title: m.Title, Summary: m.Summary, Actors: m.Actors, Tags: m.Tags, Studio: m.Studio, ReleaseDate: m.ReleaseDate, RuntimeMinutes: m.RuntimeMinutes, Provider: m.Provider, Homepage: m.Homepage}
}

// enrichWishlist 逐步提交资料和图片，失败只影响当前阶段。
func (a *App) enrichWishlist(parent context.Context, item contracts.WishlistItemDTO, attempt int) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Minute)
	defer cancel()
	// finish 即使业务超时也以独立短上下文写回任务结果。
	finish := func(state, message string, retry bool) {
		saveCtx, stop := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
		defer stop()
		if e := a.store.FinishWishlistJob(saveCtx, item.ID, item.Generation, attempt, state, message, retry); e != nil {
			a.logger.Warn("finish wishlist job", zap.Error(e))
		}
	}
	root, e := a.store.WishlistAssetRoot()
	if e != nil {
		finish("failed", "asset_storage_unavailable", false)
		return
	}
	if e = a.store.ReconcileWishlist(ctx); e != nil {
		finish("failed", "library_lookup_failed", true)
		return
	}
	fresh, e := a.store.GetWishlist(ctx, item.ID)
	if e != nil || fresh.Generation != item.Generation {
		return
	}
	var metadata scraper.Metadata
	var localFiles []storage.WishlistAssetFile
	if len(fresh.MovieIDs) == 1 {
		movie, err := a.store.GetMovieDetail(ctx, fresh.MovieIDs[0])
		if err == nil && movie.MetadataProvider != "" {
			metadata = scraper.Metadata{Number: movie.Code, Title: movie.Title, Summary: movie.Summary, Actors: movie.Actors, Tags: movie.Tags, Studio: movie.Studio, RuntimeMinutes: movie.RuntimeMinutes, ReleaseDate: movie.ReleaseDate, Provider: movie.MetadataProvider, Homepage: movie.Homepage, CoverURL: movie.CoverURL, ThumbURL: movie.ThumbURL, PreviewImages: movie.PreviewImages}
			localFiles, _ = a.store.MovieFilesForWishlist(ctx, movie.ID, a.cfg.CacheDir)
		}
	}
	if metadata.Title == "" {
		provider, ok := a.scraper.(interface {
			ScrapeWishlist(context.Context, string, string, scraper.MovieScrapeOptions) (scraper.Metadata, error)
		})
		if !ok {
			finish("failed", "provider_unavailable", false)
			return
		}
		if e = a.acquireScrapeSlot(ctx); e != nil {
			finish("failed", "scrape_cancelled", true)
			return
		}
		metadata, e = provider.ScrapeWishlist(ctx, item.ID, item.Code, a.movieScrapeOptionsForRun())
		a.releaseScrapeSlot()
		if e != nil {
			state := "failed"
			reason := "metadata_fetch_failed"
			retry := true
			if strings.Contains(e.Error(), "WISHLIST_NOT_FOUND") {
				reason = "metadata_not_found"
				retry = false
			}
			if strings.Contains(e.Error(), "WISHLIST_AMBIGUOUS") || strings.Contains(e.Error(), "WISHLIST_IDENTITY_MISMATCH") {
				state = "needs_review"
				reason = "metadata_identity_uncertain"
				retry = false
			}
			finish(state, reason, retry)
			return
		}
	}
	if e = a.store.SaveWishlistMetadata(ctx, item.ID, item.Generation, wishlistMetadata(metadata)); e != nil {
		return
	}
	existing, _ := a.store.WishlistAssetFiles(ctx, item.ID)
	failures := 0
	count := 0
	specs := []struct {
		role, url string
		position  int
	}{{"thumb", metadata.ThumbURL, 0}, {"cover", metadata.CoverURL, 0}}
	for i, u := range metadata.PreviewImages {
		if i >= 20 {
			break
		}
		specs = append(specs, struct {
			role, url string
			position  int
		}{"preview_image", u, i})
	}
	for _, spec := range specs {
		if spec.url == "" {
			continue
		}
		count++
		if ctx.Err() != nil {
			failures++
			break
		}
		ready := false
		for _, file := range existing {
			if file.Role == spec.role && file.Position == spec.position && file.SourceURL == spec.url {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(file.Path))); err == nil {
					ready = true
					break
				}
			}
		}
		if ready {
			continue
		}
		var file wishlistassets.File
		copied := false
		for _, local := range localFiles {
			if local.Role == spec.role && local.Position == spec.position {
				b, err := os.ReadFile(local.Path)
				if err == nil {
					file, e = wishlistassets.Save(root, item.ID, b)
					copied = e == nil
				}
				break
			}
		}
		if !copied {
			file, e = wishlistassets.Download(ctx, root, item.ID, spec.url, a.Proxy())
		}
		if e != nil {
			failures++
			continue
		}
		e = a.store.SaveWishlistAsset(ctx, storage.WishlistAssetFile{ID: uuid.NewString(), ItemID: item.ID, Generation: item.Generation, Role: spec.role, Position: spec.position, SourceURL: spec.url, Path: file.Path, ThumbnailPath: file.ThumbnailPath, SHA256: file.Hash})
		if e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return
			}
			failures++
		}
	}
	if failures > 0 {
		finish("partial", "images_incomplete", attempt < 3)
		return
	}
	if count == 0 {
		finish("partial", "no_images_available", false)
		return
	}
	finish("ready", "", false)
}
