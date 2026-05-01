// Package scanner walks configured library directories for video files, extracts video IDs from filenames, and reports results via Hooks.
package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

// errNoPathsConfigured is returned when Scan receives an empty path list.
var errNoPathsConfigured = errors.New("no scan paths configured")

// Service scans library directories for video files, extracts video IDs, and drives downstream processing via Hooks.
type Service struct {
	logger     *zap.Logger
	extensions map[string]struct{} // allowed video extensions (lowercase, no dot)
}

// Hooks lets callers inject progress reporting and per-file handling (e.g. synchronous ingest).
// When any callback returns an error, Scan stops early and propagates the error.
type Hooks struct {
	OnProgress     func(processed, total int, message string)
	OnFileDetected func(result contracts.ScanFileResultDTO) error
}

// NewService creates a scanner that recognizes common container extensions (mp4, mkv, avi, mov, ts).
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
		extensions: map[string]struct{}{
			"mp4": {},
			"mkv": {},
			"avi": {},
			"mov": {},
			"ts":  {},
		},
	}
}

// Scan discovers video files under the given roots, populates ScanFileResultDTO per file, and aggregates a ScanSummaryDTO.
// Context cancellation returns the partial summary alongside ctx.Err(). An empty path list returns errNoPathsConfigured.
func (s *Service) Scan(ctx context.Context, taskID string, paths []string, hooks Hooks) (contracts.ScanSummaryDTO, error) {
	if len(paths) == 0 {
		return contracts.ScanSummaryDTO{}, errNoPathsConfigured
	}

	discoveredFiles, err := s.findVideoFiles(ctx, paths)
	if err != nil {
		return contracts.ScanSummaryDTO{}, err
	}

	summary := contracts.ScanSummaryDTO{
		TaskID:          taskID,
		Paths:           slices.Clone(paths),
		FilesDiscovered: len(discoveredFiles),
		Results:         make([]contracts.ScanFileResultDTO, 0, len(discoveredFiles)),
	}

	if hooks.OnProgress != nil {
		hooks.OnProgress(0, len(discoveredFiles), "Discovered candidate video files")
	}

	for index, filePath := range discoveredFiles {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		default:
		}

		result := contracts.ScanFileResultDTO{
			TaskID:   taskID,
			Path:     filePath,
			FileName: filepath.Base(filePath),
		}

		// Extract the video ID from filename; unrecognized files are marked skipped but still reported via OnFileDetected.
		number := ExtractNumber(result.FileName)
		if number == "" {
			result.Status = "skipped"
			result.Reason = "number_not_recognized"
			summary.FilesSkipped++
			summary.Results = append(summary.Results, result)
		} else {
			result.Status = "recognized"
			result.Number = number
			summary.RecognizedNumber++
			summary.Results = append(summary.Results, result)
		}

		if hooks.OnFileDetected != nil {
			if err := hooks.OnFileDetected(result); err != nil {
				return summary, err
			}
		}

		if hooks.OnProgress != nil {
			message := "Parsed video filename and prepared scan candidate"
			if result.Status == "skipped" {
				message = "Skipped file without recognized number"
			}
			hooks.OnProgress(index+1, len(discoveredFiles), message)
		}
	}

	return summary, nil
}

// findVideoFiles walks each root with filepath.Walk, collects regular files with allowed extensions, deduplicates by cleaned path, and returns them in lexicographic order.
func (s *Service) findVideoFiles(ctx context.Context, paths []string) ([]string, error) {
	files := make([]string, 0)
	seen := make(map[string]struct{})
	var walkCount int

	for _, rootPath := range paths {
		if strings.TrimSpace(rootPath) == "" {
			continue
		}

		walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			walkCount++
			if walkCount%100 == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
			if info.IsDir() {
				return nil
			}
			if !s.isVideoFile(path) {
				return nil
			}

			cleanPath := filepath.Clean(path)
			if _, exists := seen[cleanPath]; exists {
				return nil
			}
			seen[cleanPath] = struct{}{}
			files = append(files, cleanPath)
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	slices.Sort(files)
	s.logger.Info("scanner discovered files", zap.Int("count", len(files)))
	return files, nil
}

// isVideoFile checks whether path has a configured video extension (lowercase, no dot).
func (s *Service) isVideoFile(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	_, ok := s.extensions[ext]
	return ok
}
