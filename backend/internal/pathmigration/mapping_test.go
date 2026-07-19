package pathmigration

import "testing"

func TestWindowsMappingIsCaseInsensitiveAndSegmentAware(t *testing.T) {
	mapping, err := NewMapping(`d:\Media`, `E:/Library`)
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	mapped, matched, err := mapping.Map(`D:/MEDIA/Actor/Movie.mp4`)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if !matched || mapped != `E:\Library\Actor\Movie.mp4` {
		t.Fatalf("Map = %q, %v", mapped, matched)
	}
	if _, matched, err := mapping.Map(`D:\Media2\Movie.mp4`); err != nil || matched {
		t.Fatalf("boundary Map matched=%v err=%v", matched, err)
	}
}

func TestMappingSupportsWindowsToUnix(t *testing.T) {
	mapping, err := NewMapping(`C:\Curated\Media`, `/srv/curated/media`)
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	mapped, matched, err := mapping.Map(`c:\curated\media\Studio\movie.mkv`)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if !matched || mapped != "/srv/curated/media/Studio/movie.mkv" {
		t.Fatalf("Map = %q, %v", mapped, matched)
	}
}

func TestUnixMappingRemainsCaseSensitive(t *testing.T) {
	mapping, err := NewMapping("/srv/Media", "/mnt/media")
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	if _, matched, err := mapping.Map("/srv/media/movie.mp4"); err != nil || matched {
		t.Fatalf("case-sensitive Map matched=%v err=%v", matched, err)
	}
}

func TestMappingRejectsRelativeEquivalentAndParentPaths(t *testing.T) {
	for _, testCase := range []struct {
		from string
		to   string
	}{
		{from: "relative", to: "/target"},
		{from: "/source/../escape", to: "/target"},
		{from: `D:\Media`, to: `d:/media`},
		{from: `D:\Media`, to: `d:/media/migrated`},
	} {
		if _, err := NewMapping(testCase.from, testCase.to); err == nil {
			t.Fatalf("NewMapping(%q, %q) unexpectedly succeeded", testCase.from, testCase.to)
		}
	}
}

func TestMappingPreservesSignificantUnixWhitespace(t *testing.T) {
	mapping, err := NewMapping(" /srv/media ", "/mnt/media")
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	mapped, matched, err := mapping.Map("/srv/media/movie ")
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if !matched || mapped != "/mnt/media/movie " {
		t.Fatalf("Map = %q, %v", mapped, matched)
	}
}

func TestUNCMappingPreservesShareBoundary(t *testing.T) {
	mapping, err := NewMapping(`\\server\media`, `\\archive\library`)
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	mapped, matched, err := mapping.Map(`\\SERVER\MEDIA\actor\movie.mkv`)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if !matched || mapped != `\\archive\library\actor\movie.mkv` {
		t.Fatalf("Map = %q, %v", mapped, matched)
	}
}
