package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// subtitleCleanupPathsTx returns only registered outputs whose content is unchanged.
func subtitleCleanupPathsTx(ctx context.Context, tx *sql.Tx, movieID string) ([]string, error) {
	rows, e := tx.QueryContext(ctx, `SELECT j.payload,f.location FROM subtitle_jobs j JOIN movie_files f ON f.id=j.file_id WHERE j.movie_id=?`, movieID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var raw, source string
		if e = rows.Scan(&raw, &source); e != nil {
			return nil, e
		}
		var r SubtitleRecord
		if e = json.Unmarshal([]byte(raw), &r); e != nil {
			return nil, e
		}
		dir := filepath.Join(filepath.Dir(source), "sub")
		parent, e := os.Lstat(dir)
		if e != nil || !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
			continue
		}
		for _, a := range r.Job.Assets {
			if filepath.Base(a.Name) != a.Name {
				continue
			}
			path := filepath.Join(dir, a.Name)
			info, e := os.Lstat(path)
			if e != nil || !info.Mode().IsRegular() || info.Size() != a.Size || info.Size() > 16*1024*1024 {
				continue
			}
			body, e := os.ReadFile(path)
			if e != nil {
				continue
			}
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) == a.Hash {
				result = append(result, path)
			}
		}
	}
	return result, rows.Err()
}
