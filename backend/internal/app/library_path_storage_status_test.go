package app

import (
	"context"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/storagehealth"
)

type appStorageStatusProbe struct {
	results map[string]storagehealth.ProbeResult
	calls   []string
}

func (p *appStorageStatusProbe) Probe(_ context.Context, path string) (storagehealth.ProbeResult, error) {
	p.calls = append(p.calls, path)
	return p.results[path], nil
}

func newStorageStatusTestApp(t *testing.T) (*App, *storage.SQLiteStore, *appStorageStatusProbe) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "storage-status.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	probe := &appStorageStatusProbe{results: make(map[string]storagehealth.ProbeResult)}
	return &App{
		store:         store,
		storageHealth: storagehealth.NewChecker(probe, store),
	}, store, probe
}

func TestListLibraryPathStorageStatusChecksConfiguredPaths(t *testing.T) {
	ctx := context.Background()
	a, store, probe := newStorageStatusTestApp(t)
	first, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "alpha"), "Alpha")
	if err != nil {
		t.Fatalf("AddLibraryPath first: %v", err)
	}
	second, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "beta"), "Beta")
	if err != nil {
		t.Fatalf("AddLibraryPath second: %v", err)
	}
	probe.results[first.Path] = onlineStorageProbeResult(`E:\`, "VOL-1")
	probe.results[second.Path] = onlineStorageProbeResult(`F:\`, "VOL-2")

	got, err := a.ListLibraryPathStorageStatus(ctx)
	if err != nil {
		t.Fatalf("ListLibraryPathStorageStatus: %v", err)
	}

	if len(got.Items) != 2 {
		t.Fatalf("items = %d, want 2: %+v", len(got.Items), got.Items)
	}
	for _, item := range got.Items {
		if item.Status != contracts.LibraryPathStorageStatusOnline {
			t.Fatalf("status for %s = %q, want online", item.LibraryPathID, item.Status)
		}
	}
}

func TestCheckLibraryPathStorageStatusFiltersByID(t *testing.T) {
	ctx := context.Background()
	a, store, probe := newStorageStatusTestApp(t)
	first, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "alpha"), "Alpha")
	if err != nil {
		t.Fatalf("AddLibraryPath first: %v", err)
	}
	second, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "beta"), "Beta")
	if err != nil {
		t.Fatalf("AddLibraryPath second: %v", err)
	}
	probe.results[first.Path] = onlineStorageProbeResult(`E:\`, "VOL-1")
	probe.results[second.Path] = onlineStorageProbeResult(`F:\`, "VOL-2")

	got, err := a.CheckLibraryPathStorageStatus(ctx, []string{second.ID})
	if err != nil {
		t.Fatalf("CheckLibraryPathStorageStatus: %v", err)
	}

	if len(got.Items) != 1 || got.Items[0].LibraryPathID != second.ID {
		t.Fatalf("items = %+v, want only %s", got.Items, second.ID)
	}
	if len(probe.calls) != 1 || probe.calls[0] != second.Path {
		t.Fatalf("probe calls = %+v, want only %q", probe.calls, second.Path)
	}
}

func TestRebindLibraryPathStorageUpdatesBinding(t *testing.T) {
	ctx := context.Background()
	a, store, probe := newStorageStatusTestApp(t)
	path, err := store.AddLibraryPath(ctx, filepath.Join(t.TempDir(), "movies"), "Movies")
	if err != nil {
		t.Fatalf("AddLibraryPath: %v", err)
	}
	if err := store.UpsertLibraryPathStorageBinding(ctx, storagehealth.Binding{
		LibraryPathID:      path.ID,
		RootPath:           `E:\`,
		VolumeID:           "OLD-1234",
		IdentityConfidence: "high",
	}); err != nil {
		t.Fatalf("UpsertLibraryPathStorageBinding old: %v", err)
	}
	probe.results[path.Path] = onlineStorageProbeResult(`E:\`, "NEW-5678")

	got, err := a.RebindLibraryPathStorage(ctx, path.ID)
	if err != nil {
		t.Fatalf("RebindLibraryPathStorage: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusOnline {
		t.Fatalf("status = %q, want online", got.Status)
	}
	binding, ok, err := store.GetLibraryPathStorageBinding(ctx, path.ID)
	if err != nil {
		t.Fatalf("GetLibraryPathStorageBinding: %v", err)
	}
	if !ok || binding.VolumeID != "NEW-5678" {
		t.Fatalf("binding after rebind = %+v, ok=%v", binding, ok)
	}
}

func onlineStorageProbeResult(root string, volumeID string) storagehealth.ProbeResult {
	return storagehealth.ProbeResult{
		RootPath:           root,
		RootAvailable:      true,
		PathExists:         true,
		PathIsDir:          true,
		PathReadable:       true,
		VolumeID:           volumeID,
		VolumeLabel:        "CURATED",
		FileSystem:         "NTFS",
		DriveType:          "removable",
		IdentityConfidence: "high",
	}
}
