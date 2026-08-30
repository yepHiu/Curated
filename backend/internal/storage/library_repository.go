package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
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
	Provider        string
	TrashedAt       string
	Homepage        string
}

// ListMovies returns paginated movie list items with actors, tags, and user preferences applied as display overrides.
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
		if err := scanMovieRow(rows, &row); err != nil {
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
			UserRating:     userRatingPtr(row.UserRating),
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

// GetMovieDetail loads the full movie detail including summary, preview images, and actor avatars.
func (s *SQLiteStore) GetMovieDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error) {
	var row movieRow
	err := scanMovieRow(s.db.QueryRowContext(
		ctx,
		movieSelectEffectiveColumns+`
		FROM movies m WHERE m.id = ?`,
		movieID,
	), &row)
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
		UserRating:     userRatingPtr(row.UserRating),
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
		MetadataProvider: strings.TrimSpace(row.Provider),
		Homepage:         strings.TrimSpace(row.Homepage),
	}, nil
}

func scanMovieRow(scanner interface{ Scan(dest ...any) error }, row *movieRow) error {
	return scanner.Scan(
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
		&row.Provider,
		&row.TrashedAt,
		&row.Homepage,
	)
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

// movieSelectEffectiveColumns 是复用的 SELECT 片段：查询 movies 时表别名必须为 m。
// 列表/详情里对用户可见的文本类字段优先用 user_*（非空且 trim 后非空则覆盖元数据列）；
// year：若 user_release_date 前 4 位可解析为 1800–3000 的年份则用其作为年，否则用 m.year；
// release_date 同样 user 优先；trashed_at 用 IFNULL 归一为 ”，便于 Scan 与下游判断。
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
	m.cover_url, m.thumb_url, m.preview_video_url, m.provider,
	IFNULL(m.trashed_at, '') AS trashed_at,
	IFNULL(m.homepage, '') AS homepage`

const sqlMovieEffectiveYear = `CASE
		WHEN NULLIF(TRIM(m.user_release_date), '') IS NOT NULL
			AND LENGTH(TRIM(m.user_release_date)) >= 4
			AND CAST(SUBSTR(TRIM(m.user_release_date), 1, 4) AS INTEGER) BETWEEN 1800 AND 3000
		THEN CAST(SUBSTR(TRIM(m.user_release_date), 1, 4) AS INTEGER)
		ELSE m.year
	END`

const sqlMovieEffectiveRuntime = `COALESCE(m.user_runtime_minutes, m.runtime_minutes)`

// buildMovieFilters 根据列表请求拼 WHERE 子句与参数，供 ListMovies 等 COUNT/LIMIT 查询复用。
// - mode=trash：仅回收站；否则仅非回收站（见 sqlMovie*Clause）。
// - request.Mode==favorites：额外要求 is_favorite；recent：仅最近 30 天入库。
// - tags 是前端标签聚合布局模式，后端返回与 library 相同的活动影片集合。
// - Query：在「生效」标题、番号、片商、简介上做不区分大小写的子串匹配（LIKE）。
// - Tag / Tags：通过关联表精确匹配（用户标签或 INFO/nfo 标签均可；多个标签为 AND）；Actor：逗号分隔多个演员为 AND（须同时出演，含 alias）；Studio：逗号分隔多个片商为 OR（生效片商 TRIM 后全等）。
// - PlayState / UserRating / Resolution / AddedAfter：Saved Views 所需的稳定筛选语义。
func ParseMovieTagFilters(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}

func buildMovieFilters(request contracts.ListMoviesRequest) (string, []any) {
	clauses := make([]string, 0, 10)
	args := make([]any, 0, 16)

	mode := strings.TrimSpace(strings.ToLower(request.Mode))
	if mode == "trash" {
		clauses = append(clauses, sqlMovieTrashedClause)
	} else {
		clauses = append(clauses, sqlMovieActiveClause)
	}

	if mode == "favorites" {
		clauses = append(clauses, "m.is_favorite = 1")
	}
	if mode == "recent" {
		clauses = append(clauses, "julianday(m.added_at) >= julianday('now', '-30 days')")
	}

	query := strings.TrimSpace(strings.ToLower(request.Query))
	if query != "" {
		like := "%" + query + "%"
		clauses = append(clauses, `(LOWER(COALESCE(NULLIF(TRIM(m.user_title), ''), m.title)) LIKE ? OR LOWER(m.code) LIKE ? OR LOWER(COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio)) LIKE ? OR LOWER(COALESCE(NULLIF(TRIM(m.user_summary), ''), m.summary)) LIKE ?)`)
		args = append(args, like, like, like, like)
	}

	for _, tag := range ParseMovieTagFilters(append([]string{request.Tag}, request.Tags...)...) {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM movie_tags mt
			INNER JOIN tags tag_filter ON tag_filter.id = mt.tag_id
			WHERE mt.movie_id = m.id AND LOWER(tag_filter.name) = LOWER(?)
		)`)
		args = append(args, tag)
	}

	for _, actor := range ParseMovieTagFilters(request.Actor) {
		normalizedActor := NormalizeActorIdentity(actor)
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM movie_actors ma
			INNER JOIN actors act ON act.id = ma.actor_id
			WHERE ma.movie_id = m.id AND (
				act.name = ? OR act.normalized_name = ? OR EXISTS (
					SELECT 1 FROM actor_aliases aa
					WHERE aa.canonical_actor_id = act.id AND aa.normalized_alias = ?
				)
			)
		)`)
		args = append(args, actor, normalizedActor, normalizedActor)
	}

	if studios := ParseMovieTagFilters(request.Studio); len(studios) == 1 {
		clauses = append(clauses, `TRIM(COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio)) = ?`)
		args = append(args, studios[0])
	} else if len(studios) > 1 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(studios)), ",")
		clauses = append(clauses, fmt.Sprintf(
			`TRIM(COALESCE(NULLIF(TRIM(m.user_studio), ''), m.studio)) IN (%s)`,
			placeholders,
		))
		for _, studio := range studios {
			args = append(args, studio)
		}
	}

	switch strings.ToLower(strings.TrimSpace(request.PlayState)) {
	case "unwatched":
		clauses = append(clauses, `NOT EXISTS (
			SELECT 1 FROM library_played_movies played WHERE played.movie_id = m.id
		) AND NOT EXISTS (
			SELECT 1 FROM playback_progress progress
			WHERE progress.movie_id = m.id AND progress.position_sec >= 5
		)`)
	case "in-progress":
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM playback_progress progress
			WHERE progress.movie_id = m.id
			  AND progress.position_sec >= 5
			  AND (progress.duration_sec <= 0 OR progress.position_sec < progress.duration_sec * 0.95)
		)`)
	case "completed":
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM playback_progress progress
			WHERE progress.movie_id = m.id
			  AND progress.duration_sec > 0
			  AND progress.position_sec >= progress.duration_sec * 0.95
		)`)
	}

	if request.Unrated {
		clauses = append(clauses, "m.user_rating IS NULL")
	} else if request.UserRating != nil {
		clauses = append(clauses, "m.user_rating >= ?")
		args = append(args, *request.UserRating)
	}

	if resolution := strings.ToLower(strings.TrimSpace(request.Resolution)); resolution != "" {
		if resolution == "4k" {
			clauses = append(clauses, `LOWER(TRIM(m.resolution)) IN ('4k', '2160p', 'uhd', '3840x2160')`)
		} else {
			clauses = append(clauses, "LOWER(TRIM(m.resolution)) = ?")
			args = append(args, resolution)
		}
	}

	if addedAfter := strings.TrimSpace(request.AddedAfter); addedAfter != "" {
		clauses = append(clauses, "julianday(m.added_at) >= julianday(?)")
		args = append(args, addedAfter)
	}

	switch strings.ToLower(strings.TrimSpace(request.Year)) {
	case "":
	case "unknown":
		clauses = append(clauses, "("+sqlMovieEffectiveYear+") NOT BETWEEN 1800 AND 3000")
	default:
		clauses = append(clauses, "("+sqlMovieEffectiveYear+") = ?")
		args = append(args, strings.TrimSpace(request.Year))
	}

	switch strings.ToLower(strings.TrimSpace(request.Runtime)) {
	case "short":
		clauses = append(clauses, "("+sqlMovieEffectiveRuntime+") > 0 AND ("+sqlMovieEffectiveRuntime+") < 90")
	case "standard":
		clauses = append(clauses, "("+sqlMovieEffectiveRuntime+") >= 90 AND ("+sqlMovieEffectiveRuntime+") <= 150")
	case "long":
		clauses = append(clauses, "("+sqlMovieEffectiveRuntime+") > 150")
	}

	switch strings.ToLower(strings.TrimSpace(request.Catalog)) {
	case "unscraped":
		clauses = append(clauses, `NOT EXISTS (
			SELECT 1 FROM movie_actors ma WHERE ma.movie_id = m.id
		) AND NOT EXISTS (
			SELECT 1 FROM movie_tags mt
			INNER JOIN tags catalog_tag ON catalog_tag.id = mt.tag_id
			WHERE mt.movie_id = m.id AND catalog_tag.type = 'nfo'
		)`)
	case "no-cover":
		clauses = append(clauses, `NULLIF(TRIM(m.cover_url), '') IS NULL AND NULLIF(TRIM(m.thumb_url), '') IS NULL`)
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
	for _, movieID := range movieIDs {
		metadata[movieID] = []string{}
		user[movieID] = []string{}
	}
	if len(movieIDs) == 0 {
		return metadata, user, nil
	}

	if err := forEachInClauseBatch(movieIDs, func(batch []string) error {
		rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
			`SELECT mt.movie_id, t.name, t.type
			FROM movie_tags mt
			INNER JOIN tags t ON t.id = mt.tag_id
			WHERE mt.movie_id IN (%s)
			ORDER BY t.type ASC, t.name ASC`,
			inClausePlaceholders(len(batch)),
		), inClauseArgs(batch)...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var movieID, name, tagType string
			if err := rows.Scan(&movieID, &name, &tagType); err != nil {
				_ = rows.Close()
				return err
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
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	}); err != nil {
		return nil, nil, err
	}

	return metadata, user, nil
}

// lookupPreviewImageURLs returns ordered sample/preview image source URLs per movie (from media_assets).
func (s *SQLiteStore) lookupPreviewImageURLs(ctx context.Context, movieIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(movieIDs))
	if len(movieIDs) == 0 {
		return result, nil
	}

	if err := forEachInClauseBatch(movieIDs, func(batch []string) error {
		rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
			`SELECT movie_id, source_url FROM media_assets
			WHERE movie_id IN (%s) AND type = 'preview_image' AND source_url != ''
			ORDER BY movie_id ASC, id ASC`,
			inClausePlaceholders(len(batch)),
		), inClauseArgs(batch)...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var movieID, url string
			if err := rows.Scan(&movieID, &url); err != nil {
				_ = rows.Close()
				return err
			}
			result[movieID] = append(result[movieID], url)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SQLiteStore) lookupStringRelations(ctx context.Context, movieIDs []string, queryTemplate string) (map[string][]string, error) {
	result := make(map[string][]string, len(movieIDs))
	for _, movieID := range movieIDs {
		result[movieID] = []string{}
	}
	if len(movieIDs) == 0 {
		return result, nil
	}

	if err := forEachInClauseBatch(movieIDs, func(batch []string) error {
		rows, err := s.db.QueryContext(ctx, fmt.Sprintf(queryTemplate, inClausePlaceholders(len(batch))), inClauseArgs(batch)...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var movieID, value string
			if err := rows.Scan(&movieID, &value); err != nil {
				_ = rows.Close()
				return err
			}
			result[movieID] = append(result[movieID], value)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// pathHasLibraryRoot reports whether loc (media file path) lies under root (library directory).
// Uses case-insensitive prefix matching for case-insensitive filesystems (e.g. Windows).
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

// FindActiveMoviesByCodes maps normalized video codes to active library rows.
func (s *SQLiteStore) FindActiveMoviesByCodes(ctx context.Context, codes []string) (map[string]contracts.MovieListItemDTO, error) {
	wanted := map[string]struct{}{}
	var ids []string
	var lowered []string
	for _, raw := range codes {
		normalized := moviecode.NormalizeForStorageID(raw)
		if normalized == "" {
			continue
		}
		if _, ok := wanted[normalized]; ok {
			continue
		}
		wanted[normalized] = struct{}{}
		ids = append(ids, normalized)
		lowered = append(lowered, strings.ToLower(strings.TrimSpace(raw)))
	}
	out := map[string]contracts.MovieListItemDTO{}
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := inClausePlaceholders(len(ids))
	query := `SELECT m.id, m.code, COALESCE(NULLIF(TRIM(m.user_title), ''), m.title) AS title
		FROM movies m
		WHERE IFNULL(m.trashed_at, '') = '' AND (m.id IN (` + placeholders + `) OR lower(m.code) IN (` + placeholders + `))`
	args := make([]any, 0, len(ids)+len(lowered))
	for _, id := range ids {
		args = append(args, id)
	}
	for _, code := range lowered {
		args = append(args, code)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item contracts.MovieListItemDTO
		if err := rows.Scan(&item.ID, &item.Code, &item.Title); err != nil {
			return nil, err
		}
		key := moviecode.NormalizeForStorageID(item.Code)
		if key == "" {
			key = moviecode.NormalizeForStorageID(item.ID)
		}
		if _, ok := wanted[key]; ok {
			out[key] = item
		}
	}
	return out, rows.Err()
}

// MovieCodeIndexItem is a compact active-library catalog row for import code checks.
type MovieCodeIndexItem struct {
	ID    string
	Code  string
	Title string
}

// ListActiveMovieCodeIndex returns id/code/title for every active movie.
func (s *SQLiteStore) ListActiveMovieCodeIndex(ctx context.Context) ([]MovieCodeIndexItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.code, COALESCE(NULLIF(TRIM(m.user_title), ''), m.title) AS title
		FROM movies m
		WHERE IFNULL(m.trashed_at, '') = ''
		ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MovieCodeIndexItem
	for rows.Next() {
		var item MovieCodeIndexItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Title); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if out == nil {
		out = []MovieCodeIndexItem{}
	}
	return out, rows.Err()
}
