package storagehealth

import (
	"context"
	"testing"

	"curated-backend/internal/contracts"
)

type fakeBindingStore struct {
	bindings map[string]Binding
	upserts  []Binding
}

func (s *fakeBindingStore) UpsertLibraryPathStorageBinding(_ context.Context, b Binding) error {
	if s.bindings == nil {
		s.bindings = make(map[string]Binding)
	}
	s.bindings[b.LibraryPathID] = b
	s.upserts = append(s.upserts, b)
	return nil
}

func (s *fakeBindingStore) lookupBinding(libraryPathID string) (Binding, bool) {
	if s.bindings == nil {
		return Binding{}, false
	}
	b, ok := s.bindings[libraryPathID]
	return b, ok
}

func TestCheckerClassifiesOfflineRoot(t *testing.T) {
	t.Parallel()

	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:      `E:\`,
				RootAvailable: false,
				DriveType:     "removable",
				ErrorMessage:  "drive is not ready",
			},
		},
	}, &fakeBindingStore{})

	got, err := checker.CheckPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("CheckPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusOffline {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusOffline)
	}
	if got.CanRescan || got.CanImport {
		t.Fatalf("offline path should block scan/import: %+v", got)
	}
}

func TestCheckerClassifiesVolumeMismatch(t *testing.T) {
	t.Parallel()

	store := &fakeBindingStore{
		bindings: map[string]Binding{
			"library-1": {
				LibraryPathID:      "library-1",
				RootPath:           `E:\`,
				VolumeID:           "OLD-1234",
				IdentityConfidence: "high",
			},
		},
	}
	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:           `E:\`,
				RootAvailable:      true,
				PathExists:         true,
				PathIsDir:          true,
				PathReadable:       true,
				VolumeID:           "NEW-5678",
				IdentityConfidence: "high",
				VolumeLabel:        "OTHER",
				FileSystem:         "NTFS",
				DriveType:          "removable",
			},
		},
	}, store)

	got, err := checker.CheckPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("CheckPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusVolumeMismatch {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusVolumeMismatch)
	}
	if got.ExpectedVolumeID != "OLD-1234" || got.CurrentVolumeID != "NEW-5678" {
		t.Fatalf("volume ids not reported: %+v", got)
	}
	if got.CanRescan || got.CanImport {
		t.Fatalf("mismatched volume should block scan/import: %+v", got)
	}
}

func TestCheckerBindsOnlinePathWhenMissingBinding(t *testing.T) {
	t.Parallel()

	store := &fakeBindingStore{}
	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:           `E:\`,
				RootAvailable:      true,
				PathExists:         true,
				PathIsDir:          true,
				PathReadable:       true,
				VolumeID:           "ABCD-1234",
				VolumeLabel:        "CURATED",
				FileSystem:         "NTFS",
				DriveType:          "removable",
				IdentityConfidence: "high",
			},
		},
	}, store)

	got, err := checker.CheckPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("CheckPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusOnline {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusOnline)
	}
	if !got.CanRescan || !got.CanImport {
		t.Fatalf("online path should allow scan/import: %+v", got)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(store.upserts))
	}
	if store.upserts[0].VolumeID != "ABCD-1234" {
		t.Fatalf("bound volume id = %q", store.upserts[0].VolumeID)
	}
}

func TestCheckerClassifiesPathMissingAfterVolumeMatch(t *testing.T) {
	t.Parallel()

	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:           `E:\`,
				RootAvailable:      true,
				PathExists:         false,
				VolumeID:           "ABCD-1234",
				IdentityConfidence: "high",
			},
		},
	}, &fakeBindingStore{
		bindings: map[string]Binding{
			"library-1": {
				LibraryPathID:      "library-1",
				RootPath:           `E:\`,
				VolumeID:           "ABCD-1234",
				IdentityConfidence: "high",
			},
		},
	})

	got, err := checker.CheckPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("CheckPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusPathMissing {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusPathMissing)
	}
}

func TestCheckerClassifiesPermissionDeniedAfterVolumeMatch(t *testing.T) {
	t.Parallel()

	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:           `E:\`,
				RootAvailable:      true,
				PathExists:         true,
				PathIsDir:          true,
				PathReadable:       false,
				PermissionDenied:   true,
				VolumeID:           "ABCD-1234",
				IdentityConfidence: "high",
			},
		},
	}, &fakeBindingStore{
		bindings: map[string]Binding{
			"library-1": {
				LibraryPathID:      "library-1",
				RootPath:           `E:\`,
				VolumeID:           "ABCD-1234",
				IdentityConfidence: "high",
			},
		},
	})

	got, err := checker.CheckPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("CheckPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusPermissionDenied {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusPermissionDenied)
	}
}

func TestCheckerRebindsMismatchedVolume(t *testing.T) {
	t.Parallel()

	store := &fakeBindingStore{
		bindings: map[string]Binding{
			"library-1": {
				LibraryPathID:      "library-1",
				RootPath:           `E:\`,
				VolumeID:           "OLD-1234",
				IdentityConfidence: "high",
			},
		},
	}
	checker := NewChecker(fakeProbe{
		results: map[string]ProbeResult{
			`E:\Movies`: {
				RootPath:           `E:\`,
				RootAvailable:      true,
				PathExists:         true,
				PathIsDir:          true,
				PathReadable:       true,
				VolumeID:           "NEW-5678",
				VolumeLabel:        "CURATED",
				FileSystem:         "NTFS",
				DriveType:          "removable",
				IdentityConfidence: "high",
			},
		},
	}, store)

	got, err := checker.RebindPath(context.Background(), contracts.LibraryPathDTO{
		ID:    "library-1",
		Path:  `E:\Movies`,
		Title: "External HDD",
	})
	if err != nil {
		t.Fatalf("RebindPath returned error: %v", err)
	}

	if got.Status != contracts.LibraryPathStorageStatusOnline {
		t.Fatalf("status = %q, want %q", got.Status, contracts.LibraryPathStorageStatusOnline)
	}
	if got.ExpectedVolumeID != "NEW-5678" || got.CurrentVolumeID != "NEW-5678" {
		t.Fatalf("volume ids after rebind = expected %q current %q", got.ExpectedVolumeID, got.CurrentVolumeID)
	}
	if len(store.upserts) != 1 || store.upserts[0].VolumeID != "NEW-5678" {
		t.Fatalf("binding was not overwritten with current volume: %+v", store.upserts)
	}
}

type fakeProbe struct {
	results map[string]ProbeResult
}

func (p fakeProbe) Probe(_ context.Context, path string) (ProbeResult, error) {
	return p.results[path], nil
}
