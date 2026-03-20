package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/scraper"
)

type DownloadedAsset struct {
	Type      string
	SourceURL string
	LocalPath string
}

type Service struct {
	logger   *zap.Logger
	cacheDir string
	client   *http.Client
}

func NewService(logger *zap.Logger, cacheDir string, requestTimeout time.Duration) *Service {
	if requestTimeout <= 0 {
		requestTimeout = 30 * time.Second
	}

	return &Service{
		logger:   logger,
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (s *Service) DownloadAll(ctx context.Context, metadata scraper.Metadata) ([]DownloadedAsset, error) {
	specs := make([]DownloadedAsset, 0, 2+len(metadata.PreviewImages))
	if metadata.CoverURL != "" {
		specs = append(specs, DownloadedAsset{Type: "cover", SourceURL: metadata.CoverURL})
	}
	if metadata.ThumbURL != "" {
		specs = append(specs, DownloadedAsset{Type: "thumb", SourceURL: metadata.ThumbURL})
	}
	for _, previewURL := range metadata.PreviewImages {
		if previewURL == "" {
			continue
		}
		specs = append(specs, DownloadedAsset{Type: "preview_image", SourceURL: previewURL})
	}

	results := make([]DownloadedAsset, 0, len(specs))
	for index, spec := range specs {
		localPath, err := s.downloadOne(ctx, metadata.MovieID, metadata.Number, spec.Type, index+1, spec.SourceURL)
		if err != nil {
			return results, err
		}
		spec.LocalPath = localPath
		results = append(results, spec)
	}

	return results, nil
}

func (s *Service) downloadOne(ctx context.Context, movieID, number, assetType string, sequence int, sourceURL string) (string, error) {
	if err := os.MkdirAll(filepath.Join(s.cacheDir, movieID), 0o755); err != nil {
		return "", err
	}

	fileName := buildFileName(assetType, sequence, sourceURL)
	localPath := filepath.Join(s.cacheDir, movieID, fileName)

	if info, err := os.Stat(localPath); err == nil && info.Size() > 0 {
		return localPath, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "jav-shadcn-backend/0.1")

	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected asset response status %d for %s", response.StatusCode, sourceURL)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		return "", err
	}

	s.logger.Info("asset downloaded",
		zap.String("movieId", movieID),
		zap.String("number", number),
		zap.String("type", assetType),
		zap.String("localPath", localPath),
	)
	return localPath, nil
}

func buildFileName(assetType string, sequence int, sourceURL string) string {
	ext := inferExt(sourceURL)
	switch assetType {
	case "cover":
		return "cover" + ext
	case "thumb":
		return "thumb" + ext
	default:
		return fmt.Sprintf("preview-%02d%s", sequence, ext)
	}
}

func inferExt(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		ext := strings.ToLower(path.Ext(parsed.Path))
		if ext != "" && len(ext) <= 5 {
			return ext
		}
	}
	return ".jpg"
}
