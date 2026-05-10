package storage

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/storagehealth"
)

func TestLibraryPathStorageBindingUpsertAndGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "bindings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	original := storagehealth.Binding{
		LibraryPathID:      "library-1",
		RootPath:           `E:\`,
		VolumeID:           "ABCD-1234",
		VolumeLabel:        "CURATED",
		FileSystem:         "NTFS",
		DriveType:          "removable",
		IdentityConfidence: "high",
	}
	if err := store.UpsertLibraryPathStorageBinding(ctx, original); err != nil {
		t.Fatalf("UpsertLibraryPathStorageBinding original: %v", err)
	}

	got, ok, err := store.GetLibraryPathStorageBinding(ctx, "library-1")
	if err != nil {
		t.Fatalf("GetLibraryPathStorageBinding original: %v", err)
	}
	if !ok {
		t.Fatal("binding not found")
	}
	if got.VolumeID != "ABCD-1234" || got.RootPath != `E:\` {
		t.Fatalf("binding = %+v", got)
	}

	updated := original
	updated.VolumeID = "DCBA-4321"
	updated.VolumeLabel = "CURATED-NEW"
	if err := store.UpsertLibraryPathStorageBinding(ctx, updated); err != nil {
		t.Fatalf("UpsertLibraryPathStorageBinding updated: %v", err)
	}

	got, ok, err = store.GetLibraryPathStorageBinding(ctx, "library-1")
	if err != nil {
		t.Fatalf("GetLibraryPathStorageBinding updated: %v", err)
	}
	if !ok {
		t.Fatal("updated binding not found")
	}
	if got.VolumeID != "DCBA-4321" || got.VolumeLabel != "CURATED-NEW" {
		t.Fatalf("updated binding = %+v", got)
	}
}
