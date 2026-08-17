// Package diskutil provides cross-platform disk space probes shared by the
// backup maintenance tooling and the movie import prechecks.
package diskutil

// AvailableDiskBytes reports the free bytes available to the current user on
// the volume containing path. Implemented per OS in disk_space_*.go.
