package storage

import (
	"context"
	"fmt"
)

// LibraryHealthMovieRecord contains the persisted fields needed by a read-only health scan.
type LibraryHealthMovieRecord struct {
	ID       string
	Title    string
	Code     string
	Location string
	Provider string
	Summary  string
	CoverURL string
	ThumbURL string
}

// LibraryHealthAssetRecord contains one registered local movie asset.
type LibraryHealthAssetRecord struct {
	ID             string
	MovieID        string
	Type           string
	LocalPath      string
	LastHTTPStatus int
	LastError      string
}

// LibraryHealthActorRecord contains avatar state for an actor used by an active movie.
type LibraryHealthActorRecord struct {
	ID              int64
	Name            string
	AvatarURL       string
	AvatarLocalPath string
	AvatarLastError string
}

// LibraryHealthOrphanRecord describes active user state whose parent row is missing.
type LibraryHealthOrphanRecord struct {
	TableName string
	Key       string
}

// LibraryHealthForeignKeyViolation is one PRAGMA foreign_key_check row.
type LibraryHealthForeignKeyViolation struct {
	TableName    string
	RowID        string
	ParentTable  string
	ForeignKeyID int
}

// LibraryHealthSnapshot is the database-only input for filesystem-aware health scanning.
type LibraryHealthSnapshot struct {
	Movies               []LibraryHealthMovieRecord
	Assets               []LibraryHealthAssetRecord
	Actors               []LibraryHealthActorRecord
	MetadataAttempts     []MovieMetadataScrapeAttempt
	Orphans              []LibraryHealthOrphanRecord
	QuickCheckMessages   []string
	ForeignKeyViolations []LibraryHealthForeignKeyViolation
	UploadSessionIDs     map[string]struct{}
}

// InspectLibraryHealth gathers read-only database facts. It performs no repair or mutation.
func (s *SQLiteStore) InspectLibraryHealth(ctx context.Context) (LibraryHealthSnapshot, error) {
	var out LibraryHealthSnapshot

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, code, location, provider, summary, cover_url, thumb_url
		FROM movies
		WHERE trashed_at IS NULL OR TRIM(trashed_at) = ''
		ORDER BY id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var row LibraryHealthMovieRecord
		if err := rows.Scan(&row.ID, &row.Title, &row.Code, &row.Location, &row.Provider, &row.Summary, &row.CoverURL, &row.ThumbURL); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.Movies = append(out.Movies, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	rows, err = s.db.QueryContext(ctx, `
		SELECT id, movie_id, type, local_path, last_http_status, last_error
		FROM media_assets
		ORDER BY movie_id, type, id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var row LibraryHealthAssetRecord
		if err := rows.Scan(&row.ID, &row.MovieID, &row.Type, &row.LocalPath, &row.LastHTTPStatus, &row.LastError); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.Assets = append(out.Assets, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	rows, err = s.db.QueryContext(ctx, `
		SELECT DISTINCT a.id, a.name, a.avatar, a.avatar_local_path, a.avatar_last_error
		FROM actors a
		JOIN movie_actors ma ON ma.actor_id = a.id
		JOIN movies m ON m.id = ma.movie_id
		WHERE m.trashed_at IS NULL OR TRIM(m.trashed_at) = ''
		ORDER BY a.name, a.id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var row LibraryHealthActorRecord
		if err := rows.Scan(&row.ID, &row.Name, &row.AvatarURL, &row.AvatarLocalPath, &row.AvatarLastError); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.Actors = append(out.Actors, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	rows, err = s.db.QueryContext(ctx, `
		SELECT movie_id, task_id, status, error_code, error_category, error_message,
			provider, started_at, finished_at, updated_at
		FROM movie_metadata_scrape_attempts
		WHERE status = 'failed'
		ORDER BY updated_at DESC, movie_id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var row MovieMetadataScrapeAttempt
		if err := rows.Scan(&row.MovieID, &row.TaskID, &row.Status, &row.ErrorCode, &row.ErrorCategory,
			&row.ErrorMessage, &row.Provider, &row.StartedAt, &row.FinishedAt, &row.UpdatedAt); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.MetadataAttempts = append(out.MetadataAttempts, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	orphanQueries := []struct {
		table string
		query string
	}{
		{"playback_progress", `SELECT movie_id FROM playback_progress WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_progress.movie_id)`},
		{"library_movie_comments", `SELECT movie_id FROM library_movie_comments WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_movie_comments.movie_id)`},
		{"library_played_movies", `SELECT movie_id FROM library_played_movies WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_played_movies.movie_id)`},
		{"playback_daily_watch_time", `SELECT day_key || ':' || movie_id FROM playback_daily_watch_time WHERE NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_daily_watch_time.movie_id)`},
	}
	for _, spec := range orphanQueries {
		rows, err = s.db.QueryContext(ctx, spec.query)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				_ = rows.Close()
				return out, err
			}
			out.Orphans = append(out.Orphans, LibraryHealthOrphanRecord{TableName: spec.table, Key: key})
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return out, err
		}
		if err := rows.Close(); err != nil {
			return out, err
		}
	}

	rows, err = s.db.QueryContext(ctx, `PRAGMA quick_check`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.QuickCheckMessages = append(out.QuickCheckMessages, message)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	rows, err = s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var row LibraryHealthForeignKeyViolation
		var rowID any
		if err := rows.Scan(&row.TableName, &rowID, &row.ParentTable, &row.ForeignKeyID); err != nil {
			_ = rows.Close()
			return out, err
		}
		row.RowID = fmt.Sprint(rowID)
		out.ForeignKeyViolations = append(out.ForeignKeyViolations, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}

	out.UploadSessionIDs, err = s.ListMovieImportUploadSessionIDs(ctx)
	if err != nil {
		return out, err
	}
	return out, nil
}
