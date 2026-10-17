package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"encoding/json"
)

// SubtitleRecord persists metadata only. Body checkpoints are separate files.
type SubtitleRecord struct {
	Job           contracts.SubtitleJob `json:"job"`
	AttemptID     string                `json:"attemptId,omitempty"`
	RequestID     string                `json:"requestId"`
	SourcePath    string                `json:"sourcePath"`
	Revision      string                `json:"revision"`
	AudioTrack    int                   `json:"audioTrack"`
	TextBaseURL   string                `json:"textBaseUrl"`
	TextModel     string                `json:"textModel"`
	PromptVersion string                `json:"promptVersion"`
}

func (s *SQLiteStore) ListSubtitleRecords(ctx context.Context, movieID string) ([]SubtitleRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM subtitle_jobs WHERE (?='' OR movie_id=?) ORDER BY updated_at DESC`, movieID, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SubtitleRecord{}
	for rows.Next() {
		var raw string
		var r SubtitleRecord
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
func (s *SQLiteStore) SubtitleRecord(ctx context.Context, id string) (SubtitleRecord, error) {
	var raw string
	var r SubtitleRecord
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM subtitle_jobs WHERE id=?`, id).Scan(&raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &r)
	}
	return r, err
}
func (s *SQLiteStore) SaveSubtitleRecord(ctx context.Context, r SubtitleRecord) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO subtitle_jobs(id,request_id,movie_id,file_id,state,payload,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,payload=excluded.payload,updated_at=excluded.updated_at`, r.Job.ID, r.RequestID, r.Job.MovieID, r.Job.FileID, r.Job.State, string(raw), r.Job.UpdatedAt)
	return err
}
