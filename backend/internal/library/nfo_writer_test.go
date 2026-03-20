package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jav-shadcn/backend/internal/scraper"
)

func TestWriteMovieNFO(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := scraper.Metadata{
		MovieID:        "mid-1",
		Number:         "ABC-123",
		Title:          "Test Title",
		Summary:        "Plot line.",
		Director:       "Dir",
		Studio:         "Studio X",
		Actors:         []string{"Actor A", "Actor B"},
		Tags:           []string{"Tag1"},
		RuntimeMinutes: 90,
		ReleaseDate:    "2024-01-15",
	}

	if err := WriteMovieNFO(dir, m); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "movie.nfo"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "<movie>") || !strings.Contains(s, "Test Title") || !strings.Contains(s, "ABC-123") {
		t.Fatalf("unexpected nfo content: %s", s)
	}
}
