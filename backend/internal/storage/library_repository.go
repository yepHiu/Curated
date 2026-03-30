package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"curated-backend/internal/contracts"
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
	ReleaseDate     string
	CoverURL        string
	ThumbURL        string
	PreviewVideoURL string
	TrashedAt       string
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
	countQuery := "SELECT COUNT(*) FROM movies m " + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return contracts.MoviesPageDTO{}, err
	}

	args = append(args, limit, offset)
	orderBy := "ORDER BY m.added_at DESC, m.id ASC"
	if strings.EqualFold(strings.TrimSpace(request.Mode), "trash") {
		orderBy = "ORDER BY m.trashed_at DESC, m.id ASC"
	}
	rows, err := s.db.QueryContext(
		ctx,
		movieSelectEffectiveColumns+`
		FROM movies m `+whereClause+`
		`+orderBy+`
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
			&row.ReleaseDate,
			&row.CoverURL,
			&row.ThumbURL,
			&row.PreviewVideoURL,
			&row.TrashedAt,
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
	metaTags, userTagsByMovie, err := s.lookupTagsGrouped(ctx, ids)
	if err != nil {
		return contracts.MoviesPageDTO{}, err
	}

	items := make([]contracts.MovieListItemDTO, 0, len(records))
	for _, row := range records {
		item := contracts.MovieListItemDTO{
			ID:             row.ID,
			Title:          row.Title,
			Code:           row.Code,
			Studio:         row.Studio,
			Actors:         actorsByMovie[row.ID],
			Tags:           metaTags[row.ID],
			UserTags:       userTagsByMovie[row.ID],
			RuntimeMinutes: row.RuntimeMinutes,
			Rating:         effectiveRating(row.MetadataRating, row.UserRating),
			IsFavorite:     row.IsFavorite,
			AddedAt:        row.AddedAt,
			Location:       row.Location,
			Resolution:     row.Resolution,
			Year:           row.Year,
			ReleaseDate:    row.ReleaseDate,
			CoverURL:       row.CoverURL,
			ThumbURL:       row.ThumbURL,
		}
		if strings.TrimSpace(row.TrashedAt) != "" {
			item.TrashedAt = strings.TrimSpace(row.TrashedAt)
		}
		items = append(items, item)
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
		movieSelectEffectiveColumns+`
		FROM movies m WHERE m.id = ?`,
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
		&row.ReleaseDate,
		&row.CoverURL,
		&row.ThumbURL,
		&row.PreviewVideoURL,
		&row.TrashedAt,
	)
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	actorsByMovie, err := s.lookupActors(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}
	metaTags, userTagsByMovie, err := s.lookupTagsGrouped(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	previewsByMovie, err := s.lookupPreviewImageURLs(ctx, []string{movieID})
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	actorAvatars, err := s.lookupActorAvatarURLsByMovieID(ctx, movieID)
	if err != nil {
		return contracts.MovieDetailDTO{}, err
	}

	listDTO := contracts.MovieListItemDTO{
		ID:             row.ID,
		Title:          row.Title,
		Code:           row.Code,
		Studio:         row.Studio,
		Actors:         actorsByMovie[row.ID],
		Tags:           metaTags[row.ID],
		UserTags:       userTagsByMovie[row.ID],
		RuntimeMinutes: row.RuntimeMinutes,
		Rating:         effectiveRating(row.MetadataRating, row.UserRating),
		IsFavorite:     row.IsFavorite,
		AddedAt:        row.AddedAt,
		Location:       row.Location,
		Resolution:     row.Resolution,
		Year:           row.Year,
		ReleaseDate:    row.ReleaseDate,
		CoverURL:       row.CoverURL,
		ThumbURL:       row.ThumbURL,
	}
	if strings.TrimSpace(row.TrashedAt) != "" {
		listDTO.TrashedAt = strings.TrimSpace(row.TrashedAt)
	}
	return contracts.MovieDetailDTO{
		MovieListItemDTO: listDTO,
		Summary:          row.Summary,
		PreviewImages:    previewsByMovie[movieID],
		PreviewVideoURL:  row.PreviewVideoURL,
		MetadataRating:   row.MetadataRating,
		UserRating:       userRatingPtr(row.UserRating),
		ActorAvatarURLs:  actorAvatars,
	}, nil
}

func (s *SQLiteStore) lookupActorAvatarURLsByMovieID(ctx context.Context, movieID string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.name, a.avatar FROM movie_actors ma
		INNER JOIN actors a ON a.id = ma.actor_id
		WHERE ma.movie_id = ?
		ORDER BY a.name ASC`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var name, avatar string
		if err := rows.Scan(&name, &avatar); err != nil {
			return nil, err
		}
		if strings.TrimSpace(avatar) != "" {
			out[name] = strings.TrimSpace(avatar)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// movieSelectEffectiveColumns selects list/detail fields with user_* display overrides applied (alias m).
const movieSelectEffectiveColumns = `
SELECT m.id,
	COALESCE(NULLIF(TRIM(m.user_title), ''), m.title) AS title,
	m.code,
	COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio) AS studio,
	COALESCE(NULLIF(TRIM(m.user_summary), ''), m.summary) AS summary,
	COALESCE(m.user_runtime_minutes, m.runtime_minutes) AS runtime_minutes,
	m.rating, m.user_rating, m.is_favorite, m.added_at, m.location, m.resolution,
	CASE
		WHEN NULLIF(TRIM(m.user_release_date), '') IS NOT NULL
			AND LENGTH(TRIM(m.user_release_date)) >= 4
			AND CAST(SUBSTR(TRIM(m.user_release_date), 1, 4) AS INTEGER) BETWEEN 1800 AND 3000
		THEN CAST(SUBSTR(TRIM(m.user_release_date), 1, 4) AS INTEGER)
		ELSE m.year
	END AS year,
	COALESCE(NULLIF(TRIM(m.user_release_date), ''), m.release_date) AS release_date,
	m.cover_url, m.thumb_url, m.preview_video_url,
	IFNULL(m.trashed_at, '') AS trashed_at`

func buildMovieFilters(request contracts.ListMoviesRequest) (string, []any) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 8)

	mode := strings.TrimSpace(strings.ToLower(request.Mode))
	if mode == "trash" {
		clauses = append(clauses, sqlMovieTrashedClause)
	} else {
		clauses = append(clauses, sqlMovieActiveClause)
	}

	if request.Mode == "favorites" {
		clauses = append(clauses, "m.is_favorite = 1")
	}

	query := strings.TrimSpace(strings.ToLower(request.Query))
	if query != "" {
		like := "%" + query + "%"
		clauses = append(clauses, `(LOWER(COALESCE(NULLIF(TRIM(m.user_title), ''), m.title)) LIKE ? OR LOWER(m.code) LIKE ? OR LOWER(COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio)) LIKE ? OR LOWER(COALESCE(NULLIF(TRIM(m.user_summary), ''), m.summary)) LIKE ?)`)
		args = append(args, like, like, like, like)
	}

	if actor := strings.TrimSpace(request.Actor); actor != "" {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM movie_actors ma
			INNER JOIN actors act ON act.id = ma.actor_id
			WHERE ma.movie_id = m.id AND act.name = ?
		)`)
		args = append(args, actor)
	}

	if studio := strings.TrimSpace(request.Studio); studio != "" {
		clauses = append(clauses, `TRIM(COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio)) = ?`)
		args = append(args, studio)
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

// lookupTagsGrouped returns scraper/metadata tags (type nfo) and user tags (type user) per movie.
func (s *SQLiteStore) lookupTagsGrouped(ctx context.Context, movieIDs []string) (metadata map[string][]string, user map[string][]string, err error) {
	metadata = make(map[string][]string, len(movieIDs))
	user = make(map[string][]string, len(movieIDs))
	if len(movieIDs) == 0 {
		return metadata, user, nil
	}

	placeholders := make([]string, 0, len(movieIDs))
	args := make([]any, 0, len(movieIDs))
	for _, movieID := range movieIDs {
		placeholders = append(placeholders, "?")
		args = append(args, movieID)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT mt.movie_id, t.name, t.type
		FROM movie_tags mt
		INNER JOIN tags t ON t.id = mt.tag_id
		WHERE mt.movie_id IN (%s)
		ORDER BY t.type ASC, t.name ASC`,
		strings.Join(placeholders, ", "),
	), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var movieID, name, tagType string
		if err := rows.Scan(&movieID, &name, &tagType); err != nil {
			return nil, nil, err
		}
		switch tagType {
		case tagTypeUser:
			user[movieID] = append(user[movieID], name)
		case "nfo":
			metadata[movieID] = append(metadata[movieID], name)
		default:
			metadata[movieID] = append(metadata[movieID], name)
		}
	}

	return metadata, user, rows.Err()
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

// pathHasLibraryRoot reports whether loc (media file path) lies under root (library directory).
func pathHasLibraryRoot(loc, root string) bool {
	if loc == "" || root == "" {
		return false
	}
	loc = filepath.Clean(loc)
	root = filepath.Clean(root)
	if strings.EqualFold(loc, root) {
		return true
	}
	sep := string(filepath.Separator)
	prefix := root
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	if len(loc) < len(prefix) {
		return false
	}
	return strings.EqualFold(loc[:len(prefix)], prefix)
}

// ListMovieIDsUnderLibraryRoots returns distinct movie ids whose location is under any of the given roots.
// Only rows with non-empty code and location are considered. Full table scan; typical libraries are small enough.
func (s *SQLiteStore) ListMovieIDsUnderLibraryRoots(ctx context.Context, roots []string) ([]string, error) {
	if len(roots) == 0 {
		return nil, nil
	}
	cleanRoots := make([]string, 0, len(roots))
	for _, r := range roots {
		c := filepath.Clean(strings.TrimSpace(r))
		if c != "." && c != "" {
			cleanRoots = append(cleanRoots, c)
		}
	}
	if len(cleanRoots) == 0 {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, location, code FROM movies
		 WHERE TRIM(COALESCE(location, '')) != '' AND TRIM(COALESCE(code, '')) != ''
		   AND (trashed_at IS NULL OR TRIM(trashed_at) = '')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	var ids []string
	for rows.Next() {
		var id, location, code string
		if err := rows.Scan(&id, &location, &code); err != nil {
			return nil, err
		}
		loc := filepath.Clean(strings.TrimSpace(location))
		if loc == "" || strings.TrimSpace(code) == "" {
			continue
		}
		for _, root := range cleanRoots {
			if pathHasLibraryRoot(loc, root) {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					ids = append(ids, id)
				}
				break
			}
		}
	}
	return ids, rows.Err()
}
