package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func BenchmarkCuratedLargeLibrary(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s, err := NewSQLiteStore(filepath.Join(b.TempDir(), "bench.db"))
			if err != nil {
				b.Fatal(err)
			}
			defer s.Close()
			ctx := context.Background()
			if err := s.Migrate(ctx); err != nil {
				b.Fatal(err)
			}
			_, err = s.db.Exec(`INSERT INTO movies(id,title,code,studio,summary,added_at,location,resolution,year) VALUES ('m','m','BENCH','','','','bench','',0)`)
			if err != nil {
				b.Fatal(err)
			}
			tx, err := s.db.Begin()
			if err != nil {
				b.Fatal(err)
			}
			stmt, err := tx.Prepare(`INSERT INTO curated_frames(id,movie_id,title,code,actors_json,position_sec,captured_at,tags_json,image_blob,thumb_blob) VALUES (?,'m','frame','BENCH','[]',?,'2026-09-06T00:00:00Z','[]',zeroblob(4096),zeroblob(1024))`)
			if err != nil {
				b.Fatal(err)
			}
			for i := 0; i < count; i++ {
				if _, err := stmt.Exec(fmt.Sprintf("%08d", i), i); err != nil {
					b.Fatal(err)
				}
			}
			stmt.Close()
			if err := tx.Commit(); err != nil {
				b.Fatal(err)
			}
			cursor := encodeCuratedFrameCursor(CuratedFrameMeta{CapturedAt: "2026-09-06T00:00:00Z", ID: fmt.Sprintf("%08d", count/2)})
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := s.QueryCuratedFrames(ctx, CuratedFrameQuery{Limit: 60, Cursor: cursor, SkipTotal: true}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
