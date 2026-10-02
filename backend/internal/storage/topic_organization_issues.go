package storage

import (
	"context"
	"curated-backend/internal/contracts"
)

// TagOrganizationIssues lists active movies without retrying any work.
func (s *SQLiteStore) TagOrganizationIssues(ctx context.Context, limit, offset int) ([]contracts.TagOrganizationItemDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.title,q.reason FROM movie_topic_issues q JOIN movies m ON m.id=q.movie_id WHERE `+sqlMovieActiveClause+` ORDER BY m.id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []contracts.TagOrganizationItemDTO{}
	for rows.Next() {
		v := contracts.TagOrganizationItemDTO{Status: "failed", Evidence: []contracts.TopicEvidenceDTO{}}
		if err = rows.Scan(&v.MovieID, &v.Title, &v.Reason); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
