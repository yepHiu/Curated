package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"jav-shadcn/backend/internal/contracts"
)

type movieRow struct {
	ID              string
	Title           string
	Code            string
	Studio          string
	Summary         string
	RuntimeMinutes  int
	MetadataRating  float64
	UserRating      sql.NullFloat64
	IsFavorite      bool
	AddedAt         string
	Location        string
	Resolution      string
	Year            int
	CoverURL        string
	ThumbURL        string
	PreviewVideoURL string
}

func (s *SQLiteStore) ListMovies(ctx context.Context, request contracts.ListMoviesRequest) (contracts.MoviesPageDTO, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = 24
	}

	offset := request.Offset
	if offset < 0 {
		offset = 0
	}

	whereClause, args := buildMovieFilters(request)

	var total int
	countQuery := "SELECT COUNT(*) FROM movies " + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return contracts.MoviesPageDTO{}, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, title, code, studio, summary, runtime_minutes, rating, user_rating, is_favorite, added_at, location, resolution, year,
			cover_url, thumb_url, preview_video_url
		FROM movies `+whereClause+`
		ORDER BY added_at DESC, id ASC
		LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return contracts.MoviesPageDTO{}, err
	}
	defer rows.Close()

	records := make([]movieRow, 0, limit)
	ids := make([]string, 0, limit)

	for rows.Next() {
		var row movieRow
		if err := rows.Scan(
			&row.ID,
			&row.Title,
			&row.Code,
			&row.Studio,
			&row.Summary,
			&row.RuntimeMinutes,
			&row.MetadataRating,
			&row.UserRating,
			&row.IsFavorite,
			&row.AddedAt,
			&row.Location,
			&row.Resolution,
			&row.Year,
			&row.CoverURL,
			&row.ThumbURL,
			&row.PreviewVideoURL,
		); err != nil {
			return contracts.MoviesPageDTO{}, err
		}
		records = append(records, row)
		ids = append(ids, row.ID)
	}

	if err := rows.Err(); err != nil {
		return contracts.MoviesPageDTO{}, err
	}

	actorsByMovie, err := s.lookupActors(ctx, ids)
	if err != nil {
		return contracts.MoviesPageDTO{}, err
	}
	tagsByMovie, err := s.lookupTags(ctx, ids)
	if err != nil {
		return contracts.MoviesPageDTO{}, err
	}

	items := make([]contracts.MovieListItemDTO, 0, len(records))
	for _, row := range records {
		items = append(items, contracts.MovieListItemDTO{
			ID:             row.ID,
			Title:          row.Title,
			Code:           row.Code,
			Studio:         row.Studio,
			Actors:         actorsByMovie[row.ID],
			Tags:           tagsByMovie[row.ID],
			RuntimeMinutes: row.RuntimeMinutes,
			Rating:         effectiveRating(row.MetadataRating, row.UserRating),
			IsFavorite:     row.IsFavorite,
			AddedAt:        row.AddedAt,
			Location:       row.Location,
			Resolution:     row.Resolution,
			Year:           row.Year,
			CoverURL:       row.CoverURL,
			ThumbURL:       row.ThumbURL,
		})
	}

	return contracts.MoviesPageDTO{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *SQLiteStore) GetMovieDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error) {
	var row movieRow
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, title, code, studio, summary, runtime_minutes, rating, user_rating, is_favorite, added_at, location, resolution, year,
			cover_url, thumb_url, preview_video_url
		FROM movies WHERE id = ?`,
		movieID,
	).Scan(
		&row.ID,
		&row.Title,
		&row.Code,
		&row.Studio,
		&row.Summary,
		&row.RuntimeMinutes,
		&row.MetadataRating,
		&row.UserRating,
		&row.IsFavorite,
		&row.AddedAt,
		&row.Location,
		&row.Resolution,
		&row.Year,
		&row.CoverURL,
		&row.ThumbURL,
		&row.PreviewVideoURL,
	)
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	actorsByMovie, err := s.lookupActors(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}
	tagsByMovie, err := s.lookupTags(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	previewsByMovie, err := s.lookupPreviewImageURLs(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	return contracts.MovieDetailDTO{
		MovieListItemDTO: contracts.MovieListItemDTO{
			ID:             row.ID,
			Title:          row.Title,
			Code:           row.Code,
			Studio:         row.Studio,
			Actors:         actorsByMovie[row.ID],
			Tags:           tagsByMovie[row.ID],
			RuntimeMinutes: row.RuntimeMinutes,
			Rating:         effectiveRating(row.MetadataRating, row.UserRating),
			IsFavorite:     row.IsFavorite,
			AddedAt:        row.AddedAt,
			Location:       row.Location,
			Resolution:     row.Resolution,
			Year:           row.Year,
			CoverURL:       row.CoverURL,
			ThumbURL:       row.ThumbURL,
		},
		Summary:          row.Summary,
		PreviewImages:    previewsByMovie[movieID],
		PreviewVideoURL:  row.PreviewVideoURL,
		MetadataRating:   row.MetadataRating,
		UserRating:       userRatingPtr(row.UserRating),
	}, nil
}

func buildMovieFilters(request contracts.ListMoviesRequest) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)

	if request.Mode == "favorites" {
		clauses = append(clauses, "is_favorite = 1")
	}

	query := strings.TrimSpace(strings.ToLower(request.Query))
	if query != "" {
		like := "%" + query + "%"
		clauses = append(clauses, `(LOWER(title) LIKE ? OR LOWER(code) LIKE ? OR LOWER(studio) LIKE ? OR LOWER(summary) LIKE ?)`)
		args = append(args, like, like, like, like)
	}

	if len(clauses) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (s *SQLiteStore) lookupActors(ctx context.Context, movieIDs []string) (map[string][]string, error) {
	return s.lookupStringRelations(
		ctx,
		movieIDs,
		`SELECT ma.movie_id, a.name
		FROM movie_actors ma
		INNER JOIN actors a ON a.id = ma.actor_id
		WHERE ma.movie_id IN (%s)
		ORDER BY a.name ASC`,
	)
}

func (s *SQLiteStore) lookupTags(ctx context.Context, movieIDs []string) (map[string][]string, error) {
	return s.lookupStringRelations(
		ctx,
		movieIDs,
		`SELECT mt.movie_id, t.name
		FROM movie_tags mt
		INNER JOIN tags t ON t.id = mt.tag_id
		WHERE mt.movie_id IN (%s)
		ORDER BY t.name ASC`,
	)
}

// lookupPreviewImageURLs returns ordered sample/preview image source URLs per movie (from media_assets).
func (s *SQLiteStore) lookupPreviewImageURLs(ctx context.Context, movieIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(movieIDs))
	if len(movieIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, 0, len(movieIDs))
	args := make([]any, 0, len(movieIDs))
	for _, movieID := range movieIDs {
		placeholders = append(placeholders, "?")
		args = append(args, movieID)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT movie_id, source_url FROM media_assets
		WHERE movie_id IN (%s) AND type = 'preview_image' AND source_url != ''
		ORDER BY movie_id ASC, id ASC`,
		strings.Join(placeholders, ", "),
	), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var movieID, url string
		if err := rows.Scan(&movieID, &url); err != nil {
			return nil, err
		}
		result[movieID] = append(result[movieID], url)
	}

	return result, rows.Err()
}

func (s *SQLiteStore) lookupStringRelations(ctx context.Context, movieIDs []string, queryTemplate string) (map[string][]string, error) {
	result := make(map[string][]string, len(movieIDs))
	if len(movieIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, 0, len(movieIDs))
	args := make([]any, 0, len(movieIDs))
	for _, movieID := range movieIDs {
		placeholders = append(placeholders, "?")
		args = append(args, movieID)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(queryTemplate, strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var movieID, value string
		if err := rows.Scan(&movieID, &value); err != nil {
			return nil, err
		}
		result[movieID] = append(result[movieID], value)
	}

	return result, rows.Err()
}
