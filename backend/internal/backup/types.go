// Package backup creates, verifies, preflights, and restores Curated backup packages.
package backup

import (
	"context"
	"errors"
	"time"

	"curated-backend/internal/storage"
)

// ErrDestinationExists prevents backup creation from overwriting a package.
var ErrDestinationExists = errors.New("backup destination already exists")

const (
	FormatName               = "curated-backup"
	FormatVersion            = 1
	ManifestPath             = "manifest.json"
	DatabaseArchivePath      = "database/curated.db"
	LibraryConfigArchivePath = "config/library-config.cfg"
)

// Scope records intentionally included and excluded data classes.
type Scope struct {
	DatabaseIncluded      bool `json:"databaseIncluded"`
	LibraryConfigIncluded bool `json:"libraryConfigIncluded"`
	UserAssetsIncluded    bool `json:"userAssetsIncluded"`
	MediaFilesIncluded    bool `json:"mediaFilesIncluded"`
}

// FileEntry authenticates one file stored in the package.
type FileEntry struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

// Manifest describes a portable Curated backup package.
type Manifest struct {
	Format           string      `json:"format"`
	FormatVersion    int         `json:"formatVersion"`
	CreatedAt        time.Time   `json:"createdAt"`
	AppVersion       string      `json:"appVersion"`
	AppChannel       string      `json:"appChannel"`
	Scope            Scope       `json:"scope"`
	SchemaMigrations []string    `json:"schemaMigrations"`
	Files            []FileEntry `json:"files"`
}

// CreateOptions configures creation of one backup package.
type CreateOptions struct {
	Store             *storage.SQLiteStore
	DestinationPath   string
	LibraryConfigPath string
	AppVersion        string
	AppChannel        string
	Now               func() time.Time
}

// Verification reports all deterministic package and SQLite checks.
type Verification struct {
	Valid             bool                    `json:"valid"`
	CheckedAt         time.Time               `json:"checkedAt"`
	Manifest          *Manifest               `json:"manifest,omitempty"`
	DatabaseIntegrity storage.IntegrityReport `json:"databaseIntegrity"`
	Errors            []string                `json:"errors"`
	Warnings          []string                `json:"warnings"`
}

// PreflightOptions configures a non-destructive restore assessment.
type PreflightOptions struct {
	BackupPath     string
	TargetDatabase string
	TargetConfig   string
	Now            func() time.Time
}

// RestorePreflight reports whether the current binary and destination can
// safely accept a verified backup.
type RestorePreflight struct {
	CanRestore            bool         `json:"canRestore"`
	CheckedAt             time.Time    `json:"checkedAt"`
	Verification          Verification `json:"verification"`
	TargetDatabase        string       `json:"targetDatabase"`
	TargetDatabaseExists  bool         `json:"targetDatabaseExists"`
	TargetConfig          string       `json:"targetConfig,omitempty"`
	TargetConfigExists    bool         `json:"targetConfigExists"`
	RequiredBytes         int64        `json:"requiredBytes"`
	AvailableBytes        uint64       `json:"availableBytes"`
	AvailableBytesKnown   bool         `json:"availableBytesKnown"`
	UnsupportedMigrations []string     `json:"unsupportedMigrations"`
	Errors                []string     `json:"errors"`
	Warnings              []string     `json:"warnings"`
}

// RestoreOptions configures an offline restore. Confirm must be true.
type RestoreOptions struct {
	PreflightOptions
	Confirm bool
}

// RestoreResult identifies retained rollback evidence after a successful restore.
type RestoreResult struct {
	RestoredAt           time.Time `json:"restoredAt"`
	TargetDatabase       string    `json:"targetDatabase"`
	DatabaseRollbackPath string    `json:"databaseRollbackPath,omitempty"`
	TargetConfig         string    `json:"targetConfig,omitempty"`
	ConfigRollbackPath   string    `json:"configRollbackPath,omitempty"`
	Manifest             Manifest  `json:"manifest"`
}

type verifier interface {
	Verify(context.Context, string) (Verification, error)
}
