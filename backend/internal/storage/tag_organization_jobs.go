package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrNoOrganizationMovies = errors.New("no eligible movies")

// CreateTagOrganization 固定任务范围并通过唯一约束限制并发；相同请求返回原任务。
func (s *SQLiteStore) CreateTagOrganization(ctx context.Context, id, requestID, reason, locale string, movieIDs []string) (string, error) {
	return s.createTagOrganization(ctx, id, requestID, reason, locale, movieIDs, false)
}

// CreateAllTagOrganization snapshots the active library without loading full movie DTOs.
func (s *SQLiteStore) CreateAllTagOrganization(ctx context.Context, id, requestID, reason, locale string) (string, error) {
	return s.createTagOrganization(ctx, id, requestID, reason, locale, nil, true)
}

func (s *SQLiteStore) createTagOrganization(ctx context.Context, id, requestID, reason, locale string, movieIDs []string, all bool) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	var previous string
	err = tx.QueryRowContext(ctx, `SELECT id FROM ai_tag_organization_jobs WHERE request_id=?`, requestID).Scan(&previous)
	if err == nil {
		return previous, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO ai_tag_organization_jobs(id,request_id,status,trigger_reason,locale,created_at,updated_at) VALUES(?,?,'queued',?,?,?,?)`, id, requestID, reason, locale, nowUTC(), nowUTC()); err != nil {
		return "", err
	}
	if all {
		if _, err = tx.ExecContext(ctx, `INSERT INTO ai_tag_organization_items(job_id,movie_id) SELECT ?,m.id FROM movies m WHERE `+sqlMovieActiveClause, id); err != nil {
			return "", err
		}
	} else {
		for _, mid := range movieIDs {
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO ai_tag_organization_items(job_id,movie_id) SELECT ?,id FROM movies m WHERE id=? AND `+sqlMovieActiveClause, id, mid); err != nil {
				return "", err
			}
		}
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tag_organization_items WHERE job_id=?`, id).Scan(&count); err != nil {
		return "", err
	}
	if count == 0 {
		return "", ErrNoOrganizationMovies
	}
	return id, tx.Commit()
}

// GetTagOrganization 根据逐项检查点计算真实进度，重启不依赖内存计数。
func (s *SQLiteStore) GetTagOrganization(ctx context.Context, id string) (contracts.TagOrganizationJobDTO, error) {
	v := contracts.TagOrganizationJobDTO{}
	err := s.db.QueryRowContext(ctx, `SELECT id,status,stage,trigger_reason,revision,created_at,updated_at,error,vocabulary_processed,vocabulary_ready,locale FROM ai_tag_organization_jobs WHERE id=?`, id).Scan(&v.ID, &v.Status, &v.Stage, &v.TriggerReason, &v.Revision, &v.CreatedAt, &v.UpdatedAt, &v.Error, &v.VocabularyProcessed, &v.VocabularyReady, &v.Locale)
	if err != nil {
		return v, err
	}
	v.TaskID = v.ID
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(status!='pending'),0),COALESCE(SUM(status='succeeded'),0),COALESCE(SUM(status='unresolved'),0),COALESCE(SUM(status IN ('failed','conflict')),0) FROM ai_tag_organization_items WHERE job_id=?`, id).Scan(&v.Total, &v.Processed, &v.Succeeded, &v.Unresolved, &v.Failed)
	return v, err
}

// ListTagOrganizations 返回持久任务列表，用于首次加载、重连与恢复。
func (s *SQLiteStore) ListTagOrganizations(ctx context.Context, active bool) ([]contracts.TagOrganizationJobDTO, error) {
	where := ""
	if active {
		where = " WHERE status IN ('queued','running')"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM ai_tag_organization_jobs`+where+` ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	out := []contracts.TagOrganizationJobDTO{}
	for _, id := range ids {
		v, err := s.GetTagOrganization(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// UpdateTagOrganization 推进状态，取消状态不能被迟到的运行结果复活。
func (s *SQLiteStore) UpdateTagOrganization(ctx context.Context, id, status, stage, message string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET status=?,stage=?,error=?,revision=revision+1,updated_at=? WHERE id=? AND status IN ('queued','running')`, status, stage, message, nowUTC(), id)
	return err
}

// SetTagOrganizationVocabulary 保存本次运行冻结的题材词汇。
func (s *SQLiteStore) SetTagOrganizationVocabulary(ctx context.Context, id string, defs []TopicDefinition) error {
	data, err := json.Marshal(defs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET vocabulary_json=?,vocabulary_ready=1 WHERE id=?`, string(data), id)
	return err
}

// GetTagOrganizationVocabulary 读取已冻结词汇以避免恢复时重复归纳。
func (s *SQLiteStore) GetTagOrganizationVocabulary(ctx context.Context, id string) ([]TopicDefinition, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT vocabulary_json FROM ai_tag_organization_jobs WHERE id=?`, id).Scan(&raw); err != nil {
		return nil, err
	}
	var defs []TopicDefinition
	err := json.Unmarshal([]byte(raw), &defs)
	return defs, err
}

// TagOrganizationItems 分页读取逐片结果或待执行项。
func (s *SQLiteStore) TagOrganizationItems(ctx context.Context, id string, pending bool, limit, offset int) ([]contracts.TagOrganizationItemDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	where := ""
	if pending {
		where = " AND i.status='pending'"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT i.movie_id,i.status,i.reason,COALESCE(m.title,i.movie_id),i.evidence_json FROM ai_tag_organization_items i LEFT JOIN movies m ON m.id=i.movie_id WHERE i.job_id=?`+where+` ORDER BY i.movie_id LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []contracts.TagOrganizationItemDTO{}
	for rows.Next() {
		var v contracts.TagOrganizationItemDTO
		var evidence string
		if err = rows.Scan(&v.MovieID, &v.Status, &v.Reason, &v.Title, &evidence); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(evidence), &v.Evidence); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// SetTagOrganizationItem 仅记录尚未提交项的失败／待归类，不覆盖成功检查点。
func (s *SQLiteStore) SetTagOrganizationItem(ctx context.Context, id, movieID, status, reason string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE ai_tag_organization_items SET status=?,reason=? WHERE job_id=? AND movie_id=? AND status='pending' AND EXISTS(SELECT 1 FROM ai_tag_organization_jobs j WHERE j.id=job_id AND j.status='running')`, status, reason, id, movieID)
	return err
}

// RetryTagOrganization 恢复失败和未完成项，保留成功记录与撤销历史。
func (s *SQLiteStore) RetryTagOrganization(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM ai_tag_organization_jobs WHERE id=?`, id).Scan(&status); err != nil {
		return err
	}
	var undone int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tag_organization_changes WHERE job_id=? AND undone=1`, id).Scan(&undone); err != nil {
		return err
	}
	if undone > 0 || status == "running" || status == "queued" || status == "completed" {
		return fmt.Errorf("job cannot be retried")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_items SET status='pending',reason='' WHERE job_id=? AND status IN ('failed','conflict')`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET status='queued',error='',revision=revision+1,updated_at=? WHERE id=?`, nowUTC(), id); err != nil {
		return err
	}
	return tx.Commit()
}

// SetTopicEvidence checkpoints validated evidence before applying tags, without editing source metadata.
func (s *SQLiteStore) SetTopicEvidence(ctx context.Context, jobID, movieID, fingerprint string, evidence []contracts.TopicEvidenceDTO) error {
	data, err := json.Marshal(evidence)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE ai_tag_organization_items SET evidence_json=?,input_fingerprint=? WHERE job_id=? AND movie_id=? AND status='pending' AND EXISTS(SELECT 1 FROM ai_tag_organization_jobs j WHERE j.id=job_id AND j.status='running')`, string(data), fingerprint, jobID, movieID)
	return err
}

// CheckpointTopicVocabulary atomically persists both definitions and the last analyzed page.
func (s *SQLiteStore) CheckpointTopicVocabulary(ctx context.Context, id string, defs []TopicDefinition, processed int, ready bool) error {
	data, err := json.Marshal(defs)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET vocabulary_json=?,vocabulary_processed=?,vocabulary_ready=?,revision=revision+1,updated_at=? WHERE id=? AND status='running' AND vocabulary_processed<=?`, string(data), processed, ready, nowUTC(), id, processed)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrAIWriteConflict
	}
	return err
}
