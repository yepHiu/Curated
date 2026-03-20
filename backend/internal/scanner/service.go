package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.uber.org/zap"

	"jav-shadcn/backend/internal/contracts"
)

var errNoPathsConfigured = errors.New("no scan paths configured")

type Service struct {
	logger     *zap.Logger
	extensions map[string]struct{}
}

type Hooks struct {
	OnProgress     func(processed, total int, message string)
	OnFileDetected func(result contracts.ScanFileResultDTO) error
}

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

func (s *Service) Scan(ctx context.Context, taskID string, paths []string, hooks Hooks) (contracts.ScanSummaryDTO, error) {
	if len(paths) == 0 {
		return contracts.ScanSummaryDTO{}, errNoPathsConfigured
	}

	discoveredFiles, err := s.findVideoFiles(paths)
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

func (s *Service) findVideoFiles(paths []string) ([]string, error) {
	files := make([]string, 0)
	seen := make(map[string]struct{})

	for _, rootPath := range paths {
		if strings.TrimSpace(rootPath) == "" {
			continue
		}

		walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
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

func (s *Service) isVideoFile(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	_, ok := s.extensions[ext]
	return ok
}
