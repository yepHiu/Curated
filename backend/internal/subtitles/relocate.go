package subtitles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RelocateAssets copies only registered, hash-verified outputs after a video moves.
// Originals remain available if copying or indexing fails; unrelated files are untouched.
func (s *Service) RelocateAssets(ctx context.Context, oldPath, newPath string) error {
	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records, e := s.opt.Store.ListSubtitleRecords(ctx, "")
	if e != nil {
		return e
	}
	for _, r := range records {
		if r.SourcePath != oldPath || r.Job.State != "completed" {
			continue
		}
		oldDir := filepath.Join(filepath.Dir(oldPath), "sub")
		newDir := filepath.Join(filepath.Dir(newPath), "sub")
		if oldDir != newDir {
			if e = os.MkdirAll(newDir, 0755); e != nil {
				return e
			}
			for i, a := range r.Job.Assets {
				if filepath.Base(a.Name) != a.Name {
					return fmt.Errorf("invalid subtitle filename")
				}
				raw, err := os.ReadFile(filepath.Join(oldDir, a.Name))
				if err != nil {
					return err
				}
				sum := sha256.Sum256(raw)
				if hex.EncodeToString(sum[:]) != a.Hash {
					return fmt.Errorf("subtitle modified; automatic relocation skipped")
				}
				name := a.Name
				dest := filepath.Join(newDir, name)
				if existing, err := os.ReadFile(dest); err == nil {
					digest := sha256.Sum256(existing)
					if hex.EncodeToString(digest[:]) == a.Hash {
						continue
					}
					ext := filepath.Ext(name)
					name = strings.TrimSuffix(name, ext) + "." + r.Job.ID[:8] + ext
					dest = filepath.Join(newDir, name)
				}
				f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				_, err = f.Write(raw)
				if err == nil {
					err = f.Sync()
				}
				closeErr := f.Close()
				if err != nil {
					return err
				}
				if closeErr != nil {
					return closeErr
				}
				r.Job.Assets[i].Name = name
			}
		}
		r.SourcePath = newPath
		r.Job.FileName = filepath.Base(newPath)
		if e = s.opt.Store.SaveSubtitleRecord(ctx, r); e != nil {
			return e
		}
	}
	return nil
}
