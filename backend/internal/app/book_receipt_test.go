package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

// The closures accept a store so the same checks also run after reopening SQLite.
func bookReceiptFixture(t *testing.T, ctx context.Context, store *storage.SQLiteStore, tool string) (
	map[string]any, func(*storage.SQLiteStore, string) error, func(*storage.SQLiteStore) (string, error),
) {
	t.Helper()
	if strings.Contains(tool, "_comic_") {
		book, err := store.UpsertComicBook(ctx, storage.ComicBookUpsert{
			Location: filepath.Join(t.TempDir(), "comic.cbz"), Title: "Original comic", PageCount: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if tool == "save_comic_comment" {
			return map[string]any{"comicId": book.ID, "body": "AI value"},
				func(s *storage.SQLiteStore, value string) error {
					_, err := s.UpsertComicComment(ctx, book.ID, value)
					return err
				},
				func(s *storage.SQLiteStore) (string, error) {
					note, err := s.GetComicComment(ctx, book.ID)
					return note.Body, err
				}
		}
		return map[string]any{"comicId": book.ID, "title": "AI value"},
			func(s *storage.SQLiteStore, value string) error {
				_, err := s.PatchComicBook(ctx, book.ID, contracts.PatchComicBookRequest{Title: &value})
				return err
			},
			func(s *storage.SQLiteStore) (string, error) {
				book, err := s.GetComicBookDetail(ctx, book.ID)
				return book.Title, err
			}
	}
	if strings.Contains(tool, "_photo_") {
		book, err := store.UpsertPhotoBook(ctx, storage.PhotoBookUpsert{
			Location: filepath.Join(t.TempDir(), "photo.zip"), Title: "Original photo", PageCount: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if tool == "save_photo_comment" {
			return map[string]any{"photoId": book.ID, "body": "AI value"},
				func(s *storage.SQLiteStore, value string) error {
					_, err := s.UpsertPhotoComment(ctx, book.ID, value)
					return err
				},
				func(s *storage.SQLiteStore) (string, error) {
					note, err := s.GetPhotoComment(ctx, book.ID)
					return note.Body, err
				}
		}
		return map[string]any{"photoId": book.ID, "title": "AI value"},
			func(s *storage.SQLiteStore, value string) error {
				_, err := s.PatchPhotoBook(ctx, book.ID, contracts.PatchPhotoBookRequest{Title: &value})
				return err
			},
			func(s *storage.SQLiteStore) (string, error) {
				book, err := s.GetPhotoBookDetail(ctx, book.ID)
				return book.Title, err
			}
	}
	return nil, nil, nil
}
