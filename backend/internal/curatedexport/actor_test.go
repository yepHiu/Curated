package curatedexport

import (
	"errors"
	"testing"
)

func TestFilenameActor(t *testing.T) {
	a, err := FilenameActor([]string{"  A ", "B"}, "")
	if err != nil || a != "A" {
		t.Fatalf("got %q %v", a, err)
	}
	a, err = FilenameActor([]string{"A"}, "A")
	if err != nil || a != "A" {
		t.Fatalf("ctx got %q %v", a, err)
	}
	_, err = FilenameActor([]string{"A"}, "B")
	if !errors.Is(err, ErrActorContextNotInFrame) {
		t.Fatalf("want ErrActorContextNotInFrame, got %v", err)
	}
	_, err = FilenameActor(nil, "X")
	if !errors.Is(err, ErrActorContextNotInFrame) {
		t.Fatalf("empty actors: %v", err)
	}
	a, err = FilenameActor(nil, "")
	if err != nil || a != "unknown" {
		t.Fatalf("unknown got %q %v", a, err)
	}
}
