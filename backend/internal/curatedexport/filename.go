package curatedexport

import (
	"fmt"
	"strings"
)

const maxFilenamePart = 80

var filenameBadChars = strings.NewReplacer(
	"/", "_", "\\", "_", "?", "_", "%", "_", "*", "_", ":", "_", "|", "_", `"`, "_", "<", "_", ">", "_",
)

// SanitizePathSegment mirrors frontend formatFrameFilename safety for one path component.
func SanitizePathSegment(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	t = filenameBadChars.Replace(t)
	if len(t) > maxFilenamePart {
		t = t[:maxFilenamePart]
	}
	return t
}

func exportCuratedFilename(actorForName, code string, positionSec float64, frameID string, used map[string]struct{}, ext string) string {
	safeActor := SanitizePathSegment(actorForName)
	if safeActor == "" {
		safeActor = "unknown"
	}
	safeCode := SanitizePathSegment(code)
	if safeCode == "" {
		safeCode = "frame"
	}
	sec := int(positionSec)
	if positionSec < 0 {
		sec = 0
	}
	base := fmt.Sprintf("curated-%s-%s-%ds", safeActor, safeCode, sec)
	name := base + ext
	if used != nil {
		if _, ok := used[name]; ok {
			suffix := frameID
			if len(suffix) > 8 {
				suffix = suffix[:8]
			}
			if suffix == "" {
				suffix = "x"
			}
			name = base + "-" + suffix + ext
		}
		used[name] = struct{}{}
	}
	return name
}

// ExportWebPFilename builds curated-{actor}-{code}-{sec}s.webp with optional id suffix for ZIP uniqueness.
func ExportWebPFilename(actorForName, code string, positionSec float64, frameID string, used map[string]struct{}) string {
	return exportCuratedFilename(actorForName, code, positionSec, frameID, used, ".webp")
}

// ExportPNGFilename builds curated-{actor}-{code}-{sec}s.png with optional id suffix for ZIP uniqueness.
func ExportPNGFilename(actorForName, code string, positionSec float64, frameID string, used map[string]struct{}) string {
	return exportCuratedFilename(actorForName, code, positionSec, frameID, used, ".png")
}

// ExportJPGFilename builds curated-{actor}-{code}-{sec}s.jpg with optional id suffix for ZIP uniqueness.
func ExportJPGFilename(actorForName, code string, positionSec float64, frameID string, used map[string]struct{}) string {
	return exportCuratedFilename(actorForName, code, positionSec, frameID, used, ".jpg")
}
