package subtitles

import (
	"context"
	"strings"
	"time"

	"curated-backend/internal/storage"
)

// The initial restart implementation created a replacement job with a restart-
// request ID. Link an unambiguous prior failure so it remains truthful history
// without requiring a second intervention. Never rewrite an active task's ID.
func (s *Service) reconcileLegacyRestarts(ctx context.Context) error {
	records, err := s.opt.Store.ListSubtitleRecords(ctx, "")
	if err != nil {
		return err
	}
	for _, next := range records {
		if !strings.HasPrefix(next.RequestID, "restart-") {
			continue
		}
		created, err := time.Parse(time.RFC3339Nano, next.Job.CreatedAt)
		if err != nil {
			continue
		}
		var matches []storage.SubtitleRecord
		for _, old := range records {
			if old.Job.ID == next.Job.ID || old.Job.SupersededBy != "" ||
				(old.Job.State != "failed" && old.Job.State != "blocked") ||
				old.Job.MovieID != next.Job.MovieID || old.Job.FileID != next.Job.FileID ||
				old.AudioTrack != next.AudioTrack || old.SourcePath != next.SourcePath || old.Revision != next.Revision {
				continue
			}
			updated, err := time.Parse(time.RFC3339Nano, old.Job.UpdatedAt)
			if err == nil && updated.Before(created) {
				matches = append(matches, old)
			}
		}
		// Older records have no explicit parent ID. Ambiguity must not silently
		// dismiss unrelated failures, including other audio tracks/generations.
		if len(matches) == 1 {
			old := matches[0]
			old.Job.SupersededBy = next.Job.ID
			if err := s.opt.Store.SaveSubtitleRecord(ctx, old); err != nil {
				return err
			}
			for i := range records {
				if records[i].Job.ID == old.Job.ID {
					records[i] = old
				}
			}
		}
	}
	return nil
}
