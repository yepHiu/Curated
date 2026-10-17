package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"curated-backend/internal/contracts"
)

// Read the persistent sources, not the bounded runtime task registry.
const taskCenterSource = `WITH tasks AS (
 SELECT 'subtitle:'||id AS task_key,'subtitle' AS kind,
 CASE state WHEN 'completed' THEN 'succeeded' WHEN 'partial_failed' THEN 'partial' ELSE state END AS state,
 json_extract(payload,'$.job.createdAt') AS created_at,
 COALESCE(json_extract(payload,'$.job.supersededBy'),'') AS superseded_by FROM subtitle_jobs
 UNION ALL
 SELECT 'organize:'||id,'organize',
 CASE WHEN status='running' AND stage='waiting_quota' THEN 'waiting'
 WHEN status='completed' THEN 'succeeded' WHEN status='partial_failed' THEN 'partial' ELSE status END,
 created_at,'' FROM ai_tag_organization_jobs WHERE deleted_at=''
 UNION ALL
 SELECT 'book-organize:'||id,'organize', CASE status WHEN 'completed' THEN 'succeeded' WHEN 'partial_failed' THEN 'partial' ELSE status END,created_at,'' FROM ai_book_organization_jobs
) `

type TaskCenterFilter struct {
	Kind, State, Cursor string
	Limit               int
}
type taskCenterCursor struct {
	Time string `json:"time"`
	Key  string `json:"key"`
}

var ErrTaskCenterFilter = errors.New("invalid task filter or cursor")

func (s *SQLiteStore) TaskCenterSummary(ctx context.Context) (contracts.TaskCenterSummary, error) {
	var out contracts.TaskCenterSummary
	err := s.db.QueryRowContext(ctx, taskCenterSource+`SELECT COUNT(*),
 COALESCE(SUM(state='running'),0),COALESCE(SUM(state='queued'),0),COALESCE(SUM(state='waiting'),0),
 COALESCE(SUM(state IN ('blocked','failed','partial','needs_confirmation') AND superseded_by=''),0),
 COALESCE(SUM(state IN ('succeeded','partial','failed','cancelled') OR superseded_by<>''),0) FROM tasks`).Scan(&out.Total, &out.Running, &out.Queued, &out.Waiting, &out.Attention, &out.History)
	return out, err
}

func (s *SQLiteStore) TaskCenterKeys(ctx context.Context, f TaskCenterFilter) ([]string, string, error) {
	if f.Kind != "" && f.Kind != "all" && f.Kind != "subtitle" && f.Kind != "organize" {
		return nil, "", ErrTaskCenterFilter
	}
	where := " WHERE 1=1"
	args := []any{}
	if f.Kind != "" && f.Kind != "all" {
		where += " AND kind=?"
		args = append(args, f.Kind)
	}
	switch f.State {
	case "", "all":
	case "active":
		where += " AND state IN ('queued','running','waiting')"
	case "attention":
		where += " AND state IN ('blocked','failed','partial','needs_confirmation') AND superseded_by=''"
	case "history":
		where += " AND (state IN ('succeeded','partial','failed','cancelled') OR superseded_by<>'')"
	default:
		return nil, "", ErrTaskCenterFilter
	}
	if f.Cursor != "" {
		var c taskCenterCursor
		raw, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		if err != nil || len(raw) > 1024 || json.Unmarshal(raw, &c) != nil || c.Time == "" || c.Key == "" {
			return nil, "", ErrTaskCenterFilter
		}
		where += " AND (created_at<? OR (created_at=? AND task_key<?))"
		args = append(args, c.Time, c.Time, c.Key)
	}
	if f.Limit <= 0 {
		f.Limit = 30
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	args = append(args, f.Limit+1)
	rows, err := s.db.QueryContext(ctx, taskCenterSource+`SELECT task_key,created_at FROM tasks`+where+` ORDER BY created_at DESC,task_key DESC LIMIT ?`, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	keys := []string{}
	var last taskCenterCursor
	next := ""
	for rows.Next() {
		var c taskCenterCursor
		if err = rows.Scan(&c.Key, &c.Time); err != nil {
			return nil, "", err
		}
		if len(keys) == f.Limit {
			b, _ := json.Marshal(last)
			next = base64.RawURLEncoding.EncodeToString(b)
			break
		}
		keys = append(keys, c.Key)
		last = c
	}
	return keys, next, rows.Err()
}

func (s *SQLiteStore) TaskCenterTask(ctx context.Context, key string) (contracts.TaskCenterTask, error) {
	kind, id, ok := strings.Cut(key, ":")
	if !ok || id == "" {
		return contracts.TaskCenterTask{}, ErrTaskCenterFilter
	}
	out := contracts.TaskCenterTask{Key: key, Kind: kind, ID: id}
	switch kind {
	case "subtitle":
		r, err := s.SubtitleRecord(ctx, id)
		if err != nil {
			return out, err
		}
		j := r.Job
		// Provider errors can contain paths. Details expose a safe business category.
		if j.Error != "" {
			j.Error = "SUBTITLE_TASK_FAILED"
		}
		out.State = contracts.TaskCenterState(j.State, j.Stage)
		out.Stage = j.Stage
		out.Completed = j.Completed
		out.Total = j.Total
		out.Unit = "segments"
		if j.Stage == "translating" {
			out.Unit = "cues"
		}
		out.CreatedAt = j.CreatedAt
		out.UpdatedAt = j.UpdatedAt
		out.Subtitle = &j
		out.AudioTrack = &r.AudioTrack
		out.TextModel = r.TextModel
	case "book-organize":
		j, err := s.GetBookOrganization(ctx, id)
		if err != nil {
			return out, err
		}
		out.Kind, out.State, out.Stage = "organize", contracts.TaskCenterState(j.Status, j.Stage), j.Stage
		out.Completed, out.Total, out.Unit = j.Processed, j.Total, "books"
		out.CreatedAt, out.UpdatedAt, out.BookOrganization = j.CreatedAt, j.UpdatedAt, &j
	case "organize":
		j, err := s.GetTagOrganization(ctx, id)
		if err != nil {
			return out, err
		}
		out.State = contracts.TaskCenterState(j.Status, j.Stage)
		out.Stage = j.Stage
		out.Completed = j.Processed
		out.Total = j.Total
		out.Unit = "movies"
		if j.Stage == "vocabulary" {
			out.Completed = j.VocabularyProcessed
		}
		if j.Stage == "translating_metadata" {
			out.Completed = j.TranslationProcessed
		}
		out.CreatedAt = j.CreatedAt
		out.UpdatedAt = j.UpdatedAt
		out.Organization = &j
	default:
		return out, ErrTaskCenterFilter
	}
	return out, nil
}
