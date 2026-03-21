package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"jav-shadcn/backend/internal/scraper"
)

// ErrMovieNotFoundForMetadata is returned when SaveMovieMetadata updates zero rows (unknown movie id).
var ErrMovieNotFoundForMetadata = errors.New("movie not found for metadata update")

func (s *SQLiteStore) SaveMovieMetadata(ctx context.Context, metadata scraper.Metadata) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.ExecContext(
		ctx,
		`UPDATE movies SET
			title = ?,
			studio = ?,
			summary = ?,
			runtime_minutes = ?,
			rating = ?,
			year = CASE WHEN ? != '' THEN CAST(substr(?, 1, 4) AS INTEGER) ELSE year END,
			provider = ?,
			homepage = ?,
			director = ?,
			release_date = ?,
			cover_url = ?,
			thumb_url = ?,
			preview_video_url = ?,
			updated_at = ?
		WHERE id = ?`,
		coalesce(metadata.Title, metadata.Number),
		coalesce(metadata.Studio, "Unknown"),
		coalesce(metadata.Summary, "Metadata pending scrape."),
		metadata.RuntimeMinutes,
		metadata.Rating,
		metadata.ReleaseDate,
		metadata.ReleaseDate,
		metadata.Provider,
		metadata.Homepage,
		metadata.Director,
		metadata.ReleaseDate,
		metadata.CoverURL,
		metadata.ThumbURL,
		metadata.PreviewVideoURL,
		nowUTC(),
		metadata.MovieID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w (id=%s)", ErrMovieNotFoundForMetadata, metadata.MovieID)
	}

	if err := replaceMovieActors(ctx, tx, metadata.MovieID, metadata.Actors); err != nil {
		return err
	}
	if err := replaceMovieTags(ctx, tx, metadata.MovieID, metadata.Tags); err != nil {
		return err
	}
	if err := replaceMediaAssets(ctx, tx, metadata); err != nil {
		return err
	}

	return tx.Commit()
}

func replaceMovieActors(ctx context.Context, tx *sql.Tx, movieID string, actors []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_actors WHERE movie_id = ?`, movieID); err != nil {
		return err
	}

	for _, actor := range actors {
		actor = strings.TrimSpace(actor)
		if actor == "" {
			continue
		}

		actorID, err := ensureActor(ctx, tx, actor)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT OR IGNORE INTO movie_actors (movie_id, actor_id) VALUES (?, ?)`,
			movieID,
			actorID,
		); err != nil {
			return err
		}
	}

	return nil
}

// replaceMovieMetadataTagsTx verifies the movie exists, then replaces only nfo-type tag links (same as scraper write path).
func replaceMovieMetadataTagsTx(ctx context.Context, tx *sql.Tx, movieID string, tags []string) error {
	var one int
	switch err := tx.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id = ?`, movieID).Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return ErrMovieNotFoundForPatch
	case err != nil:
		return err
	}
	return replaceMovieTags(ctx, tx, movieID, tags)
}

func replaceMovieTags(ctx context.Context, tx *sql.Tx, movieID string, tags []string) error {
	// Only replace scraper/metadata (nfo) links; user tags stay on movie_tags.
	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_tags WHERE movie_id = ? AND tag_id IN (SELECT id FROM tags WHERE type = 'nfo')`, movieID); err != nil {
		return err
	}

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		tagID, err := ensureTag(ctx, tx, tag)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT OR IGNORE INTO movie_tags (movie_id, tag_id) VALUES (?, ?)`,
			movieID,
			tagID,
		); err != nil {
			return err
		}
	}

	return nil
}

func replaceMediaAssets(ctx context.Context, tx *sql.Tx, metadata scraper.Metadata) error {
	if metadata.CoverURL != "" {
		if err := upsertMediaAsset(ctx, tx, metadata.MovieID+":cover", metadata.MovieID, "cover", metadata.CoverURL); err != nil {
			return err
		}
	}
	if metadata.ThumbURL != "" {
		if err := upsertMediaAsset(ctx, tx, metadata.MovieID+":thumb", metadata.MovieID, "thumb", metadata.ThumbURL); err != nil {
			return err
		}
	}
	for index, previewURL := range metadata.PreviewImages {
		if previewURL == "" {
			continue
		}
		if err := upsertMediaAsset(ctx, tx, fmt.Sprintf("%s:preview:%02d", metadata.MovieID, index+1), metadata.MovieID, "preview_image", previewURL); err != nil {
			return err
		}
	}
	return nil
}

func upsertMediaAsset(ctx context.Context, tx *sql.Tx, id, movieID, assetType, sourceURL string) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO media_assets (id, movie_id, type, source_url, local_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, '', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_url = excluded.source_url,
			updated_at = excluded.updated_at`,
		id,
		movieID,
		assetType,
		sourceURL,
		nowUTC(),
		nowUTC(),
	)
	return err
}

func ensureActor(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO actors (name, avatar) VALUES (?, '')`, name)
	if err != nil {
		return 0, err
	}
	return lookupEntityID(ctx, tx, `SELECT id FROM actors WHERE name = ?`, name)
}

func ensureTag(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name, type) VALUES (?, 'nfo')`, name)
	if err != nil {
		return 0, err
	}
	return lookupTagID(ctx, tx, name, "nfo")
}

func lookupTagID(ctx context.Context, tx *sql.Tx, name, tagType string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE name = ? AND type = ?`, name, tagType).Scan(&id)
	switch {
	case err == nil:
		return id, nil
	case errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("tag not found after insert: %s (%s)", name, tagType)
	default:
		return 0, err
	}
}

func lookupEntityID(ctx context.Context, tx *sql.Tx, query string, value string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, query, value).Scan(&id)
	switch {
	case err == nil:
		return id, nil
	case errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("entity not found after insert: %s", value)
	default:
		return 0, err
	}
}

func coalesce(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
