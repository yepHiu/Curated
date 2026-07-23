package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

const maxActorMergeAuditPageSize = 100

type actorMergeQueryer interface {
	actorIdentityQueryer
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type actorMergeRow struct {
	id                   int64
	name                 string
	avatar               string
	avatarLocalPath      string
	avatarLastHTTPStatus int
	avatarLastError      string
	avatarLastFetchedAt  string
	summary              string
	homepage             string
	provider             string
	providerActorID      string
	height               int
	birthday             string
	profileUpdatedAt     string
	aliases              []string
	movieIDs             []string
	userTags             []string
	externalLinks        []string
}

type actorMergeBuild struct {
	preview             contracts.ActorMergePreviewDTO
	source              actorMergeRow
	target              actorMergeRow
	feedbackRows        []actorMergeFeedbackRow
	curatedFrameUpdates []actorMergeCuratedFrameUpdate
}

type actorMergeFeedbackRow struct {
	ID               string `json:"id"`
	TargetValue      string `json:"targetValue"`
	NormalizedTarget string `json:"normalizedTarget"`
	CreatedAt        string `json:"createdAt"`
	Side             string `json:"side"`
}

type actorMergeCuratedFrameUpdate struct {
	ID           string `json:"id"`
	OriginalJSON string `json:"originalJson"`
	ResultJSON   string `json:"resultJson"`
}

func (s *SQLiteStore) PreviewActorMerge(ctx context.Context, req contracts.ActorMergePreviewRequest) (contracts.ActorMergePreviewDTO, error) {
	build, err := buildActorMerge(ctx, s.db, req.SourceName, req.TargetName)
	if err != nil {
		return contracts.ActorMergePreviewDTO{}, err
	}
	return build.preview, nil
}

func buildActorMerge(ctx context.Context, q actorMergeQueryer, sourceName, targetName string) (actorMergeBuild, error) {
	sourceName = strings.TrimSpace(sourceName)
	targetName = strings.TrimSpace(targetName)
	if sourceName == "" || targetName == "" {
		return actorMergeBuild{}, fmt.Errorf("%w: sourceName and targetName are required", contracts.ErrActorMergeInvalid)
	}

	source, err := loadExactActorMergeRow(ctx, q, sourceName)
	if err != nil {
		if !errors.Is(err, contracts.ErrActorMergeNotFound) {
			return actorMergeBuild{}, err
		}
		resolved, resolveErr := resolveActorIdentity(ctx, q, sourceName)
		if resolveErr == nil && resolved.WasAlias {
			return actorMergeBuild{}, contracts.ErrActorMergeSourceAlias
		}
		if resolveErr != nil && !errors.Is(resolveErr, contracts.ErrActorNotFound) {
			return actorMergeBuild{}, resolveErr
		}
		return actorMergeBuild{}, contracts.ErrActorMergeNotFound
	}

	targetIdentity, err := resolveActorIdentity(ctx, q, targetName)
	if errors.Is(err, contracts.ErrActorNotFound) {
		return actorMergeBuild{}, contracts.ErrActorMergeNotFound
	}
	if err != nil {
		return actorMergeBuild{}, err
	}
	if source.id == targetIdentity.ID {
		return actorMergeBuild{}, contracts.ErrActorMergeSelf
	}
	target, err := loadActorMergeRowByID(ctx, q, targetIdentity.ID)
	if err != nil {
		return actorMergeBuild{}, err
	}

	preview := contracts.ActorMergePreviewDTO{
		Source: contracts.ActorMergeActorRefDTO{
			ID:      source.id,
			Name:    source.name,
			Aliases: append([]string{}, source.aliases...),
		},
		Target: contracts.ActorMergeActorRefDTO{
			ID:      target.id,
			Name:    target.name,
			Aliases: append([]string{}, target.aliases...),
		},
		Movies:          buildActorMergeAssociationSummary(source.movieIDs, target.movieIDs),
		UserTags:        buildActorMergeValuesSummary(source.userTags, target.userTags),
		ExternalLinks:   buildActorMergeValuesSummary(source.externalLinks, target.externalLinks),
		AliasesToMove:   stableUniqueActorIdentityValues(append([]string{source.name}, source.aliases...)),
		ProfileFields:   buildActorMergeProfileFields(source, target),
		BlockingReasons: []contracts.ActorMergeBlockingReasonDTO{},
		CanApply:        true,
	}
	feedbackRows, feedbackSummary, err := loadActorMergeFeedback(ctx, q, source, target)
	if err != nil {
		return actorMergeBuild{}, err
	}
	preview.RecommendationFeedback = feedbackSummary
	curatedFrameUpdates, err := loadActorMergeCuratedFrameUpdates(ctx, q, source, target)
	if err != nil {
		return actorMergeBuild{}, err
	}
	preview.CuratedFramesAffected = len(curatedFrameUpdates)
	for _, field := range preview.ProfileFields {
		if field.Conflict {
			preview.RequiredDecisions = append(preview.RequiredDecisions, field.Field)
		}
	}
	if len(preview.ExternalLinks.Result) > maxActorExternalLinks {
		preview.CanApply = false
		preview.BlockingReasons = append(preview.BlockingReasons, contracts.ActorMergeBlockingReasonDTO{
			Code: contracts.ErrorCodeActorMergeLinkLimit,
			Message: fmt.Sprintf(
				"merged external links would contain %d entries; maximum is %d",
				len(preview.ExternalLinks.Result), maxActorExternalLinks,
			),
		})
	}
	aliasBlocks, err := actorMergeAliasBlockingReasons(ctx, q, source.id, target.id, preview.AliasesToMove)
	if err != nil {
		return actorMergeBuild{}, err
	}
	if len(aliasBlocks) > 0 {
		preview.CanApply = false
		preview.BlockingReasons = append(preview.BlockingReasons, aliasBlocks...)
	}

	token, err := actorMergePreviewToken(preview, source, target, feedbackRows, curatedFrameUpdates)
	if err != nil {
		return actorMergeBuild{}, err
	}
	preview.PreviewToken = token
	return actorMergeBuild{
		preview:             preview,
		source:              source,
		target:              target,
		feedbackRows:        feedbackRows,
		curatedFrameUpdates: curatedFrameUpdates,
	}, nil
}

func loadExactActorMergeRow(ctx context.Context, q actorMergeQueryer, name string) (actorMergeRow, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM actors WHERE name = ?`, strings.TrimSpace(name)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return actorMergeRow{}, contracts.ErrActorMergeNotFound
	}
	if err != nil {
		return actorMergeRow{}, err
	}
	return loadActorMergeRowByID(ctx, q, id)
}

func loadActorMergeRowByID(ctx context.Context, q actorMergeQueryer, id int64) (actorMergeRow, error) {
	var row actorMergeRow
	err := q.QueryRowContext(ctx, `
		SELECT id, name, avatar, avatar_local_path, avatar_last_http_status,
		       avatar_last_error, avatar_last_fetched_at, summary, homepage,
		       provider, provider_actor_id, height, birthday, profile_updated_at
		FROM actors WHERE id = ?`, id,
	).Scan(
		&row.id,
		&row.name,
		&row.avatar,
		&row.avatarLocalPath,
		&row.avatarLastHTTPStatus,
		&row.avatarLastError,
		&row.avatarLastFetchedAt,
		&row.summary,
		&row.homepage,
		&row.provider,
		&row.providerActorID,
		&row.height,
		&row.birthday,
		&row.profileUpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return actorMergeRow{}, contracts.ErrActorMergeNotFound
	}
	if err != nil {
		return actorMergeRow{}, err
	}
	if row.aliases, err = queryActorMergeStrings(ctx, q,
		`SELECT alias FROM actor_aliases WHERE canonical_actor_id = ? ORDER BY id`, id); err != nil {
		return actorMergeRow{}, err
	}
	if row.movieIDs, err = queryActorMergeStrings(ctx, q,
		`SELECT movie_id FROM movie_actors WHERE actor_id = ? ORDER BY movie_id`, id); err != nil {
		return actorMergeRow{}, err
	}
	if row.userTags, err = queryActorMergeStrings(ctx, q,
		`SELECT tag FROM actor_user_tags WHERE actor_id = ? ORDER BY tag`, id); err != nil {
		return actorMergeRow{}, err
	}
	if row.externalLinks, err = queryActorMergeStrings(ctx, q,
		`SELECT url FROM actor_external_links WHERE actor_id = ? ORDER BY sort_order, id`, id); err != nil {
		return actorMergeRow{}, err
	}
	return row, nil
}

func queryActorMergeStrings(ctx context.Context, q actorMergeQueryer, query string, arg any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func buildActorMergeAssociationSummary(source, target []string) contracts.ActorMergeAssociationSummaryDTO {
	targetSet := make(map[string]struct{}, len(target))
	for _, value := range target {
		targetSet[value] = struct{}{}
	}
	duplicates := 0
	for _, value := range source {
		if _, ok := targetSet[value]; ok {
			duplicates++
		}
	}
	return contracts.ActorMergeAssociationSummaryDTO{
		SourceCount:    len(source),
		TargetCount:    len(target),
		DuplicateCount: duplicates,
		ResultCount:    len(source) + len(target) - duplicates,
	}
}

func buildActorMergeValuesSummary(source, target []string) contracts.ActorMergeValuesSummaryDTO {
	return contracts.ActorMergeValuesSummaryDTO{
		Source: append([]string{}, source...),
		Target: append([]string{}, target...),
		Result: stableUniqueValues(append(append([]string{}, target...), source...)),
	}
}

func stableUniqueValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stableUniqueActorIdentityValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := NormalizeActorIdentity(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func buildActorMergeProfileFields(source, target actorMergeRow) []contracts.ActorMergeProfileFieldDTO {
	values := []struct {
		field  string
		source string
		target string
	}{
		{field: "avatarRemoteUrl", source: source.avatar, target: target.avatar},
		{field: "avatarLocalPath", source: source.avatarLocalPath, target: target.avatarLocalPath},
		{field: "summary", source: source.summary, target: target.summary},
		{field: "homepage", source: source.homepage, target: target.homepage},
		{field: "provider", source: source.provider, target: target.provider},
		{field: "providerActorId", source: source.providerActorID, target: target.providerActorID},
		{field: "height", source: actorMergeIntValue(source.height), target: actorMergeIntValue(target.height)},
		{field: "birthday", source: source.birthday, target: target.birthday},
	}
	out := make([]contracts.ActorMergeProfileFieldDTO, 0, len(values))
	for _, value := range values {
		sourceValue := strings.TrimSpace(value.source)
		targetValue := strings.TrimSpace(value.target)
		selection := "target"
		if targetValue == "" && sourceValue != "" {
			selection = "source"
		}
		out = append(out, contracts.ActorMergeProfileFieldDTO{
			Field:            value.field,
			SourceValue:      sourceValue,
			TargetValue:      targetValue,
			DefaultSelection: selection,
			Conflict:         sourceValue != "" && targetValue != "" && sourceValue != targetValue,
		})
	}
	return out
}

func actorMergeIntValue(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func actorMergeAliasBlockingReasons(
	ctx context.Context,
	q actorMergeQueryer,
	sourceID int64,
	targetID int64,
	aliases []string,
) ([]contracts.ActorMergeBlockingReasonDTO, error) {
	out := make([]contracts.ActorMergeBlockingReasonDTO, 0)
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		normalized := NormalizeActorIdentity(alias)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}

		var conflictingName string
		err := q.QueryRowContext(ctx, `
			SELECT name FROM actors
			WHERE normalized_name = ? AND id NOT IN (?, ?)
			ORDER BY id LIMIT 1`, normalized, sourceID, targetID,
		).Scan(&conflictingName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			out = append(out, contracts.ActorMergeBlockingReasonDTO{
				Code:    contracts.ErrorCodeActorMergeConflict,
				Message: fmt.Sprintf("alias %q conflicts with canonical actor %q", alias, conflictingName),
			})
			continue
		}

		var conflictingAlias string
		err = q.QueryRowContext(ctx, `
			SELECT alias FROM actor_aliases
			WHERE normalized_alias = ? AND canonical_actor_id NOT IN (?, ?)
			LIMIT 1`, normalized, sourceID, targetID,
		).Scan(&conflictingAlias)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			out = append(out, contracts.ActorMergeBlockingReasonDTO{
				Code:    contracts.ErrorCodeActorMergeConflict,
				Message: fmt.Sprintf("alias %q conflicts with existing alias %q", alias, conflictingAlias),
			})
		}
	}
	return out, nil
}

func actorMergeIdentityNorms(actor actorMergeRow) map[string]struct{} {
	out := make(map[string]struct{}, len(actor.aliases)+1)
	for _, value := range append([]string{actor.name}, actor.aliases...) {
		if normalized := NormalizeActorIdentity(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func actorMergeExactNames(actor actorMergeRow) map[string]struct{} {
	out := make(map[string]struct{}, len(actor.aliases)+1)
	for _, value := range append([]string{actor.name}, actor.aliases...) {
		if value = strings.TrimSpace(value); value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func loadActorMergeFeedback(
	ctx context.Context,
	q actorMergeQueryer,
	source actorMergeRow,
	target actorMergeRow,
) ([]actorMergeFeedbackRow, contracts.ActorMergeAssociationSummaryDTO, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, target_value, normalized_target, created_at
		FROM homepage_recommendation_feedback
		WHERE action = 'less' AND target_type = 'actor'
		ORDER BY created_at DESC, id ASC`)
	if err != nil {
		return nil, contracts.ActorMergeAssociationSummaryDTO{}, err
	}
	defer rows.Close()
	sourceNorms := actorMergeIdentityNorms(source)
	targetNorms := actorMergeIdentityNorms(target)
	sourceExact := actorMergeExactNames(source)
	targetExact := actorMergeExactNames(target)
	matched := make([]actorMergeFeedbackRow, 0)
	sourceCount := 0
	targetCount := 0
	for rows.Next() {
		var row actorMergeFeedbackRow
		if err := rows.Scan(&row.ID, &row.TargetValue, &row.NormalizedTarget, &row.CreatedAt); err != nil {
			return nil, contracts.ActorMergeAssociationSummaryDTO{}, err
		}
		exact := strings.TrimSpace(row.TargetValue)
		normalized := NormalizeActorIdentity(row.TargetValue)
		if normalized == "" {
			normalized = NormalizeActorIdentity(row.NormalizedTarget)
		}
		switch {
		case containsActorMergeValue(sourceExact, exact):
			row.Side = "source"
		case containsActorMergeValue(targetExact, exact):
			row.Side = "target"
		case containsActorMergeValue(sourceNorms, normalized):
			row.Side = "source"
		case containsActorMergeValue(targetNorms, normalized):
			row.Side = "target"
		default:
			continue
		}
		if row.Side == "source" {
			sourceCount++
		} else {
			targetCount++
		}
		matched = append(matched, row)
	}
	if err := rows.Err(); err != nil {
		return nil, contracts.ActorMergeAssociationSummaryDTO{}, err
	}
	resultCount := 0
	duplicates := 0
	if len(matched) > 0 {
		resultCount = 1
		duplicates = len(matched) - 1
	}
	return matched, contracts.ActorMergeAssociationSummaryDTO{
		SourceCount:    sourceCount,
		TargetCount:    targetCount,
		DuplicateCount: duplicates,
		ResultCount:    resultCount,
	}, nil
}

func containsActorMergeValue(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func loadActorMergeCuratedFrameUpdates(
	ctx context.Context,
	q actorMergeQueryer,
	source actorMergeRow,
	target actorMergeRow,
) ([]actorMergeCuratedFrameUpdate, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, actors_json FROM curated_frames ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sourceNorms := actorMergeIdentityNorms(source)
	updates := make([]actorMergeCuratedFrameUpdate, 0)
	for rows.Next() {
		var id string
		var actorsJSON string
		if err := rows.Scan(&id, &actorsJSON); err != nil {
			return nil, err
		}
		var actors []string
		if err := json.Unmarshal([]byte(actorsJSON), &actors); err != nil {
			return nil, fmt.Errorf("decode curated frame %s actors: %w", id, err)
		}
		changed := false
		for index, actor := range actors {
			if _, ok := sourceNorms[NormalizeActorIdentity(actor)]; ok {
				actors[index] = target.name
				changed = true
			}
		}
		if !changed {
			continue
		}
		actors = stableUniqueActorIdentityValues(actors)
		resultJSON, err := json.Marshal(actors)
		if err != nil {
			return nil, err
		}
		updates = append(updates, actorMergeCuratedFrameUpdate{
			ID:           id,
			OriginalJSON: actorsJSON,
			ResultJSON:   string(resultJSON),
		})
	}
	return updates, rows.Err()
}

func actorMergePreviewToken(
	preview contracts.ActorMergePreviewDTO,
	source actorMergeRow,
	target actorMergeRow,
	feedbackRows []actorMergeFeedbackRow,
	curatedFrameUpdates []actorMergeCuratedFrameUpdate,
) (string, error) {
	preview.PreviewToken = ""
	tokenState := struct {
		Preview             contracts.ActorMergePreviewDTO `json:"preview"`
		SourceMovieIDs      []string                       `json:"sourceMovieIds"`
		TargetMovieIDs      []string                       `json:"targetMovieIds"`
		SourceInternal      []any                          `json:"sourceInternal"`
		TargetInternal      []any                          `json:"targetInternal"`
		FeedbackRows        []actorMergeFeedbackRow        `json:"feedbackRows"`
		CuratedFrameUpdates []actorMergeCuratedFrameUpdate `json:"curatedFrameUpdates"`
	}{
		Preview:        preview,
		SourceMovieIDs: source.movieIDs,
		TargetMovieIDs: target.movieIDs,
		SourceInternal: []any{
			source.id, source.name, source.avatar, source.avatarLocalPath,
			source.avatarLastHTTPStatus, source.avatarLastError, source.avatarLastFetchedAt,
			source.summary, source.homepage, source.provider, source.providerActorID,
			source.height, source.birthday, source.profileUpdatedAt,
			source.aliases, source.userTags, source.externalLinks,
		},
		TargetInternal: []any{
			target.id, target.name, target.avatar, target.avatarLocalPath,
			target.avatarLastHTTPStatus, target.avatarLastError, target.avatarLastFetchedAt,
			target.summary, target.homepage, target.provider, target.providerActorID,
			target.height, target.birthday, target.profileUpdatedAt,
			target.aliases, target.userTags, target.externalLinks,
		},
		FeedbackRows:        feedbackRows,
		CuratedFrameUpdates: curatedFrameUpdates,
	}
	raw, err := json.Marshal(tokenState)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *SQLiteStore) ApplyActorMerge(ctx context.Context, req contracts.ApplyActorMergeRequest, now time.Time) (contracts.ActorMergeAuditDTO, error) {
	if !req.Confirm || strings.TrimSpace(req.PreviewToken) == "" {
		return contracts.ActorMergeAuditDTO{}, fmt.Errorf("%w: confirm and previewToken are required", contracts.ErrActorMergeInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	build, err := buildActorMerge(ctx, tx, req.SourceName, req.TargetName)
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	providedToken := strings.TrimSpace(req.PreviewToken)
	if len(providedToken) != len(build.preview.PreviewToken) ||
		subtle.ConstantTimeCompare([]byte(providedToken), []byte(build.preview.PreviewToken)) != 1 {
		return contracts.ActorMergeAuditDTO{}, contracts.ErrActorMergeStalePreview
	}
	if !build.preview.CanApply {
		for _, reason := range build.preview.BlockingReasons {
			if reason.Code == contracts.ErrorCodeActorMergeLinkLimit {
				return contracts.ActorMergeAuditDTO{}, fmt.Errorf("%w: %s", contracts.ErrActorMergeLinkLimit, reason.Message)
			}
		}
		return contracts.ActorMergeAuditDTO{}, contracts.ErrActorMergeConflict
	}

	decisions, err := validateActorMergeProfileDecisions(build.preview.ProfileFields, req.ProfileDecisions)
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	mergedProfile, avatarStatus, err := selectActorMergeProfile(build.source, build.target, build.preview.ProfileFields, decisions)
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE actors SET
			avatar = ?, avatar_local_path = ?, avatar_last_http_status = ?,
			avatar_last_error = ?, avatar_last_fetched_at = ?, summary = ?, homepage = ?,
			provider = ?, provider_actor_id = ?, height = ?, birthday = ?, profile_updated_at = ?
		WHERE id = ?`,
		mergedProfile["avatarRemoteUrl"],
		mergedProfile["avatarLocalPath"],
		avatarStatus.httpStatus,
		avatarStatus.lastError,
		avatarStatus.lastFetchedAt,
		mergedProfile["summary"],
		mergedProfile["homepage"],
		mergedProfile["provider"],
		mergedProfile["providerActorId"],
		actorMergeParsedInt(mergedProfile["height"]),
		mergedProfile["birthday"],
		latestActorMergeTimestamp(build.source.profileUpdatedAt, build.target.profileUpdatedAt),
		build.target.id,
	); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO movie_actors (movie_id, actor_id)
		SELECT movie_id, ? FROM movie_actors WHERE actor_id = ?`, build.target.id, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM movie_actors WHERE actor_id = ?`, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO actor_user_tags (actor_id, tag)
		SELECT ?, tag FROM actor_user_tags WHERE actor_id = ?`, build.target.id, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM actor_user_tags WHERE actor_id = ?`, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM actor_external_links WHERE actor_id IN (?, ?)`, build.source.id, build.target.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	for index, link := range build.preview.ExternalLinks.Result {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO actor_external_links (actor_id, url, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, build.target.id, link, index, nowText, nowText); err != nil {
			return contracts.ActorMergeAuditDTO{}, err
		}
	}
	if err := applyActorMergeRecommendationFeedback(ctx, tx, build.feedbackRows, build.target.name, nowText); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	for _, update := range build.curatedFrameUpdates {
		result, err := tx.ExecContext(ctx,
			`UPDATE curated_frames SET actors_json = ? WHERE id = ? AND actors_json = ?`,
			update.ResultJSON, update.ID, update.OriginalJSON,
		)
		if err != nil {
			return contracts.ActorMergeAuditDTO{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return contracts.ActorMergeAuditDTO{}, err
		}
		if affected != 1 {
			return contracts.ActorMergeAuditDTO{}, contracts.ErrActorMergeStalePreview
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE actor_aliases
		SET canonical_actor_id = ?, updated_at = ?
		WHERE canonical_actor_id = ?`, build.target.id, nowText, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	for _, alias := range build.preview.AliasesToMove {
		normalized := NormalizeActorIdentity(alias)
		if normalized == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO actor_aliases (
				canonical_actor_id, alias, normalized_alias, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?)`, build.target.id, alias, normalized, nowText, nowText); err != nil {
			return contracts.ActorMergeAuditDTO{}, err
		}
		var ownerID int64
		if err := tx.QueryRowContext(ctx,
			`SELECT canonical_actor_id FROM actor_aliases WHERE normalized_alias = ?`, normalized,
		).Scan(&ownerID); err != nil {
			return contracts.ActorMergeAuditDTO{}, err
		}
		if ownerID != build.target.id {
			return contracts.ActorMergeAuditDTO{}, contracts.ErrActorMergeConflict
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM actors WHERE id = ?`, build.source.id); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}

	auditID, err := newActorMergeAuditID()
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	audit := contracts.ActorMergeAuditDTO{
		ID:            auditID,
		SourceActorID: build.source.id,
		TargetActorID: build.target.id,
		SourceName:    build.source.name,
		TargetName:    build.target.name,
		PreviewToken:  build.preview.PreviewToken,
		AppliedAt:     nowText,
		Summary: contracts.ActorMergeAuditSummaryDTO{
			Movies:                 build.preview.Movies,
			UserTags:               append([]string{}, build.preview.UserTags.Result...),
			ExternalLinks:          append([]string{}, build.preview.ExternalLinks.Result...),
			Aliases:                stableUniqueActorIdentityValues(append(append([]string{}, build.target.aliases...), build.preview.AliasesToMove...)),
			RecommendationFeedback: build.preview.RecommendationFeedback,
			CuratedFramesAffected:  build.preview.CuratedFramesAffected,
			ProfileDecisions:       decisions,
		},
	}
	summaryJSON, err := json.Marshal(audit.Summary)
	if err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO actor_merge_audits (
			id, source_actor_id, target_actor_id, source_name, target_name,
			preview_token, summary_json, applied_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		audit.ID,
		audit.SourceActorID,
		audit.TargetActorID,
		audit.SourceName,
		audit.TargetName,
		audit.PreviewToken,
		string(summaryJSON),
		audit.AppliedAt,
	); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return contracts.ActorMergeAuditDTO{}, err
	}
	return audit, nil
}

func applyActorMergeRecommendationFeedback(
	ctx context.Context,
	tx *sql.Tx,
	rows []actorMergeFeedbackRow,
	targetName string,
	nowText string,
) error {
	if len(rows) == 0 {
		return nil
	}
	keeper := rows[0]
	for _, row := range rows {
		if row.Side == "target" {
			keeper = row
			break
		}
	}
	for _, row := range rows {
		if row.ID == keeper.ID {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM homepage_recommendation_feedback WHERE id = ?`, row.ID); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE homepage_recommendation_feedback
		SET target_value = ?, normalized_target = ?, updated_at = ?
		WHERE id = ?`, targetName, NormalizeActorIdentity(targetName), nowText, keeper.ID)
	return err
}

func validateActorMergeProfileDecisions(
	fields []contracts.ActorMergeProfileFieldDTO,
	raw map[string]string,
) (map[string]string, error) {
	if raw == nil {
		raw = map[string]string{}
	}
	allowed := make(map[string]contracts.ActorMergeProfileFieldDTO, len(fields))
	for _, field := range fields {
		allowed[field.Field] = field
	}
	for field, selection := range raw {
		if _, ok := allowed[field]; !ok {
			return nil, fmt.Errorf("%w: unknown profile field %q", contracts.ErrActorMergeInvalid, field)
		}
		if selection != "source" && selection != "target" {
			return nil, fmt.Errorf("%w: profile decision for %q must be source or target", contracts.ErrActorMergeInvalid, field)
		}
	}
	out := make(map[string]string, len(fields))
	for _, field := range fields {
		selection := raw[field.Field]
		if selection == "" {
			if field.Conflict {
				return nil, fmt.Errorf("%w: profile decision required for %q", contracts.ErrActorMergeConflict, field.Field)
			}
			selection = field.DefaultSelection
		}
		out[field.Field] = selection
	}
	return out, nil
}

type actorMergeAvatarStatus struct {
	httpStatus    int
	lastError     string
	lastFetchedAt string
}

func selectActorMergeProfile(
	source actorMergeRow,
	target actorMergeRow,
	fields []contracts.ActorMergeProfileFieldDTO,
	decisions map[string]string,
) (map[string]string, actorMergeAvatarStatus, error) {
	out := make(map[string]string, len(fields))
	for _, field := range fields {
		switch decisions[field.Field] {
		case "source":
			out[field.Field] = field.SourceValue
		case "target":
			out[field.Field] = field.TargetValue
		default:
			return nil, actorMergeAvatarStatus{}, contracts.ErrActorMergeInvalid
		}
	}
	avatarStatus := actorMergeAvatarStatus{
		httpStatus:    target.avatarLastHTTPStatus,
		lastError:     target.avatarLastError,
		lastFetchedAt: target.avatarLastFetchedAt,
	}
	if decisions["avatarLocalPath"] == "source" {
		avatarStatus = actorMergeAvatarStatus{
			httpStatus:    source.avatarLastHTTPStatus,
			lastError:     source.avatarLastError,
			lastFetchedAt: source.avatarLastFetchedAt,
		}
	}
	return out, avatarStatus, nil
}

func actorMergeParsedInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func latestActorMergeTimestamp(values ...string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	sort.Strings(clean)
	return clean[len(clean)-1]
}

func newActorMergeAuditID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "amrg_" + hex.EncodeToString(raw[:]), nil
}

func (s *SQLiteStore) ListActorMergeAudits(ctx context.Context, limit, offset int) (contracts.ActorMergeAuditListDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > maxActorMergeAuditPageSize {
		limit = maxActorMergeAuditPageSize
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM actor_merge_audits`).Scan(&total); err != nil {
		return contracts.ActorMergeAuditListDTO{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_actor_id, target_actor_id, source_name, target_name,
		       preview_token, summary_json, applied_at
		FROM actor_merge_audits
		ORDER BY applied_at DESC, id ASC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return contracts.ActorMergeAuditListDTO{}, err
	}
	defer rows.Close()
	items := make([]contracts.ActorMergeAuditDTO, 0)
	for rows.Next() {
		var item contracts.ActorMergeAuditDTO
		var summaryJSON string
		if err := rows.Scan(
			&item.ID,
			&item.SourceActorID,
			&item.TargetActorID,
			&item.SourceName,
			&item.TargetName,
			&item.PreviewToken,
			&summaryJSON,
			&item.AppliedAt,
		); err != nil {
			return contracts.ActorMergeAuditListDTO{}, err
		}
		if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
			return contracts.ActorMergeAuditListDTO{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return contracts.ActorMergeAuditListDTO{}, err
	}
	return contracts.ActorMergeAuditListDTO{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}
