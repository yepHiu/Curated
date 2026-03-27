package curatedexport

import (
	"errors"
	"strings"
)

// ErrActorContextNotInFrame is returned when actorName is set but not present in the frame's actors list.
var ErrActorContextNotInFrame = errors.New("actor not in frame")

// FilenameActor picks the actor string used in export filenames.
// If contextActor is non-empty, it must equal (after TrimSpace) one of actors; if actors is empty, returns ErrActorContextNotInFrame.
// If contextActor is empty, uses the first non-empty trimmed actor, or "unknown".
func FilenameActor(actors []string, contextActor string) (string, error) {
	ctx := strings.TrimSpace(contextActor)
	if ctx != "" {
		if len(actors) == 0 {
			return "", ErrActorContextNotInFrame
		}
		for _, a := range actors {
			if strings.TrimSpace(a) == ctx {
				return ctx, nil
			}
		}
		return "", ErrActorContextNotInFrame
	}
	for _, a := range actors {
		t := strings.TrimSpace(a)
		if t != "" {
			return t, nil
		}
	}
	return "unknown", nil
}
