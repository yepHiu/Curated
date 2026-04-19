// Package version exposes build stamp (date + time), channel (dev vs release), and helpers for health/logs.
//
// Build alignment (same as internal/config data paths):
//
//	go run ./cmd/curated              → channel dev
//	go build -tags release ./cmd/curated → channel release
//
// Stamp (date-time) sources, in order:
//  1. Link-time -X curated-backend/internal/version.BuildStamp=20060102.150405 (CI / 精确编译时刻)
//  2. Else runtime/debug BuildInfo vcs.time（Git 提交时间，非严格「编译瞬间」）
//  3. Else "unknown"
//
// Example release build:
//
//	go build -tags release -ldflags "-X curated-backend/internal/version.BuildStamp=20260328.143052" ./cmd/curated
package version

import (
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// BuildStamp is an optional link-time override, compact form YYYYMMDD.HHMMSS (UTC recommended).
var BuildStamp = ""

// InstallerVersion is an optional link-time override for the packaged release version
// (for example, the version allocated from scripts/release/version.json during publish).
var InstallerVersion = ""

const devFallbackPackageVersion = "0.0.0"

var (
	resolvedFallback string
	resolveOnce      sync.Once
)

// Stamp returns the date-time build identifier without channel suffix.
func Stamp() string {
	if s := strings.TrimSpace(BuildStamp); s != "" {
		return s
	}
	resolveOnce.Do(func() {
		resolvedFallback = stampFromBuildInfo()
	})
	return resolvedFallback
}

func stampFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	var vcsTime, vcsRevision string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.time":
			vcsTime = s.Value
		case "vcs.revision":
			vcsRevision = s.Value
		}
	}
	if vcsTime != "" {
		if t, err := time.Parse(time.RFC3339, vcsTime); err == nil {
			return t.UTC().Format("20060102.150405")
		}
	}
	if vcsRevision != "" {
		short := vcsRevision
		if len(short) > 12 {
			short = short[:12]
		}
		return "git." + short
	}
	return "unknown"
}

// Display returns stamp and channel in one string, e.g. "20260328.143052-dev" or "...-release".
func Display() string {
	return Stamp() + "-" + Channel
}

// PackageVersion returns the packaged release version when embedded into the binary.
func PackageVersion() string {
	if version := strings.TrimSpace(InstallerVersion); version != "" {
		return version
	}
	if Channel != "release" {
		return devFallbackPackageVersion
	}
	return ""
}
