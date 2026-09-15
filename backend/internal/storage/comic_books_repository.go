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
	"unicode/utf8"

	"curated-backend/internal/contracts"
)

const comicDisplayTitleSQL = `COALESCE(NULLIF(TRIM(cb.user_title), ''), cb.title)`

var ErrComicBookNotFound = errors.New("comic book not found")

type ComicBookUpsert struct {
	LibraryPathID  string
	Location       string
	SourceFileName string
	Title          string
	FileSize       int64
	FileModifiedAt string
	PageCount      int
}

func (s *SQLiteStore) UpsertComicBook(ctx context.Context, in ComicBookUpsert) (contracts.ComicBookDetailDTO, error) {
	location := filepath.Clean(strings.TrimSpace(in.Location))
	if location == "" || location == "." {
		return contracts.ComicBookDetailDTO{}, fmt.Errorf("location is required")
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
	id := "comic-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ts := nowUTC()
	addedAt := time.Now().UTC().Format("2006-01-02")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comic_books (
			id, library_path_id, title, source_file_name, location, file_size, file_modified_at,
			page_count, cover_page_index, read_status, added_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 'unread', ?, ?)
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
		return contracts.ComicBookDetailDTO{}, err
	}
	var actualID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM comic_books WHERE location = ?`, location).Scan(&actualID); err != nil {
		return contracts.ComicBookDetailDTO{}, err
	}
	return s.GetComicBookDetail(ctx, actualID)
}

func (s *SQLiteStore) GetComicBookDetail(ctx context.Context, id string) (contracts.ComicBookDetailDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
	}
	var item contracts.ComicBookListItemDTO
	var favoriteInt int
	var rating sql.NullFloat64
	err := s.db.QueryRowContext(ctx,
		`SELECT cb.id, `+comicDisplayTitleSQL+`, cb.source_file_name, cb.location, cb.page_count, cb.is_favorite,
		        cb.user_rating, cb.read_status, cb.added_at, cb.updated_at, cb.last_read_at, cb.completed_at,
		        COALESCE(crp.current_page_index, 0)
		   FROM comic_books cb
		   LEFT JOIN comic_reading_progress crp ON crp.comic_id = cb.id
		  WHERE cb.id = ?`,
		id,
	).Scan(
		&item.ID,
		&item.Title,
		&item.SourceFileName,
		&item.Location,
		&item.PageCount,
		&favoriteInt,
		&rating,
		&item.ReadStatus,
		&item.AddedAt,
		&item.UpdatedAt,
		&item.LastReadAt,
		&item.CompletedAt,
		&item.CurrentPageIndex,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
		}
		return contracts.ComicBookDetailDTO{}, err
	}
	item.IsFavorite = favoriteInt != 0
	if rating.Valid {
		item.Rating = &rating.Float64
	}
	tags, err := s.listComicTags(ctx, id)
	if err != nil {
		return contracts.ComicBookDetailDTO{}, err
	}
	pages, err := s.ListComicPages(ctx, id)
	if err != nil {
		return contracts.ComicBookDetailDTO{}, err
	}
	item.Tags = tags
	return contracts.ComicBookDetailDTO{ComicBookListItemDTO: item, Pages: pages}, nil
}

func (s *SQLiteStore) GetComicBookByLocation(ctx context.Context, location string) (contracts.ComicBookDetailDTO, error) {
	location = filepath.Clean(strings.TrimSpace(location))
	if location == "" || location == "." {
		return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
	}
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM comic_books WHERE location = ?`, location).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
		}
		return contracts.ComicBookDetailDTO{}, err
	}
	return s.GetComicBookDetail(ctx, id)
}

func (s *SQLiteStore) ListComicBooks(ctx context.Context, req contracts.ListComicBooksRequest) (contracts.ComicBooksPageDTO, error) {
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	where, args := comicBookWhere(req)
	var total int
	countQuery := `SELECT COUNT(DISTINCT cb.id) FROM comic_books cb ` + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return contracts.ComicBooksPageDTO{}, err
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT cb.id, `+comicDisplayTitleSQL+`, cb.source_file_name, cb.location, cb.page_count, cb.is_favorite,
		        cb.user_rating, cb.read_status, cb.added_at, cb.updated_at, cb.last_read_at, cb.completed_at,
		        COALESCE(crp.current_page_index, 0)
		   FROM comic_books cb
		   LEFT JOIN comic_reading_progress crp ON crp.comic_id = cb.id `+
			where+
			` GROUP BY cb.id
		      ORDER BY cb.added_at DESC, cb.id ASC
		      LIMIT ? OFFSET ?`,
		listArgs...,
	)
	if err != nil {
		return contracts.ComicBooksPageDTO{}, err
	}

	items := make([]contracts.ComicBookListItemDTO, 0)
	for rows.Next() {
		var item contracts.ComicBookListItemDTO
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
			&item.ReadStatus,
			&item.AddedAt,
			&item.UpdatedAt,
			&item.LastReadAt,
			&item.CompletedAt,
			&item.CurrentPageIndex,
		); err != nil {
			return contracts.ComicBooksPageDTO{}, err
		}
		item.IsFavorite = favoriteInt != 0
		if rating.Valid {
			item.Rating = &rating.Float64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return contracts.ComicBooksPageDTO{}, err
	}
	if err := rows.Close(); err != nil {
		return contracts.ComicBooksPageDTO{}, err
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	tagsByID, err := s.listComicTagsByIDs(ctx, ids)
	if err != nil {
		return contracts.ComicBooksPageDTO{}, err
	}
	for i := range items {
		items[i].Tags = tagsByID[items[i].ID]
	}
	return contracts.ComicBooksPageDTO{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func comicBookWhere(req contracts.ListComicBooksRequest) (string, []any) {
	var clauses []string
	var args []any
	if q := strings.TrimSpace(req.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clauses = append(clauses, `(LOWER(`+comicDisplayTitleSQL+`) LIKE ? OR LOWER(cb.title) LIKE ? OR LOWER(cb.source_file_name) LIKE ? OR LOWER(cb.location) LIKE ? OR EXISTS (
			SELECT 1 FROM comic_book_tags cbt WHERE cbt.comic_id = cb.id AND LOWER(cbt.tag) LIKE ?
		))`)
		args = append(args, like, like, like, like, like)
	}
	if tag := strings.TrimSpace(req.Tag); tag != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM comic_book_tags cbt WHERE cbt.comic_id = cb.id AND cbt.tag = ?)`)
		args = append(args, tag)
	}
	if req.Favorite != nil {
		v := 0
		if *req.Favorite {
			v = 1
		}
		clauses = append(clauses, `cb.is_favorite = ?`)
		args = append(args, v)
	}
	if status := strings.TrimSpace(req.ReadStatus); status != "" {
		clauses = append(clauses, `cb.read_status = ?`)
		args = append(args, status)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (s *SQLiteStore) PatchComicBook(ctx context.Context, comicID string, patch contracts.PatchComicBookRequest) (contracts.ComicBookDetailDTO, error) {
	comicID = strings.TrimSpace(comicID)
	if comicID == "" {
		return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.ComicBookDetailDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if patch.ExpectedTitle != nil {
		var current string
		err := tx.QueryRowContext(ctx, `SELECT `+comicDisplayTitleSQL+` FROM comic_books cb WHERE cb.id = ?`, comicID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
		}
		if err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
		if current != *patch.ExpectedTitle {
			return contracts.ComicBookDetailDTO{}, ErrAIWriteConflict
		}
	}

	if patch.Tags != nil {
		normalized, err := NormalizeUserTagsForPatch(*patch.Tags)
		if err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
		if err := replaceComicTagsTx(ctx, tx, comicID, normalized); err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
	}

	var sets []string
	var args []any
	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" {
			return contracts.ComicBookDetailDTO{}, fmt.Errorf("title is required")
		}
		if utf8.RuneCountInString(title) > contracts.MaxBookTitleRunes {
			return contracts.ComicBookDetailDTO{}, fmt.Errorf("title too long")
		}
		sets = append(sets, "user_title = ?")
		args = append(args, title)
	}
	if patch.Favorite != nil {
		v := 0
		if *patch.Favorite {
			v = 1
		}
		sets = append(sets, "is_favorite = ?")
		args = append(args, v)
	}
	if patch.RatingSet {
		if patch.RatingClear || patch.Rating == nil {
			sets = append(sets, "user_rating = NULL")
		} else {
			if *patch.Rating < 0 || *patch.Rating > 5 {
				return contracts.ComicBookDetailDTO{}, fmt.Errorf("comic rating must be between 0 and 5")
			}
			sets = append(sets, "user_rating = ?")
			args = append(args, *patch.Rating)
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, nowUTC(), comicID)
		res, err := tx.ExecContext(ctx, `UPDATE comic_books SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
		if err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
		if n == 0 {
			return contracts.ComicBookDetailDTO{}, ErrComicBookNotFound
		}
	}
	if len(sets) == 0 && patch.Tags == nil {
		if _, err := tx.ExecContext(ctx, `SELECT id FROM comic_books WHERE id = ?`, comicID); err != nil {
			return contracts.ComicBookDetailDTO{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return contracts.ComicBookDetailDTO{}, err
	}
	return s.GetComicBookDetail(ctx, comicID)
}

func (s *SQLiteStore) DeleteComicBookIndex(ctx context.Context, comicID string) error {
	comicID = strings.TrimSpace(comicID)
	if comicID == "" {
		return ErrComicBookNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`DELETE FROM comic_pages WHERE comic_id = ?`,
		`DELETE FROM comic_book_tags WHERE comic_id = ?`,
		`DELETE FROM comic_reading_progress WHERE comic_id = ?`,
		`DELETE FROM comic_reading_preferences WHERE comic_id = ?`,
		`DELETE FROM comic_book_comments WHERE comic_id = ?`,
		`DELETE FROM comic_cache_entries WHERE comic_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, comicID); err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM comic_books WHERE id = ?`, comicID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrComicBookNotFound
	}
	return tx.Commit()
}

func nullEmptyString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}
