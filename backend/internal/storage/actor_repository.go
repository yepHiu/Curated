package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"jav-shadcn/backend/internal/contracts"
	"jav-shadcn/backend/internal/scraper"
)

// GetActorProfile loads one row from actors by exact name (library display name).
func (s *SQLiteStore) GetActorProfile(ctx context.Context, name string) (contracts.ActorProfileDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return contracts.ActorProfileDTO{}, contracts.ErrActorNotFound
	}
	var (
		dto              contracts.ActorProfileDTO
		actorID          int64
		avatar           sql.NullString
		summary          sql.NullString
		homepage         sql.NullString
		provider         sql.NullString
		providerActorID  sql.NullString
		height           int
		birthday         sql.NullString
		profileUpdatedAt sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, avatar, summary, homepage, provider, provider_actor_id, height, birthday, profile_updated_at
		FROM actors WHERE name = ?`, name,
	).Scan(
		&actorID,
		&dto.Name,
		&avatar,
		&summary,
		&homepage,
		&provider,
		&providerActorID,
		&height,
		&birthday,
		&profileUpdatedAt,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ActorProfileDTO{}, contracts.ErrActorNotFound
	case err != nil:
		return contracts.ActorProfileDTO{}, err
	}
	dto.AvatarURL = avatar.String
	dto.Summary = summary.String
	dto.Homepage = homepage.String
	dto.Provider = provider.String
	dto.ProviderActorID = providerActorID.String
	dto.Height = height
	dto.Birthday = birthday.String
	dto.ProfileUpdatedAt = profileUpdatedAt.String
	tagsByID, err := s.loadActorUserTagsForIDs(ctx, []int64{actorID})
	if err != nil {
		return contracts.ActorProfileDTO{}, err
	}
	dto.UserTags = tagsByID[actorID]
	return dto, nil
}

// ActorNameExists reports whether an actors row exists for the exact display name.
func (s *SQLiteStore) ActorNameExists(ctx context.Context, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM actors WHERE name = ? LIMIT 1`, name).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

// UpdateActorProfile persists scraped fields for the library actor row keyed by DisplayName.
func (s *SQLiteStore) UpdateActorProfile(ctx context.Context, p scraper.ActorProfile) error {
	name := strings.TrimSpace(p.DisplayName)
	if name == "" {
		return errors.New("empty actor display name")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE actors SET
			avatar = ?,
			summary = ?,
			homepage = ?,
			provider = ?,
			provider_actor_id = ?,
			height = ?,
			birthday = ?,
			profile_updated_at = ?
		WHERE name = ?`,
		strings.TrimSpace(p.AvatarURL),
		strings.TrimSpace(p.Summary),
		strings.TrimSpace(p.Homepage),
		strings.TrimSpace(p.Provider),
		strings.TrimSpace(p.ProviderActorID),
		p.Height,
		strings.TrimSpace(p.Birthday),
		nowUTC(),
		name,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return contracts.ErrActorNotFound
	}
	return nil
}

// ListActors returns paginated actors that appear in at least one library movie (movie_actors),
// with counts and actor-scoped user tags (not movie tags).
func (s *SQLiteStore) ListActors(ctx context.Context, req contracts.ListActorsRequest) (contracts.ListActorsResponse, error) {
	q := strings.TrimSpace(req.Q)
	tagFilter := strings.TrimSpace(req.ActorTag)
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	sort := strings.TrimSpace(strings.ToLower(req.Sort))
	orderSQL := "a.name COLLATE NOCASE ASC"
	if sort == "moviecount" {
		orderSQL = "movie_count DESC, a.name COLLATE NOCASE ASC"
	}

	whereParts := []string{"1=1"}
	args := make([]any, 0, 6)
	if q != "" {
		// 顶栏搜索：演员名子串 或 演员用户标签（actor_user_tags）子串，均不区分大小写
		whereParts = append(whereParts, `(
			INSTR(LOWER(a.name), LOWER(?)) > 0
			OR EXISTS (
				SELECT 1 FROM actor_user_tags aut
				WHERE aut.actor_id = a.id AND INSTR(LOWER(aut.tag), LOWER(?)) > 0
			)
		)`)
		args = append(args, q, q)
	}
	if tagFilter != "" {
		whereParts = append(whereParts, `EXISTS (SELECT 1 FROM actor_user_tags aut WHERE aut.actor_id = a.id AND aut.tag = ?)`)
		args = append(args, tagFilter)
	}
	whereClause := strings.Join(whereParts, " AND ")

	// 仅列出库内仍有影片的演员：movie_actors 且对应 movies 行存在；参演部数 > 0
	countQuery := fmt.Sprintf(
		`SELECT COUNT(DISTINCT a.id) FROM actors a
		 INNER JOIN movie_actors ma ON ma.actor_id = a.id
		 INNER JOIN movies m ON m.id = ma.movie_id
		 WHERE %s`,
		whereClause,
	)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return contracts.ListActorsResponse{}, err
	}

	listQuery := fmt.Sprintf(`
		SELECT a.id, a.name, a.avatar, COUNT(DISTINCT ma.movie_id) AS movie_count
		FROM actors a
		INNER JOIN movie_actors ma ON ma.actor_id = a.id
		INNER JOIN movies m ON m.id = ma.movie_id
		WHERE %s
		GROUP BY a.id, a.name, a.avatar
		HAVING COUNT(DISTINCT ma.movie_id) > 0
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, orderSQL)

	argsWithPaging := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx, listQuery, argsWithPaging...)
	if err != nil {
		return contracts.ListActorsResponse{}, err
	}
	defer rows.Close()

	type row struct {
		id         int64
		name       string
		avatar     string
		movieCount int
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.avatar, &r.movieCount); err != nil {
			return contracts.ListActorsResponse{}, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return contracts.ListActorsResponse{}, err
	}

	if len(list) == 0 {
		return contracts.ListActorsResponse{Total: total, Actors: []contracts.ActorListItemDTO{}}, nil
	}

	ids := make([]int64, len(list))
	for i, r := range list {
		ids[i] = r.id
	}
	tagsByActor, err := s.loadActorUserTagsForIDs(ctx, ids)
	if err != nil {
		return contracts.ListActorsResponse{}, err
	}

	out := make([]contracts.ActorListItemDTO, 0, len(list))
	for _, r := range list {
		if r.movieCount <= 0 {
			continue
		}
		out = append(out, contracts.ActorListItemDTO{
			Name:       r.name,
			AvatarURL:  strings.TrimSpace(r.avatar),
			MovieCount: r.movieCount,
			UserTags:   tagsByActor[r.id],
		})
	}
	return contracts.ListActorsResponse{Total: total, Actors: out}, nil
}

func (s *SQLiteStore) loadActorUserTagsForIDs(ctx context.Context, ids []int64) (map[int64][]string, error) {
	out := make(map[int64][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = strings.TrimSuffix(placeholders, ",")
	query := fmt.Sprintf(
		`SELECT actor_id, tag FROM actor_user_tags WHERE actor_id IN (%s) ORDER BY actor_id, tag`,
		placeholders,
	)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var actorID int64
		var tag string
		if err := rows.Scan(&actorID, &tag); err != nil {
			return nil, err
		}
		out[actorID] = append(out[actorID], tag)
	}
	return out, rows.Err()
}

// ReplaceActorUserTagsByName replaces all actor_user_tags for actors.name (exact).
func (s *SQLiteStore) ReplaceActorUserTagsByName(ctx context.Context, name string, rawTags []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return contracts.ErrActorNotFound
	}
	tags, err := NormalizeUserTagsForPatch(rawTags)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var actorID int64
	switch err := tx.QueryRowContext(ctx, `SELECT id FROM actors WHERE name = ?`, name).Scan(&actorID); {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ErrActorNotFound
	case err != nil:
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM actor_user_tags WHERE actor_id = ?`, actorID); err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO actor_user_tags (actor_id, tag) VALUES (?, ?)`, actorID, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ActorListItemByName returns one list row by exact actors.name (for PATCH response).
func (s *SQLiteStore) ActorListItemByName(ctx context.Context, name string) (contracts.ActorListItemDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return contracts.ActorListItemDTO{}, contracts.ErrActorNotFound
	}
	var id int64
	var avatar string
	var movieCount int
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.avatar, COUNT(ma.movie_id)
		FROM actors a
		LEFT JOIN movie_actors ma ON ma.actor_id = a.id
		WHERE a.name = ?
		GROUP BY a.id, a.name, a.avatar`, name,
	).Scan(&id, &avatar, &movieCount)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ActorListItemDTO{}, contracts.ErrActorNotFound
	case err != nil:
		return contracts.ActorListItemDTO{}, err
	}
	tagsByActor, err := s.loadActorUserTagsForIDs(ctx, []int64{id})
	if err != nil {
		return contracts.ActorListItemDTO{}, err
	}
	return contracts.ActorListItemDTO{
		Name:       name,
		AvatarURL:  strings.TrimSpace(avatar),
		MovieCount: movieCount,
		UserTags:   tagsByActor[id],
	}, nil
}
