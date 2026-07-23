package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"curated-backend/internal/contracts"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type actorIdentityQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type resolvedActorIdentity struct {
	ID       int64
	Name     string
	WasAlias bool
}

// NormalizeActorIdentity applies the one canonical comparison form used by
// actor names, aliases, merge previews, and metadata ingestion.
func NormalizeActorIdentity(value string) string {
	value = norm.NFKC.String(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	return cases.Fold().String(value)
}

func resolveActorIdentity(ctx context.Context, q actorIdentityQueryer, rawName string) (resolvedActorIdentity, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return resolvedActorIdentity{}, contracts.ErrActorNotFound
	}

	var out resolvedActorIdentity
	err := q.QueryRowContext(ctx, `SELECT id, name FROM actors WHERE name = ?`, name).Scan(&out.ID, &out.Name)
	switch {
	case err == nil:
		return out, nil
	case !errors.Is(err, sql.ErrNoRows):
		return resolvedActorIdentity{}, err
	}

	normalized := NormalizeActorIdentity(name)
	if normalized == "" {
		return resolvedActorIdentity{}, contracts.ErrActorNotFound
	}
	err = q.QueryRowContext(ctx, `
		SELECT a.id, a.name
		FROM actor_aliases aa
		JOIN actors a ON a.id = aa.canonical_actor_id
		WHERE aa.normalized_alias = ?`, normalized,
	).Scan(&out.ID, &out.Name)
	switch {
	case err == nil:
		out.WasAlias = true
		return out, nil
	case !errors.Is(err, sql.ErrNoRows):
		return resolvedActorIdentity{}, err
	}

	// Existing libraries can contain multiple pre-merge names that normalize to
	// the same identity. Resolve deterministically to prevent creating a third
	// fragment; the merge workbench remains the explicit way to consolidate them.
	err = q.QueryRowContext(ctx, `
		SELECT id, name
		FROM actors
		WHERE normalized_name = ?
		ORDER BY id ASC
		LIMIT 1`, normalized,
	).Scan(&out.ID, &out.Name)
	switch {
	case err == nil:
		return out, nil
	case errors.Is(err, sql.ErrNoRows):
		return resolvedActorIdentity{}, contracts.ErrActorNotFound
	default:
		return resolvedActorIdentity{}, err
	}
}

// ResolveActorCanonicalName returns the canonical display name for either a
// canonical name or any persisted alias.
func (s *SQLiteStore) ResolveActorCanonicalName(ctx context.Context, rawName string) (string, error) {
	identity, err := resolveActorIdentity(ctx, s.db, rawName)
	if err != nil {
		return "", err
	}
	return identity.Name, nil
}

func (s *SQLiteStore) backfillActorNormalizedNames(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM actors ORDER BY id`)
	if err != nil {
		return err
	}
	type actorNameRow struct {
		id   int64
		name string
	}
	items := make([]actorNameRow, 0)
	for rows.Next() {
		var item actorNameRow
		if err := rows.Scan(&item.id, &item.name); err != nil {
			_ = rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range items {
		if _, err := tx.ExecContext(ctx,
			`UPDATE actors SET normalized_name = ? WHERE id = ?`,
			NormalizeActorIdentity(item.name), item.id,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) backfillActorFeedbackNormalizedTargets(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, target_value
		FROM homepage_recommendation_feedback
		WHERE action = 'less' AND target_type = 'actor'
		ORDER BY updated_at DESC, created_at DESC, id ASC`)
	if err != nil {
		return err
	}
	type feedbackTargetRow struct {
		id     string
		target string
	}
	items := make([]feedbackTargetRow, 0)
	for rows.Next() {
		var item feedbackTargetRow
		if err := rows.Scan(&item.id, &item.target); err != nil {
			_ = rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	seen := make(map[string]string, len(items))
	keepNormalized := make(map[string]string, len(items))
	duplicateIDs := make([]string, 0)
	for _, item := range items {
		normalized := NormalizeActorIdentity(item.target)
		if normalized == "" {
			continue
		}
		if _, duplicate := seen[normalized]; duplicate {
			duplicateIDs = append(duplicateIDs, item.id)
			continue
		}
		seen[normalized] = item.id
		keepNormalized[item.id] = normalized
	}
	for _, id := range duplicateIDs {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM homepage_recommendation_feedback WHERE id = ?`, id); err != nil {
			return err
		}
	}
	for id, normalized := range keepNormalized {
		if _, err := tx.ExecContext(ctx, `
			UPDATE homepage_recommendation_feedback
			SET normalized_target = ?
			WHERE id = ?`, normalized, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
