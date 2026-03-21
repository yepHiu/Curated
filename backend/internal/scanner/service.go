// Package scanner 在配置的库目录下递归遍历视频文件，按扩展名筛选后从文件名解析番号，
// 并通过 Hooks 将每条结果交给上层（例如入库、任务进度上报）。
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

// errNoPathsConfigured 在 Scan 收到空路径列表时返回。
var errNoPathsConfigured = errors.New("no scan paths configured")

// Service 执行库目录扫描：发现候选视频文件、解析番号，并通过 Hooks 回调驱动后续流程。
type Service struct {
	logger     *zap.Logger
	extensions map[string]struct{} // 允许的视频扩展名（不含点，小写）
}

// Hooks 由调用方注入，用于进度通知与每条扫描结果的处理（如同步入库）。
// 任一回调返回错误时 Scan 会提前结束并向上传递该错误。
type Hooks struct {
	OnProgress     func(processed, total int, message string)
	OnFileDetected func(result contracts.ScanFileResultDTO) error
}

// NewService 构造扫描器，默认识别常见容器扩展名：mp4、mkv、avi、mov、ts。
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

// Scan 在给定根路径下查找视频文件，逐条填充 ScanFileResultDTO（含 Path、FileName、Number、Status 等），
// 并汇总为 ScanSummaryDTO。支持 ctx 取消：取消时返回已累积的 summary 与 ctx.Err()。
// paths 为空时返回 errNoPathsConfigured。
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

		// 从文件名解析番号；无法识别则记为 skipped，仍回调 OnFileDetected 供上层决定是否忽略。
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

// findVideoFiles 对每个根路径做 filepath.Walk，收集扩展名在白名单内的普通文件路径，
// 按 Clean 后的路径去重，最终按字典序排序后返回。
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

// isVideoFile 根据扩展名（小写、无点）判断是否为配置中的视频类型。
func (s *Service) isVideoFile(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	_, ok := s.extensions[ext]
	return ok
}
