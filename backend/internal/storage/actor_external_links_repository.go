package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"curated-backend/internal/contracts"
)

var ErrInvalidActorExternalLinks = errors.New("invalid actor external links")

const maxActorExternalLinks = 16
const maxActorExternalLinkLength = 2048

func NormalizeActorExternalLinksForPatch(raw []string) ([]string, error) {
	if raw == nil {
		raw = []string{}
	}
	if len(raw) > maxActorExternalLinks {
		return nil, fmt.Errorf("%w: at most %d links per actor", ErrInvalidActorExternalLinks, maxActorExternalLinks)
	}

	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if len(item) > maxActorExternalLinkLength {
			return nil, fmt.Errorf("%w: url longer than %d characters", ErrInvalidActorExternalLinks, maxActorExternalLinkLength)
		}
		parsed, err := url.ParseRequestURI(item)
		if err != nil || parsed == nil || strings.TrimSpace(parsed.Host) == "" {
			return nil, fmt.Errorf("%w: invalid url %q", ErrInvalidActorExternalLinks, item)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidActorExternalLinks, parsed.Scheme)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func (s *SQLiteStore) loadActorExternalLinksForIDs(ctx context.Context, ids []int64) (map[int64][]string, error) {
	out := make(map[int64][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT actor_id, url FROM actor_external_links WHERE actor_id IN (%s) ORDER BY actor_id, sort_order, id`,
		placeholders,
	), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var actorID int64
		var item string
		if err := rows.Scan(&actorID, &item); err != nil {
			return nil, err
		}
		out[actorID] = append(out[actorID], item)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceActorExternalLinksByName(ctx context.Context, name string, rawLinks []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return contracts.ErrActorNotFound
	}

	links, err := NormalizeActorExternalLinksForPatch(rawLinks)
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

	if _, err := tx.ExecContext(ctx, `DELETE FROM actor_external_links WHERE actor_id = ?`, actorID); err != nil {
		return err
	}
	for i, item := range links {
		now := nowUTC()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO actor_external_links (actor_id, url, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			actorID, item, i, now, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
