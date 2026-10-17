package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"slices"
)

// Translation writes and their checkpoints commit together. Source metadata is read-only.
type OrganizationTranslationInput struct {
	MovieID, Kind, Source, Override string
	Available                       bool
}

func (s *SQLiteStore) NextOrganizationTranslation(ctx context.Context, id string) (OrganizationTranslationInput, error) {
	var v OrganizationTranslationInput
	err := s.db.QueryRowContext(ctx, `SELECT t.movie_id,t.kind,COALESCE(CASE t.kind WHEN 'title' THEN m.title ELSE m.summary END,''),COALESCE(CASE t.kind WHEN 'title' THEN m.user_title ELSE m.user_summary END,''),m.id IS NOT NULL AND (m.trashed_at IS NULL OR TRIM(m.trashed_at)='') FROM ai_organization_translations t LEFT JOIN movies m ON m.id=t.movie_id WHERE t.job_id=? AND t.status='pending' ORDER BY t.movie_id,t.kind LIMIT 1`, id).Scan(&v.MovieID, &v.Kind, &v.Source, &v.Override, &v.Available)
	return v, err
}

func (s *SQLiteStore) CompleteOrganizationTranslation(ctx context.Context, id string, v OrganizationTranslationInput, proposed, status, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var active int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_organization_translations t JOIN ai_tag_organization_jobs j ON j.id=t.job_id WHERE j.id=? AND j.status='running' AND j.deleted_at='' AND t.movie_id=? AND t.kind=? AND t.status='pending'`, id, v.MovieID, v.Kind).Scan(&active)
	if err != nil {
		return err
	}
	if active != 1 {
		return ErrAIWriteConflict
	}
	if status == "succeeded" {
		column := "title"
		if v.Kind == "summary" {
			column = "summary"
		} else if v.Kind != "title" {
			return ErrAIWriteConflict
		}
		result, e := tx.ExecContext(ctx, `UPDATE movies AS m SET user_`+column+`=?,updated_at=? WHERE id=? AND `+column+`=? AND COALESCE(user_`+column+`,'')=? AND `+sqlMovieActiveClause, proposed, nowUTC(), v.MovieID, v.Source, v.Override)
		if e != nil {
			return e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			status, reason = "conflict", "TRANSLATION_SOURCE_CHANGED"
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_organization_translations SET status=?,reason=? WHERE job_id=? AND movie_id=? AND kind=?`, status, reason, id, v.MovieID, v.Kind); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET revision=revision+1,updated_at=? WHERE id=?`, nowUTC(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) organizationCounts(ctx context.Context, j *contracts.TagOrganizationJobDTO) error {
	// Aggregate by movie, so multiple selected actions never inflate movie counts.
	tags := slices.Contains(j.Tasks, "tags")
	return s.db.QueryRowContext(ctx, `WITH results AS (
 SELECT i.movie_id,
 (? AND i.status='pending') OR EXISTS(SELECT 1 FROM ai_organization_translations t WHERE t.job_id=i.job_id AND t.movie_id=i.movie_id AND t.status='pending') pending,
 (? AND i.status IN ('failed','conflict')) OR EXISTS(SELECT 1 FROM ai_organization_translations t WHERE t.job_id=i.job_id AND t.movie_id=i.movie_id AND t.status IN ('failed','conflict')) failed,
 (? AND i.status='unresolved') unresolved,
 NOT EXISTS(SELECT 1 FROM ai_organization_translations t WHERE t.job_id=i.job_id AND t.movie_id=i.movie_id AND t.status='pending') translated
 FROM ai_tag_organization_items i WHERE i.job_id=?)
 SELECT COUNT(*),COALESCE(SUM(NOT pending),0),COALESCE(SUM(NOT pending AND NOT failed AND NOT unresolved),0),COALESCE(SUM(NOT pending AND NOT failed AND unresolved),0),COALESCE(SUM(NOT pending AND failed),0),COALESCE(SUM(translated),0) FROM results`, tags, tags, tags, j.ID).Scan(&j.Total, &j.Processed, &j.Succeeded, &j.Unresolved, &j.Failed, &j.TranslationProcessed)
}

func (s *SQLiteStore) mergeOrganizationTranslations(ctx context.Context, id string, tasks []string, v *contracts.TagOrganizationItemDTO) error {
	rows, err := s.db.QueryContext(ctx, `SELECT kind,status,reason FROM ai_organization_translations WHERE job_id=? AND movie_id=? ORDER BY kind`, id, v.MovieID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	pending, failed := false, false
	if !slices.Contains(tasks, "tags") {
		v.Status = "succeeded"
		v.Reason = ""
	}
	for rows.Next() {
		var r contracts.OrganizationTranslationDTO
		if err = rows.Scan(&r.Kind, &r.Status, &r.Reason); err != nil {
			return err
		}
		v.Translations = append(v.Translations, r)
		pending = pending || r.Status == "pending"
		if r.Status == "failed" || r.Status == "conflict" {
			failed = true
			v.Reason = r.Reason
		}
	}
	if pending || v.Status == "pending" {
		v.Status = "pending"
	} else if failed {
		v.Status = "failed"
	}
	return rows.Err()
}
