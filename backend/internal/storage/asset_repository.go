package storage

import "context"

func (s *SQLiteStore) UpdateMediaAssetLocalPath(ctx context.Context, movieID, assetType, sourceURL, localPath string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE media_assets
		SET local_path = ?, last_error = '', last_fetched_at = ?, updated_at = ?
		WHERE movie_id = ? AND type = ? AND source_url = ?`,
		localPath,
		nowUTC(),
		nowUTC(),
		movieID,
		assetType,
		sourceURL,
	)
	return err
}

func (s *SQLiteStore) UpdateMediaAssetFetchState(ctx context.Context, movieID, assetType, sourceURL string, httpStatus int, lastErr string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE media_assets
		SET last_http_status = ?, last_error = ?, last_fetched_at = ?, updated_at = ?
		WHERE movie_id = ? AND type = ? AND source_url = ?`,
		httpStatus,
		lastErr,
		nowUTC(),
		nowUTC(),
		movieID,
		assetType,
		sourceURL,
	)
	return err
}
