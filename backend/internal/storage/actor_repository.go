package storage

import (
	"context"
	"database/sql"
	"errors"
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
		SELECT name, avatar, summary, homepage, provider, provider_actor_id, height, birthday, profile_updated_at
		FROM actors WHERE name = ?`, name,
	).Scan(
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
