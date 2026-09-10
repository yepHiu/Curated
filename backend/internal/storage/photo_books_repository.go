package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

var ErrPhotoBookNotFound = errors.New("photo book not found")

type PhotoBookUpsert struct {
	LibraryPathID  string
	Location       string
	SourceFileName string
	Title          string
	FileSize       int64
	FileModifiedAt string
	PageCount      int
}

func (s *SQLiteStore) UpsertPhotoBook(ctx context.Context, in PhotoBookUpsert) (contracts.PhotoBookDetailDTO, error) {
	location := filepath.Clean(strings.TrimSpace(in.Location))
	if location == "" || location == "." {
		return contracts.PhotoBookDetailDTO{}, fmt.Errorf("location is required")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSuffix(in.SourceFileName, filepath.Ext(in.SourceFileName))
	}
	if title == "" {
		title = filepath.Base(location)
	}
	sourceFileName := strings.TrimSpace(in.SourceFileName)
	if sourceFileName == "" {
		sourceFileName = filepath.Base(location)
	}
	id := "photo-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := nowUTC()
	addedAt := time.Now().UTC().Format("2006-01-02")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO photo_books (
			id, library_path_id, title, source_file_name, location, file_size, file_modified_at,
			page_count, cover_page_index, added_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
		ON CONFLICT(location) DO UPDATE SET
			library_path_id = excluded.library_path_id,
			title = excluded.title,
			source_file_name = excluded.source_file_name,
			file_size = excluded.file_size,
			file_modified_at = excluded.file_modified_at,
			page_count = excluded.page_count,
			updated_at = excluded.updated_at`,
		id,
		nullEmptyString(in.LibraryPathID),
		title,
		sourceFileName,
		location,
		in.FileSize,
		strings.TrimSpace(in.FileModifiedAt),
		in.PageCount,
		addedAt,
		ts,
	)
	if err != nil {
		return contracts.PhotoBookDetailDTO{}, err
	}
	var actualID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM photo_books WHERE location = ?`, location).Scan(&actualID); err != nil {
		return contracts.PhotoBookDetailDTO{}, err
	}
	return s.GetPhotoBookDetail(ctx, actualID)
}

func (s *SQLiteStore) GetPhotoBookDetail(ctx context.Context, id string) (contracts.PhotoBookDetailDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.PhotoBookDetailDTO{}, ErrPhotoBookNotFound
	}
	var item contracts.PhotoBookListItemDTO
	var favoriteInt int
	var rating sql.NullFloat64
	err := s.db.QueryRowContext(ctx,
		`SELECT pb.id, pb.title, pb.source_file_name, pb.location, pb.page_count, pb.is_favorite,
		        pb.user_rating, pb.added_at, pb.updated_at, pb.last_viewed_at, pb.completed_at,
		        COALESCE(pvp.current_page_index, 0)
		   FROM photo_books pb
		   LEFT JOIN photo_viewing_progress pvp ON pvp.photo_id = pb.id
		  WHERE pb.id = ?`,
		id,
	).Scan(
		&item.ID,
		&item.Title,
		&item.SourceFileName,
		&item.Location,
		&item.PageCount,
		&favoriteInt,
		&rating,
		&item.AddedAt,
		&item.UpdatedAt,
		&item.LastViewedAt,
		&item.CompletedAt,
		&item.CurrentPageIndex,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.PhotoBookDetailDTO{}, ErrPhotoBookNotFound
		}
		return contracts.PhotoBookDetailDTO{}, err
	}
	item.IsFavorite = favoriteInt != 0
	if rating.Valid {
		item.Rating = &rating.Float64
	}
	tags, err := s.listPhotoTags(ctx, id)
	if err != nil {
		return contracts.PhotoBookDetailDTO{}, err
	}
	pages, err := s.ListPhotoPages(ctx, id)
	if err != nil {
		return contracts.PhotoBookDetailDTO{}, err
	}
	item.Tags = tags
	return contracts.PhotoBookDetailDTO{PhotoBookListItemDTO: item, Pages: pages}, nil
}

func (s *SQLiteStore) GetPhotoBookByLocation(ctx context.Context, location string) (contracts.PhotoBookDetailDTO, error) {
	location = filepath.Clean(strings.TrimSpace(location))
	if location == "" || location == "." {
		return contracts.PhotoBookDetailDTO{}, ErrPhotoBookNotFound
	}
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM photo_books WHERE location = ?`, location).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.PhotoBookDetailDTO{}, ErrPhotoBookNotFound
		}
		return contracts.PhotoBookDetailDTO{}, err
	}
	return s.GetPhotoBookDetail(ctx, id)
}

func (s *SQLiteStore) ListPhotoBooks(ctx context.Context, req contracts.ListPhotoBooksRequest) (contracts.PhotoBooksPageDTO, error) {
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	where, args := photoBookWhere(req)
	var total int
	countQuery := `SELECT COUNT(DISTINCT pb.id) FROM photo_books pb ` + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return contracts.PhotoBooksPageDTO{}, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT pb.id, pb.title, pb.source_file_name, pb.location, pb.page_count, pb.is_favorite,
		        pb.user_rating, pb.added_at, pb.updated_at, pb.last_viewed_at, pb.completed_at,
		        COALESCE(pvp.current_page_index, 0)
		   FROM photo_books pb
		   LEFT JOIN photo_viewing_progress pvp ON pvp.photo_id = pb.id `+
			where+
			` GROUP BY pb.id
		      ORDER BY pb.added_at DESC, pb.id ASC
		      LIMIT ? OFFSET ?`,
		listArgs...,
	)
	if err != nil {
		return contracts.PhotoBooksPageDTO{}, err
	}

	items := make([]contracts.PhotoBookListItemDTO, 0)
	for rows.Next() {
		var item contracts.PhotoBookListItemDTO
		var favoriteInt int
		var rating sql.NullFloat64
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.SourceFileName,
			&item.Location,
			&item.PageCount,
			&favoriteInt,
			&rating,
			&item.AddedAt,
			&item.UpdatedAt,
			&item.LastViewedAt,
			&item.CompletedAt,
			&item.CurrentPageIndex,
		); err != nil {
			return contracts.PhotoBooksPageDTO{}, err
		}
		item.IsFavorite = favoriteInt != 0
		if rating.Valid {
			item.Rating = &rating.Float64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return contracts.PhotoBooksPageDTO{}, err
	}
	if err := rows.Close(); err != nil {
		return contracts.PhotoBooksPageDTO{}, err
	}
	for i := range items {
		tags, err := s.listPhotoTags(ctx, items[i].ID)
		if err != nil {
			return contracts.PhotoBooksPageDTO{}, err
		}
		items[i].Tags = tags
	}
	return contracts.PhotoBooksPageDTO{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func photoBookWhere(req contracts.ListPhotoBooksRequest) (string, []any) {
	var clauses []string
	var args []any
	if q := strings.TrimSpace(req.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clauses = append(clauses, `(LOWER(pb.title) LIKE ? OR LOWER(pb.source_file_name) LIKE ? OR LOWER(pb.location) LIKE ? OR EXISTS (
			SELECT 1 FROM photo_book_tags pbt WHERE pbt.photo_id = pb.id AND LOWER(pbt.tag) LIKE ?
		))`)
		args = append(args, like, like, like, like)
	}
	if tag := strings.TrimSpace(req.Tag); tag != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM photo_book_tags pbt WHERE pbt.photo_id = pb.id AND pbt.tag = ?)`)
		args = append(args, tag)
	}
	if req.Favorite != nil {
		v := 0
		if *req.Favorite {
			v = 1
		}
		clauses = append(clauses, `pb.is_favorite = ?`)
		args = append(args, v)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
