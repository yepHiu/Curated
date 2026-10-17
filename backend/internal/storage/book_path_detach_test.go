package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
)

func TestBookPathRemovalDetachesAndRestores(t *testing.T) {
	for _, kind := range []string{"comic", "photo"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			s := newComicRepositoryTestStore(t, root)
			add := func(path string) string {
				t.Helper()
				if kind == "comic" {
					p, e := s.AddComicLibraryPath(ctx, path, "")
					if e != nil {
						t.Fatal(e)
					}
					return p.ID
				}
				p, e := s.AddPhotoLibraryPath(ctx, path, "")
				if e != nil {
					t.Fatal(e)
				}
				return p.ID
			}
			remove := func(id string) error {
				if kind == "comic" {
					return s.DeleteComicLibraryPath(ctx, id)
				}
				return s.DeletePhotoLibraryPath(ctx, id)
			}
			upsert := func(pathID, location string) (string, error) {
				if kind == "comic" {
					b, e := s.UpsertComicBook(ctx, ComicBookUpsert{LibraryPathID: pathID, Location: location, Title: "source", PageCount: 1})
					return b.ID, e
				}
				b, e := s.UpsertPhotoBook(ctx, PhotoBookUpsert{LibraryPathID: pathID, Location: location, Title: "source", PageCount: 1})
				return b.ID, e
			}
			total := func() int {
				t.Helper()
				if kind == "comic" {
					p, e := s.ListComicBooks(ctx, contracts.ListComicBooksRequest{})
					if e != nil {
						t.Fatal(e)
					}
					return p.Total
				}
				p, e := s.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{})
				if e != nil {
					t.Fatal(e)
				}
				return p.Total
			}
			main := filepath.Join(root, "books")
			nested := filepath.Join(main, "nested")
			sibling := filepath.Join(root, "books-other")
			mainID, childID, siblingID := add(main), add(nested), add(sibling)
			if err := os.MkdirAll(main, 0755); err != nil {
				t.Fatal(err)
			}
			location := filepath.Join(main, "one.cbz")
			if err := os.WriteFile(location, []byte("source unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
			book, err := upsert(mainID, location)
			if err != nil {
				t.Fatal(err)
			}
			child, err := upsert(mainID, filepath.Join(nested, "two.cbz"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = upsert(siblingID, filepath.Join(sibling, "three.cbz")); err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.ExecContext(ctx, `UPDATE `+kind+`_books SET user_title='saved title',is_favorite=1,metadata_json='{"cleanTitle":"saved"}' WHERE id=?`, book); err != nil {
				t.Fatal(err)
			}
			if err = remove(mainID); err != nil {
				t.Fatal(err)
			}
			if total() != 2 {
				t.Fatalf("wrong remaining count: %d", total())
			}
			var binding string
			if err = s.db.QueryRowContext(ctx, `SELECT library_path_id FROM `+kind+`_books WHERE id=?`, child).Scan(&binding); err != nil || binding != childID {
				t.Fatalf("nested binding %s %v", binding, err)
			}
			if kind == "comic" {
				_, err = s.GetComicBookDetail(ctx, book)
				if !errors.Is(err, ErrComicBookNotFound) {
					t.Fatalf("detached detail: %v", err)
				}
			} else {
				_, err = s.GetPhotoBookDetail(ctx, book)
				if !errors.Is(err, ErrPhotoBookNotFound) {
					t.Fatalf("detached detail: %v", err)
				}
			}
			if err = remove(childID); err != nil {
				t.Fatal(err)
			}
			if total() != 1 {
				t.Fatal("last nested root did not detach book")
			}
			// An in-flight scan using a removed root cannot reintroduce a card.
			if _, err = upsert(mainID, location); err == nil {
				t.Fatal("accepted removed path")
			}
			mainID = add(main)
			restored, err := upsert(mainID, location)
			if err != nil || restored != book {
				t.Fatalf("restore changed identity: %s %v", restored, err)
			}
			var title, metadata string
			var favorite int
			if err = s.db.QueryRowContext(ctx, `SELECT user_title,metadata_json,is_favorite FROM `+kind+`_books WHERE id=?`, book).Scan(&title, &metadata, &favorite); err != nil || title != "saved title" || metadata != `{"cleanTitle":"saved"}` || favorite != 1 {
				t.Fatalf("lost saved metadata: %s %s %d %v", title, metadata, favorite, err)
			}
			content, err := os.ReadFile(location)
			if err != nil || string(content) != "source unchanged" {
				t.Fatal("source file changed")
			}
			if total() != 2 {
				t.Fatal("rescan did not restore book")
			}
		})
	}
}

func TestDetachedBookMigrationRepairsLegacyOrphans(t *testing.T) {
	ctx := context.Background()
	s := newComicRepositoryTestStore(t, t.TempDir())
	for _, kind := range []string{"comic", "photo"} {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE `+kind+`_books DROP COLUMN detached`); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO `+kind+`_books (id,title,source_file_name,location,added_at,updated_at) VALUES ('legacy','saved','legacy.zip',?,'2026-10-17','2026-10-17')`, filepath.Join(t.TempDir(), "legacy.zip")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE name='0070_detached_books.sql'`); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"comic", "photo"} {
		var detached int
		if err := s.db.QueryRowContext(ctx, `SELECT detached FROM `+kind+`_books WHERE id='legacy'`).Scan(&detached); err != nil || detached != 1 {
			t.Fatalf("legacy %s: %d %v", kind, detached, err)
		}
	}
}
