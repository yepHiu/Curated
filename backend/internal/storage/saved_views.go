package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

const MaxSavedViews = 50

var (
	ErrSavedViewNotFound     = errors.New("saved view not found")
	ErrSavedViewNameConflict = errors.New("saved view name conflict")
	ErrSavedViewLimit        = errors.New("saved view limit reached")
	ErrSavedViewOrderInvalid = errors.New("saved view order is incomplete or duplicated")
)

func normalizedSavedViewName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func savedViewTime(now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano)
}

func (s *SQLiteStore) ListSavedViews(ctx context.Context) ([]contracts.SavedViewDTO, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, schema_version, filters_json, sort_order, created_at, updated_at
		FROM library_saved_views
		ORDER BY sort_order ASC, created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.SavedViewDTO, 0)
	for rows.Next() {
		var item contracts.SavedViewDTO
		var schemaVersion int
		var filtersJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&schemaVersion,
			&filtersJSON,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(filtersJSON), &item.Filters); err != nil {
			return nil, fmt.Errorf("decode saved view %s filters: %w", item.ID, err)
		}
		if item.Filters.SchemaVersion != schemaVersion {
			return nil, fmt.Errorf(
				"saved view %s schema mismatch: row=%d payload=%d",
				item.ID,
				schemaVersion,
				item.Filters.SchemaVersion,
			)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStore) CreateSavedView(
	ctx context.Context,
	id string,
	name string,
	filters contracts.SavedViewFiltersV1,
	now time.Time,
) (contracts.SavedViewDTO, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_saved_views`).Scan(&count); err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if count >= MaxSavedViews {
		return contracts.SavedViewDTO{}, ErrSavedViewLimit
	}

	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	timestamp := savedViewTime(now)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO library_saved_views (
			id, name, normalized_name, schema_version, filters_json, sort_order, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		name,
		normalizedSavedViewName(name),
		contracts.SavedViewSchemaVersion,
		string(filtersJSON),
		count,
		timestamp,
		timestamp,
	)
	if isSQLiteConstraint(err) {
		return contracts.SavedViewDTO{}, ErrSavedViewNameConflict
	}
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if err := saveAIApplyReceiptTx(ctx, tx, contracts.SavedViewDTO{ID: id, Name: name, Filters: filters, SortOrder: count, CreatedAt: timestamp, UpdatedAt: timestamp}); err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return contracts.SavedViewDTO{}, err
	}
	return contracts.SavedViewDTO{
		ID:        id,
		Name:      name,
		Filters:   filters,
		SortOrder: count,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}, nil
}

func (s *SQLiteStore) UpdateSavedView(
	ctx context.Context,
	id string,
	name string,
	filters contracts.SavedViewFiltersV1,
	now time.Time,
) (contracts.SavedViewDTO, error) {
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE library_saved_views
		SET name = ?, normalized_name = ?, schema_version = ?, filters_json = ?, updated_at = ?
		WHERE id = ?`,
		name,
		normalizedSavedViewName(name),
		contracts.SavedViewSchemaVersion,
		string(filtersJSON),
		savedViewTime(now),
		id,
	)
	if isSQLiteConstraint(err) {
		return contracts.SavedViewDTO{}, ErrSavedViewNameConflict
	}
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if affected == 0 {
		return contracts.SavedViewDTO{}, ErrSavedViewNotFound
	}
	return s.GetSavedView(ctx, id)
}

func (s *SQLiteStore) GetSavedView(ctx context.Context, id string) (contracts.SavedViewDTO, error) {
	var item contracts.SavedViewDTO
	var schemaVersion int
	var filtersJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, schema_version, filters_json, sort_order, created_at, updated_at
		FROM library_saved_views WHERE id = ?`, id).Scan(
		&item.ID,
		&item.Name,
		&schemaVersion,
		&filtersJSON,
		&item.SortOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.SavedViewDTO{}, ErrSavedViewNotFound
	}
	if err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if err := json.Unmarshal([]byte(filtersJSON), &item.Filters); err != nil {
		return contracts.SavedViewDTO{}, err
	}
	if item.Filters.SchemaVersion != schemaVersion {
		return contracts.SavedViewDTO{}, fmt.Errorf("saved view %s schema mismatch", id)
	}
	return item, nil
}

func (s *SQLiteStore) DeleteSavedView(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `DELETE FROM library_saved_views WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSavedViewNotFound
	}
	if err := compactSavedViewOrder(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func compactSavedViewOrder(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM library_saved_views
		ORDER BY sort_order ASC, created_at ASC, id ASC`)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for index, id := range ids {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE library_saved_views SET sort_order = ? WHERE id = ?`,
			index,
			id,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) ReorderSavedViews(ctx context.Context, ids []string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `SELECT id FROM library_saved_views`)
	if err != nil {
		return err
	}
	existing := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		existing[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(existing) != len(ids) {
		return ErrSavedViewOrderInvalid
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := existing[id]; !ok {
			return ErrSavedViewOrderInvalid
		}
		if _, duplicate := seen[id]; duplicate {
			return ErrSavedViewOrderInvalid
		}
		seen[id] = struct{}{}
	}
	timestamp := savedViewTime(now)
	for index, id := range ids {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE library_saved_views SET sort_order = ?, updated_at = ? WHERE id = ?`,
			index,
			timestamp,
			id,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
