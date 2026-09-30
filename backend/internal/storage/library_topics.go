package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"database/sql"
)

// ListLibraryTopics 在活动库范围内聚合用户题材，计数与影片列表共用可见性规则。
func (s *SQLiteStore) ListLibraryTopics(ctx context.Context, limit, offset int) ([]contracts.LibraryTopicDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,t.name,p.description,p.hidden,
	(SELECT COUNT(*) FROM movie_tags mt JOIN movies m ON m.id=mt.movie_id WHERE mt.tag_id=t.id AND `+sqlMovieActiveClause+`)
	FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' ORDER BY p.id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []contracts.LibraryTopicDTO{}
	for rows.Next() {
		var v contracts.LibraryTopicDTO
		if err = rows.Scan(&v.ID, &v.Name, &v.Description, &v.Hidden, &v.MovieCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetLibraryTopic 返回单个题材，名称变化不依赖 URL 参数。
func (s *SQLiteStore) GetLibraryTopic(ctx context.Context, id string) (contracts.LibraryTopicDTO, error) {
	v := contracts.LibraryTopicDTO{}
	err := s.db.QueryRowContext(ctx, `SELECT p.id,t.name,p.description,p.hidden,
	(SELECT COUNT(*) FROM movie_tags mt JOIN movies m ON m.id=mt.movie_id WHERE mt.tag_id=t.id AND `+sqlMovieActiveClause+`)
	FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' WHERE p.id=?`, id).Scan(&v.ID, &v.Name, &v.Description, &v.Hidden, &v.MovieCount)
	return v, err
}

// SetLibraryTopicHidden 仅更新呈现偏好，不改变任何影片标签。
func (s *SQLiteStore) SetLibraryTopicHidden(ctx context.Context, id string, hidden bool) error {
	r, err := s.db.ExecContext(ctx, `UPDATE library_topics SET hidden=? WHERE id=?`, hidden, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
