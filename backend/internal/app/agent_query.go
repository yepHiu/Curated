package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
)

func (a *App) AgentLibraryOverview(ctx context.Context) (map[string]any, error) {
	movies, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 1})
	if err != nil {
		return nil, err
	}
	trash, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{Mode: "trash", Limit: 1})
	if err != nil {
		return nil, err
	}
	frames, err := a.store.CountCuratedFrames(ctx)
	if err != nil {
		return nil, err
	}
	paths, err := a.store.ListLibraryPaths(ctx)
	if err != nil {
		return nil, err
	}
	pathRows := make([]map[string]any, 0, len(paths))
	if statuses, statusErr := a.ListLibraryPathStorageStatus(ctx); statusErr == nil {
		byID := map[string]contracts.LibraryPathStorageStatusDTO{}
		for _, row := range statuses.Items {
			byID[row.LibraryPathID] = row
		}
		for _, path := range paths {
			item := map[string]any{"id": path.ID, "title": path.Title}
			if st, ok := byID[path.ID]; ok {
				item["status"] = st.Status
				item["canRescan"] = st.CanRescan
			}
			pathRows = append(pathRows, item)
		}
	} else {
		for _, path := range paths {
			pathRows = append(pathRows, map[string]any{"id": path.ID, "title": path.Title})
		}
	}
	overview := map[string]any{
		"movieCount":          movies.Total,
		"trashCount":          trash.Total,
		"curatedFrameCount":   frames,
		"libraryPathCount":    len(paths),
		"libraryPaths":        pathRows,
		"comicLibraryEnabled": a.ComicLibraryEnabled(),
		"photoLibraryEnabled": a.PhotoLibraryEnabled(),
	}
	if a.ComicLibraryEnabled() {
		comics, err := a.store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Limit: 1})
		if err != nil {
			return nil, err
		}
		overview["comicCount"] = comics.Total
	}
	if a.PhotoLibraryEnabled() {
		photos, err := a.store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Limit: 1})
		if err != nil {
			return nil, err
		}
		overview["photoCount"] = photos.Total
	}
	return overview, nil
}

func (a *App) ListMovies(ctx context.Context, req contracts.ListMoviesRequest) (contracts.MoviesPageDTO, error) {
	return a.store.ListMovies(ctx, req)
}

func (a *App) GetMovieDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error) {
	return a.store.GetMovieDetail(ctx, movieID)
}

func (a *App) GetMovieComment(ctx context.Context, movieID string) (contracts.MovieCommentDTO, error) {
	return a.store.GetMovieComment(ctx, movieID)
}

func (a *App) MovieExists(ctx context.Context, movieID string) (bool, error) {
	return a.store.MovieExists(ctx, movieID)
}

func (a *App) UpsertMovieComment(ctx context.Context, movieID, body string, expected ...string) (contracts.MovieCommentDTO, error) {
	return a.store.UpsertMovieComment(ctx, movieID, body, expected...)
}

func (a *App) PatchMovieDisplayOverrides(ctx context.Context, movieID string, patch contracts.PatchMovieInput) (contracts.MovieDetailDTO, error) {
	if err := a.store.PatchMovieUserPrefs(ctx, movieID, patch); err != nil {
		return contracts.MovieDetailDTO{}, err
	}
	return a.store.GetMovieDetail(ctx, movieID)
}

func (a *App) CreateSavedView(ctx context.Context, name string, filters contracts.SavedViewFiltersV1) (contracts.SavedViewDTO, error) {
	normalizedName, err := storage.NormalizeSavedViewName(name)
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	normalizedFilters, err := storage.NormalizeSavedViewFilters(filters)
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	id, err := newAgentSavedViewID()
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	return a.store.CreateSavedView(ctx, id, normalizedName, normalizedFilters, time.Now())
}

func newAgentSavedViewID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "view_" + hex.EncodeToString(random[:]), nil
}

func (a *App) GetPlaybackProgress(ctx context.Context, movieID string) (float64, float64, string, bool, error) {
	row, err := a.store.GetPlaybackProgress(ctx, movieID)
	if err != nil {
		return 0, 0, "", false, err
	}
	if row == nil {
		return 0, 0, "", false, nil
	}
	return row.PositionSec, row.DurationSec, row.UpdatedAt, true, nil
}

func (a *App) ListActors(ctx context.Context, req contracts.ListActorsRequest) (contracts.ListActorsResponse, error) {
	return a.store.ListActors(ctx, req)
}

func (a *App) GetActorProfile(ctx context.Context, name string) (contracts.ActorProfileDTO, error) {
	return a.store.GetActorProfile(ctx, name)
}

func (a *App) ListWatchHistory(ctx context.Context, limit int) ([]tools.WatchHistoryItem, error) {
	rows, err := a.store.ListPlaybackProgressByUpdatedDesc(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]tools.WatchHistoryItem, 0, len(rows))
	for _, row := range rows {
		item := tools.WatchHistoryItem{
			MovieID:     row.MovieID,
			PositionSec: row.PositionSec,
			DurationSec: row.DurationSec,
			UpdatedAt:   row.UpdatedAt,
		}
		if detail, err := a.store.GetMovieDetail(ctx, row.MovieID); err == nil {
			item.Title = detail.Title
			item.Code = detail.Code
		}
		out = append(out, item)
	}
	return out, nil
}

func (a *App) QueryCuratedFrames(ctx context.Context, q, actor, movieID, tag string, limit, offset int) (contracts.CuratedFramesListDTO, error) {
	page, err := a.store.QueryCuratedFrames(ctx, storage.CuratedFrameQuery{
		Query:   q,
		Actor:   actor,
		MovieID: movieID,
		Tag:     tag,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return contracts.CuratedFramesListDTO{}, err
	}
	items := make([]contracts.CuratedFrameItemDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, contracts.CuratedFrameItemDTO{
			ID:          item.ID,
			MovieID:     item.MovieID,
			Title:       item.Title,
			Code:        item.Code,
			Actors:      item.Actors,
			PositionSec: item.PositionSec,
			CapturedAt:  item.CapturedAt,
			Tags:        item.Tags,
		})
	}
	return contracts.CuratedFramesListDTO{Items: items, Total: page.Total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (a *App) CountCuratedFrames(ctx context.Context) (int, error) {
	return a.store.CountCuratedFrames(ctx)
}

func (a *App) GetTask(_ context.Context, taskID string) (contracts.TaskDTO, bool) {
	if a.tasks == nil {
		return contracts.TaskDTO{}, false
	}
	return a.tasks.Get(taskID)
}

func (a *App) SearchProviderTitles(ctx context.Context, keyword string, limit int) ([]tools.ProviderTitleHit, error) {
	if a == nil || a.scraper == nil {
		return nil, fmt.Errorf("metadata search is unavailable")
	}
	searcher, ok := a.scraper.(scraper.TitleSearcher)
	if !ok {
		return nil, fmt.Errorf("metadata search is unavailable")
	}
	hits, err := searcher.SearchTitles(ctx, keyword, limit)
	if err != nil {
		return nil, err
	}
	out := make([]tools.ProviderTitleHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, tools.ProviderTitleHit{
			Code:     hit.Number,
			Title:    hit.Title,
			Provider: hit.Provider,
			Homepage: hit.Homepage,
			Score:    hit.Score,
		})
	}
	return out, nil
}

func (a *App) FindLibraryMoviesByCodes(ctx context.Context, codes []string) (map[string]contracts.MovieListItemDTO, error) {
	if a == nil || a.store == nil {
		return map[string]contracts.MovieListItemDTO{}, nil
	}
	return a.store.FindActiveMoviesByCodes(ctx, codes)
}

// ListComicBooks lists comics for Agent search after the comic Beta gate.
func (a *App) ListComicBooks(ctx context.Context, req contracts.ListComicBooksRequest) (contracts.ComicBooksPageDTO, error) {
	if !a.ComicLibraryEnabled() {
		return contracts.ComicBooksPageDTO{}, tools.ErrComicLibraryDisabled
	}
	return a.store.ListComicBooks(ctx, req)
}

// GetComicBookDetail returns one comic for Agent after the comic Beta gate.
func (a *App) GetComicBookDetail(ctx context.Context, comicID string) (contracts.ComicBookDetailDTO, error) {
	if !a.ComicLibraryEnabled() {
		return contracts.ComicBookDetailDTO{}, tools.ErrComicLibraryDisabled
	}
	return a.store.GetComicBookDetail(ctx, comicID)
}

// GetComicComment returns the personal note for one comic.
func (a *App) GetComicComment(ctx context.Context, comicID string) (contracts.ComicCommentDTO, error) {
	if !a.ComicLibraryEnabled() {
		return contracts.ComicCommentDTO{}, tools.ErrComicLibraryDisabled
	}
	return a.store.GetComicComment(ctx, comicID)
}

// UpsertComicComment writes a comic note after comparing the previewed previous body.
func (a *App) UpsertComicComment(ctx context.Context, comicID, body string, expected ...string) (contracts.ComicCommentDTO, error) {
	if !a.ComicLibraryEnabled() {
		return contracts.ComicCommentDTO{}, tools.ErrComicLibraryDisabled
	}
	current, err := a.store.GetComicComment(ctx, comicID)
	if err != nil {
		return contracts.ComicCommentDTO{}, err
	}
	if len(expected) > 0 && current.Body != expected[0] {
		return contracts.ComicCommentDTO{}, storage.ErrAIWriteConflict
	}
	return a.store.UpsertComicComment(ctx, comicID, body)
}

// ListPhotoBooks lists photo books for Agent search after the photo Beta gate.
func (a *App) ListPhotoBooks(ctx context.Context, req contracts.ListPhotoBooksRequest) (contracts.PhotoBooksPageDTO, error) {
	if !a.PhotoLibraryEnabled() {
		return contracts.PhotoBooksPageDTO{}, tools.ErrPhotoLibraryDisabled
	}
	return a.store.ListPhotoBooks(ctx, req)
}

// GetPhotoBookDetail returns one photo book for Agent after the photo Beta gate.
func (a *App) GetPhotoBookDetail(ctx context.Context, photoID string) (contracts.PhotoBookDetailDTO, error) {
	if !a.PhotoLibraryEnabled() {
		return contracts.PhotoBookDetailDTO{}, tools.ErrPhotoLibraryDisabled
	}
	return a.store.GetPhotoBookDetail(ctx, photoID)
}

// GetPhotoComment returns the personal note for one photo book.
func (a *App) GetPhotoComment(ctx context.Context, photoID string) (contracts.PhotoCommentDTO, error) {
	if !a.PhotoLibraryEnabled() {
		return contracts.PhotoCommentDTO{}, tools.ErrPhotoLibraryDisabled
	}
	return a.store.GetPhotoComment(ctx, photoID)
}

// UpsertPhotoComment writes a photo note after comparing the previewed previous body.
func (a *App) UpsertPhotoComment(ctx context.Context, photoID, body string, expected ...string) (contracts.PhotoCommentDTO, error) {
	if !a.PhotoLibraryEnabled() {
		return contracts.PhotoCommentDTO{}, tools.ErrPhotoLibraryDisabled
	}
	current, err := a.store.GetPhotoComment(ctx, photoID)
	if err != nil {
		return contracts.PhotoCommentDTO{}, err
	}
	if len(expected) > 0 && current.Body != expected[0] {
		return contracts.PhotoCommentDTO{}, storage.ErrAIWriteConflict
	}
	return a.store.UpsertPhotoComment(ctx, photoID, body)
}

// PatchComicBook updates comic fields for Agent title writes after the comic Beta gate.
func (a *App) PatchComicBook(ctx context.Context, comicID string, patch contracts.PatchComicBookRequest) (contracts.ComicBookDetailDTO, error) {
	if !a.ComicLibraryEnabled() {
		return contracts.ComicBookDetailDTO{}, tools.ErrComicLibraryDisabled
	}
	return a.store.PatchComicBook(ctx, comicID, patch)
}

// PatchPhotoBook updates photo fields for Agent title writes after the photo Beta gate.
func (a *App) PatchPhotoBook(ctx context.Context, photoID string, patch contracts.PatchPhotoBookRequest) (contracts.PhotoBookDetailDTO, error) {
	if !a.PhotoLibraryEnabled() {
		return contracts.PhotoBookDetailDTO{}, tools.ErrPhotoLibraryDisabled
	}
	return a.store.PatchPhotoBook(ctx, photoID, patch)
}
