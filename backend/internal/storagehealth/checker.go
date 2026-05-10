package storagehealth

import (
	"context"
	"strings"
	"time"

	"curated-backend/internal/contracts"
)

// Probe reads platform-specific storage state for a configured library path.
type Probe interface {
	Probe(ctx context.Context, path string) (ProbeResult, error)
}

// BindingStore persists expected storage identities for configured library paths.
type BindingStore interface {
	UpsertLibraryPathStorageBinding(ctx context.Context, binding Binding) error
}

type bindingReader interface {
	BindingStore
	lookupBinding(libraryPathID string) (Binding, bool)
}

// Checker classifies a configured library path's current backing-storage status.
type Checker struct {
	probe Probe
	store BindingStore
}

// NewChecker returns a storage-health checker using the provided platform probe and binding store.
func NewChecker(probe Probe, store BindingStore) *Checker {
	return &Checker{probe: probe, store: store}
}

// CheckPath probes and classifies one configured library path.
func (c *Checker) CheckPath(ctx context.Context, path contracts.LibraryPathDTO) (contracts.LibraryPathStorageStatusDTO, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	out := contracts.LibraryPathStorageStatusDTO{
		LibraryPathID: path.ID,
		Path:          path.Path,
		Title:         path.Title,
		Status:        contracts.LibraryPathStorageStatusUnknown,
		CheckedAt:     now,
		CanRescan:     false,
		CanImport:     false,
	}
	if c == nil || c.probe == nil {
		out.Message = "Storage status checker is not available."
		return out, nil
	}

	probed, err := c.probe.Probe(ctx, path.Path)
	if err != nil {
		return out, err
	}
	out.RootPath = probed.RootPath
	out.DriveType = probed.DriveType
	out.VolumeLabel = probed.VolumeLabel
	out.FileSystem = probed.FileSystem
	out.IdentityConfidence = probed.IdentityConfidence
	out.CurrentVolumeID = probed.VolumeID

	binding, hasBinding := c.lookupBinding(path.ID)
	if hasBinding {
		out.ExpectedVolumeID = binding.VolumeID
	}

	out.Status = classify(probed, binding, hasBinding)
	out.Message = statusMessage(out.Status, probed.ErrorMessage)
	out.CanRescan = out.Status == contracts.LibraryPathStorageStatusOnline
	out.CanImport = out.Status == contracts.LibraryPathStorageStatusOnline

	if c.store != nil {
		bindingToSave := bindingFromProbe(path.ID, probed)
		if hasBinding {
			bindingToSave = binding
		}
		if out.Status == contracts.LibraryPathStorageStatusOnline {
			bindingToSave = bindingFromProbe(path.ID, probed)
		}
		if bindingToSave.LibraryPathID == "" {
			bindingToSave.LibraryPathID = path.ID
		}
		if bindingToSave.RootPath == "" {
			bindingToSave.RootPath = probed.RootPath
		}
		if err := c.store.UpsertLibraryPathStorageBinding(ctx, bindingToSave); err != nil {
			return out, err
		}
	}

	return out, nil
}

// RebindPath replaces the expected backing-volume identity with the current probe result.
func (c *Checker) RebindPath(ctx context.Context, path contracts.LibraryPathDTO) (contracts.LibraryPathStorageStatusDTO, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	out := contracts.LibraryPathStorageStatusDTO{
		LibraryPathID: path.ID,
		Path:          path.Path,
		Title:         path.Title,
		Status:        contracts.LibraryPathStorageStatusUnknown,
		CheckedAt:     now,
		CanRescan:     false,
		CanImport:     false,
	}
	if c == nil || c.probe == nil {
		out.Message = "Storage status checker is not available."
		return out, nil
	}

	probed, err := c.probe.Probe(ctx, path.Path)
	if err != nil {
		return out, err
	}
	currentBinding := bindingFromProbe(path.ID, probed)
	out.RootPath = probed.RootPath
	out.DriveType = probed.DriveType
	out.VolumeLabel = probed.VolumeLabel
	out.FileSystem = probed.FileSystem
	out.IdentityConfidence = probed.IdentityConfidence
	out.ExpectedVolumeID = currentBinding.VolumeID
	out.CurrentVolumeID = probed.VolumeID
	out.Status = classify(probed, currentBinding, currentBinding.VolumeID != "")
	out.Message = statusMessage(out.Status, probed.ErrorMessage)
	out.CanRescan = out.Status == contracts.LibraryPathStorageStatusOnline
	out.CanImport = out.Status == contracts.LibraryPathStorageStatusOnline

	if out.Status == contracts.LibraryPathStorageStatusOnline && c.store != nil {
		if err := c.store.UpsertLibraryPathStorageBinding(ctx, currentBinding); err != nil {
			return out, err
		}
	}

	return out, nil
}

func (c *Checker) lookupBinding(libraryPathID string) (Binding, bool) {
	if c == nil || c.store == nil {
		return Binding{}, false
	}
	if r, ok := c.store.(bindingReader); ok {
		return r.lookupBinding(libraryPathID)
	}
	type exportedLookup interface {
		GetLibraryPathStorageBinding(context.Context, string) (Binding, bool, error)
	}
	if r, ok := c.store.(exportedLookup); ok {
		b, found, err := r.GetLibraryPathStorageBinding(context.Background(), libraryPathID)
		if err == nil {
			return b, found
		}
	}
	return Binding{}, false
}

func classify(probed ProbeResult, binding Binding, hasBinding bool) contracts.LibraryPathStorageStatus {
	if !probed.RootAvailable {
		return contracts.LibraryPathStorageStatusOffline
	}
	if hasBinding && binding.VolumeID != "" && probed.VolumeID != "" && !strings.EqualFold(binding.VolumeID, probed.VolumeID) {
		return contracts.LibraryPathStorageStatusVolumeMismatch
	}
	if !probed.PathExists || !probed.PathIsDir {
		return contracts.LibraryPathStorageStatusPathMissing
	}
	if probed.PermissionDenied || !probed.PathReadable {
		return contracts.LibraryPathStorageStatusPermissionDenied
	}
	return contracts.LibraryPathStorageStatusOnline
}

func bindingFromProbe(libraryPathID string, probed ProbeResult) Binding {
	return Binding{
		LibraryPathID:      libraryPathID,
		RootPath:           probed.RootPath,
		VolumeID:           probed.VolumeID,
		VolumeLabel:        probed.VolumeLabel,
		FileSystem:         probed.FileSystem,
		DriveType:          probed.DriveType,
		IdentityConfidence: probed.IdentityConfidence,
	}
}

func statusMessage(status contracts.LibraryPathStorageStatus, detail string) string {
	switch status {
	case contracts.LibraryPathStorageStatusOnline:
		return "Storage path is online."
	case contracts.LibraryPathStorageStatusOffline:
		if strings.TrimSpace(detail) != "" {
			return detail
		}
		return "The storage device may be offline or disconnected."
	case contracts.LibraryPathStorageStatusVolumeMismatch:
		return "The current volume does not match the storage device previously bound to this path."
	case contracts.LibraryPathStorageStatusPathMissing:
		return "The storage device is online, but the library directory is missing."
	case contracts.LibraryPathStorageStatusPermissionDenied:
		return "The library directory exists, but Curated does not have permission to read it."
	default:
		if strings.TrimSpace(detail) != "" {
			return detail
		}
		return "Curated could not confirm this storage path."
	}
}
