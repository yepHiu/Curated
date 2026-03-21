package assets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	logger          *zap.Logger
	cacheDir        string
	client          *http.Client
	maxConcurrent   int
	maxResponseBody int64
}

// NewService creates an asset downloader. Pass 0 for maxConcurrentDownloads to default to 3,
// and 0 for maxResponseBodyMB to default to 50 MiB per response.
func NewService(logger *zap.Logger, cacheDir string, requestTimeout time.Duration, maxConcurrentDownloads int, maxResponseBodyMB int) *Service {
	if requestTimeout <= 0 {
		requestTimeout = 30 * time.Second
	}
	if maxConcurrentDownloads <= 0 {
		maxConcurrentDownloads = 3
	}
	maxBytes := int64(maxResponseBodyMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 50 * 1024 * 1024
	}

	return &Service{
		logger:   logger,
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: requestTimeout,
		},
		maxConcurrent:   maxConcurrentDownloads,
		maxResponseBody: maxBytes,
	}
}

func (s *Service) DownloadAll(ctx context.Context, metadata scraper.Metadata) ([]DownloadedAsset, error) {
	destDir := filepath.Join(s.cacheDir, metadata.MovieID)
	return s.DownloadAllTo(ctx, metadata, destDir)
}

// DownloadAllTo downloads cover, thumb, and preview images into destDir (e.g. library番号 folder).
func (s *Service) DownloadAllTo(ctx context.Context, metadata scraper.Metadata, destDir string) ([]DownloadedAsset, error) {
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

	if len(specs) == 0 {
		return nil, nil
	}

	results := make([]DownloadedAsset, len(specs))
	errs := make([]error, len(specs))
	sem := make(chan struct{}, s.maxConcurrent)
	var wg sync.WaitGroup

	for i, spec := range specs {
		i, spec := i, spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs[i] = ctx.Err()
				return
			}
			localPath, err := s.downloadOne(ctx, destDir, metadata.Number, spec.Type, i+1, spec.SourceURL)
			if err != nil {
				errs[i] = err
				return
			}
			spec.LocalPath = localPath
			results[i] = spec
		}()
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	out := make([]DownloadedAsset, 0, len(results))
	for _, r := range results {
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) downloadOne(ctx context.Context, destDir, number, assetType string, sequence int, sourceURL string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	fileName := buildFileName(assetType, sequence, sourceURL)
	localPath := filepath.Join(destDir, fileName)

	if info, err := os.Stat(localPath); err == nil && info.Size() > 0 {
		return localPath, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "Curated-backend/0.1")

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

	limited := http.MaxBytesReader(nil, response.Body, s.maxResponseBody)
	if _, err := io.Copy(file, limited); err != nil {
		_ = os.Remove(localPath)
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return "", fmt.Errorf("asset response exceeds max size (%d bytes) for %s", s.maxResponseBody, sourceURL)
		}
		return "", err
	}

	s.logger.Info("asset downloaded",
		zap.String("destDir", destDir),
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
