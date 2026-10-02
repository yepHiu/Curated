package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"curated-backend/internal/contracts"
)

// topicOrganizationCoverageTx streams a single snapshot, avoiding one query per
// movie and reusing the exact classification fingerprint (never user preferences).
func topicOrganizationCoverageTx(ctx context.Context, tx *sql.Tx, visit func(string, string, string)) error {
	rows, err := tx.QueryContext(ctx, `SELECT m.id,m.title,m.summary,
	 (SELECT json_group_array(name) FROM (SELECT t.name FROM tags t JOIN movie_tags mt ON mt.tag_id=t.id WHERE mt.movie_id=m.id AND t.type='nfo' ORDER BY t.name)),
	 COALESCE(a.input_fingerprint,''),COALESCE(a.outcome,'')
	 FROM movies m LEFT JOIN movie_topic_analysis a ON a.movie_id=m.id WHERE `+sqlMovieActiveClause)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, title, summary, tagsJSON, fingerprint, outcome string
		if err = rows.Scan(&id, &title, &summary, &tagsJSON, &fingerprint, &outcome); err != nil {
			return err
		}
		status := "unorganized"
		if fingerprint != "" {
			var tags []string
			if err = json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
				return err
			}
			status = "organized"
			if topicInputFingerprint(title, summary, tags) != fingerprint {
				status = "outdated"
			}
		}
		visit(id, status, outcome)
	}
	return rows.Err()
}

// TagOrganizationStats counts the live library, including analyzed no-match
// movies, but excluding trash, failed attempts and vocabulary-only progress.
func (s *SQLiteStore) TagOrganizationStats(ctx context.Context) (contracts.TagOrganizationStatsDTO, error) {
	stats := contracts.TagOrganizationStatsDTO{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer func() { _ = tx.Rollback() }()
	err = topicOrganizationCoverageTx(ctx, tx, func(_ string, status string, outcome string) {
		stats.Total++
		switch status {
		case "organized":
			stats.Organized++
			if outcome == "unresolved" {
				stats.Unresolved++
			}
		case "outdated":
			stats.Outdated++
		default:
			stats.Unorganized++
		}
	})
	return stats, err
}

func recordTopicAnalysisTx(ctx context.Context, tx *sql.Tx, jobID string, input TopicMovieInput, outcome string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO movie_topic_analysis(movie_id,job_id,input_fingerprint,outcome,analyzed_at) VALUES(?,?,?,?,?)
	 ON CONFLICT(movie_id) DO UPDATE SET job_id=excluded.job_id,input_fingerprint=excluded.input_fingerprint,outcome=excluded.outcome,analyzed_at=excluded.analyzed_at`, input.MovieID, jobID, input.Fingerprint, outcome, nowUTC())
	return err
}

// CompleteUnresolvedTopic validates the source again and commits no-match
// completion and its checkpoint atomically, just like successful tag application.
func (s *SQLiteStore) CompleteUnresolvedTopic(ctx context.Context, jobID string, input TopicMovieInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := topicMovieInputTx(ctx, tx, input.MovieID)
	if err != nil {
		return err
	}
	if current.Fingerprint != input.Fingerprint || current.Revision != input.Revision {
		return ErrAIWriteConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE ai_tag_organization_items SET status='unresolved',reason='INSUFFICIENT_EVIDENCE',input_fingerprint=? WHERE job_id=? AND movie_id=? AND status='pending' AND EXISTS(SELECT 1 FROM ai_tag_organization_jobs j WHERE j.id=job_id AND j.status='running')`, input.Fingerprint, jobID, input.MovieID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrAIWriteConflict
	}
	if err = recordTopicAnalysisTx(ctx, tx, jobID, input, "unresolved"); err != nil {
		return err
	}
	return tx.Commit()
}
