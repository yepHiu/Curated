package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestComicLibraryPathsRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newComicRepositoryTestStore(t, root)

	mainRoot := filepath.Join(root, "main")
	path, err := store.AddComicLibraryPath(ctx, mainRoot, "Main Comics")
	if err != nil {
		t.Fatalf("add comic path: %v", err)
	}
	if path.ID == "" || path.Path != mainRoot || path.Title != "Main Comics" || !path.FirstLibraryScanPending {
		t.Fatalf("unexpected path dto: %+v", path)
	}

	list, err := store.ListComicLibraryPaths(ctx)
	if err != nil {
		t.Fatalf("list comic paths: %v", err)
	}
	if len(list) != 1 || list[0].ID != path.ID {
		t.Fatalf("list = %+v, want one path %q", list, path.ID)
	}

	updated, err := store.UpdateComicLibraryPathTitle(ctx, path.ID, "Shelf")
	if err != nil {
		t.Fatalf("update comic path title: %v", err)
	}
	if updated.Title != "Shelf" {
		t.Fatalf("updated title = %q, want Shelf", updated.Title)
	}

	_, err = store.AddComicLibraryPath(ctx, mainRoot, "Duplicate")
	if !errors.Is(err, ErrComicLibraryPathDuplicate) {
		t.Fatalf("duplicate add error = %v, want ErrComicLibraryPathDuplicate", err)
	}

	_, err = store.AddComicLibraryPath(ctx, "relative/comics", "Relative")
	if !errors.Is(err, ErrComicLibraryPathNotAbsolute) {
		t.Fatalf("relative add error = %v, want ErrComicLibraryPathNotAbsolute", err)
	}

	if err := store.DeleteComicLibraryPath(ctx, path.ID); err != nil {
		t.Fatalf("delete comic path: %v", err)
	}
	if err := store.DeleteComicLibraryPath(ctx, path.ID); !errors.Is(err, ErrComicLibraryPathNotFound) {
		t.Fatalf("delete missing error = %v, want ErrComicLibraryPathNotFound", err)
	}
}

func TestListComicLibraryPathStrings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := newComicRepositoryTestStore(t, root)

	first := filepath.Join(root, "a")
	second := filepath.Join(root, "b")
	if _, err := store.AddComicLibraryPath(ctx, second, "Second"); err != nil {
		t.Fatalf("add second comic path: %v", err)
	}
	if _, err := store.AddComicLibraryPath(ctx, first, "First"); err != nil {
		t.Fatalf("add first comic path: %v", err)
	}

	got, err := store.ListComicLibraryPathStrings(ctx)
	if err != nil {
		t.Fatalf("list comic path strings: %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("path strings = %#v, want [%q %q]", got, first, second)
	}
}
