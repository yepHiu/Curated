package library

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"jav-shadcn/backend/internal/scraper"
)

// WriteMovieNFO writes a Kodi-oriented movie.nfo into dir (番号目录，与视频同级).
func WriteMovieNFO(dir string, m scraper.Metadata) error {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	type actorEl struct {
		Name string `xml:"name"`
		Role string `xml:"role,omitempty"`
	}
	type doc struct {
		XMLName       xml.Name  `xml:"movie"`
		Title         string    `xml:"title"`
		OriginalTitle string    `xml:"originaltitle,omitempty"`
		SortTitle     string    `xml:"sorttitle,omitempty"`
		Plot          string    `xml:"plot,omitempty"`
		Outline       string    `xml:"outline,omitempty"`
		Director      string    `xml:"director,omitempty"`
		Studio        string    `xml:"studio,omitempty"`
		Year          string    `xml:"year,omitempty"`
		Genre         []string  `xml:"genre,omitempty"`
		Actor         []actorEl `xml:"actor,omitempty"`
		Runtime       string    `xml:"runtime,omitempty"`
	}

	year := ""
	if len(m.ReleaseDate) >= 4 {
		year = m.ReleaseDate[:4]
	}
	actors := make([]actorEl, 0, len(m.Actors))
	for _, a := range m.Actors {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		actors = append(actors, actorEl{Name: a})
	}

	runtime := ""
	if m.RuntimeMinutes > 0 {
		runtime = strconv.Itoa(m.RuntimeMinutes)
	}

	d := doc{
		Title:         m.Title,
		OriginalTitle: m.Number,
		SortTitle:     m.Number,
		Plot:          m.Summary,
		Outline:       trimOutline(m.Summary),
		Director:      m.Director,
		Studio:        firstNonEmpty(m.Studio, m.Label),
		Year:          year,
		Genre:         append([]string(nil), m.Tags...),
		Actor:         actors,
		Runtime:       runtime,
	}

	path := filepath.Join(dir, "movie.nfo")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(d); err != nil {
		return err
	}
	_, err = f.WriteString("\n")
	return err
}

func firstNonEmpty(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}

func trimOutline(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "…"
}
