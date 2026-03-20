package storage

import "context"

func (s *SQLiteStore) UpdateMediaAssetLocalPath(ctx context.Context, movieID, assetType, sourceURL, localPath string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE media_assets
		SET local_path = ?, updated_at = ?
		WHERE movie_id = ? AND type = ? AND source_url = ?`,
		localPath,
		nowUTC(),
		movieID,
		assetType,
		sourceURL,
	)
	return err
}
