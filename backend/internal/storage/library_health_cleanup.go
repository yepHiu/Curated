package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// LibraryHealthCleanupAudit is one confirmed orphan-state cleanup result.
type LibraryHealthCleanupAudit struct {
	ID          int64
	TaskID      string
	FindingID   string
	Action      string
	EntityType  string
	EntityID    string
	Outcome     string
	Diagnostic  string
	CreatedAt   string
	CompletedAt string
}

// CleanupOrphanUserState conditionally deletes a still-orphaned whitelisted row and writes an audit atomically.
func (s *SQLiteStore) CleanupOrphanUserState(
	ctx context.Context, taskID, findingID, tableName, key string, now time.Time,
) (bool, error) {
	tableName = strings.TrimSpace(tableName)
	key = strings.TrimSpace(key)
	if taskID == "" || findingID == "" || key == "" {
		return false, fmt.Errorf("cleanup identity is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	removed, diagnostic, err := deleteWhitelistedOrphanUserState(ctx, tx, tableName, key)
	if err != nil {
		return false, err
	}
	outcome := "skipped"
	if removed {
		outcome = "removed"
	}
	timestamp := now.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO library_health_cleanup_audits (
			task_id, finding_id, action, entity_type, entity_id, outcome, diagnostic, created_at, completed_at
		) VALUES (?, ?, 'cleanup_orphan_state', ?, ?, ?, ?, ?, ?)`,
		taskID, findingID, tableName, key, outcome, diagnostic, timestamp, timestamp); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return removed, nil
}

func deleteWhitelistedOrphanUserState(ctx context.Context, tx *sql.Tx, tableName, key string) (bool, string, error) {
	var (
		result sql.Result
		err    error
	)
	switch tableName {
	case "playback_progress":
		result, err = tx.ExecContext(ctx, `
			DELETE FROM playback_progress
			WHERE movie_id = ? AND NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_progress.movie_id)`, key)
	case "library_movie_comments":
		result, err = tx.ExecContext(ctx, `
			DELETE FROM library_movie_comments
			WHERE movie_id = ? AND NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_movie_comments.movie_id)`, key)
	case "library_played_movies":
		result, err = tx.ExecContext(ctx, `
			DELETE FROM library_played_movies
			WHERE movie_id = ? AND NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = library_played_movies.movie_id)`, key)
	case "playback_daily_watch_time":
		dayKey, movieID, ok := strings.Cut(key, ":")
		if !ok || strings.TrimSpace(dayKey) == "" || strings.TrimSpace(movieID) == "" {
			return false, "invalid daily watch-time orphan key", nil
		}
		result, err = tx.ExecContext(ctx, `
			DELETE FROM playback_daily_watch_time
			WHERE day_key = ? AND movie_id = ?
			  AND NOT EXISTS (SELECT 1 FROM movies WHERE movies.id = playback_daily_watch_time.movie_id)`, dayKey, movieID)
	default:
		return false, "entity type is not in the orphan cleanup whitelist", nil
	}
	if err != nil {
		return false, "", err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, "", err
	}
	if affected == 1 {
		return true, "confirmed orphan user state removed", nil
	}
	return false, "row no longer exists or is no longer orphaned", nil
}

func (s *SQLiteStore) ListLibraryHealthCleanupAudits(ctx context.Context, limit int) ([]LibraryHealthCleanupAudit, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, finding_id, action, entity_type, entity_id, outcome,
			diagnostic, created_at, completed_at
		FROM library_health_cleanup_audits ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LibraryHealthCleanupAudit, 0)
	for rows.Next() {
		var audit LibraryHealthCleanupAudit
		if err := rows.Scan(&audit.ID, &audit.TaskID, &audit.FindingID, &audit.Action,
			&audit.EntityType, &audit.EntityID, &audit.Outcome, &audit.Diagnostic,
			&audit.CreatedAt, &audit.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, audit)
	}
	return out, rows.Err()
}
