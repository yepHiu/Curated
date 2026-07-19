package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const (
	libraryHealthDefaultFindingLimit = 1000
	libraryHealthMaxFindingLimit     = 5000
)

var libraryHealthUploadDirPattern = regexp.MustCompile(`^upload_[0-9a-f]{16}$`)

type libraryHealthCollector struct {
	limit    int
	findings []contracts.LibraryHealthFindingDTO
	summary  contracts.LibraryHealthSummaryDTO
}

func newLibraryHealthCollector(limit int) *libraryHealthCollector {
	return &libraryHealthCollector{
		limit: limit,
		summary: contracts.LibraryHealthSummaryDTO{
			CategoryCounts: make(map[string]int),
		},
	}
}

func (c *libraryHealthCollector) add(finding contracts.LibraryHealthFindingDTO) {
	finding.ID = stableLibraryHealthFindingID(finding.Category, finding.EntityType, finding.EntityID, finding.Path)
	c.summary.TotalFindings++
	c.summary.CategoryCounts[finding.Category]++
	switch finding.Severity {
	case "critical":
		c.summary.CriticalFindings++
	case "warning":
		c.summary.WarningFindings++
	default:
		c.summary.InfoFindings++
	}
	if len(c.findings) < c.limit {
		c.findings = append(c.findings, finding)
	}
}

func stableLibraryHealthFindingID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}
	return "health_" + hex.EncodeToString(h.Sum(nil)[:12])
}

func (h *Handler) handleScanLibraryHealth(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.libraryPathStorageStatus == nil {
		writeAppError(w, http.StatusServiceUnavailable, contracts.ErrorCodeInternal, "library health service is unavailable")
		return
	}
	limit := libraryHealthDefaultFindingLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("findingLimit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > libraryHealthMaxFindingLimit {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "findingLimit must be between 1 and 5000")
			return
		}
		limit = parsed
	}

	report, err := h.scanLibraryHealth(r.Context(), limit)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("scan library health failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to scan library health")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) scanLibraryHealth(ctx context.Context, limit int) (contracts.LibraryHealthReportDTO, error) {
	snapshot, err := h.store.InspectLibraryHealth(ctx)
	if err != nil {
		return contracts.LibraryHealthReportDTO{}, fmt.Errorf("inspect database: %w", err)
	}
	storageList, err := h.libraryPathStorageStatus.CheckLibraryPathStorageStatus(ctx, nil)
	if err != nil {
		return contracts.LibraryHealthReportDTO{}, fmt.Errorf("check storage status: %w", err)
	}
	sort.Slice(storageList.Items, func(i, j int) bool {
		return storageList.Items[i].Path < storageList.Items[j].Path
	})

	collector := newLibraryHealthCollector(limit)
	quickOK := len(snapshot.QuickCheckMessages) == 1 && strings.EqualFold(strings.TrimSpace(snapshot.QuickCheckMessages[0]), "ok")
	if !quickOK {
		if len(snapshot.QuickCheckMessages) == 0 {
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:      "database_integrity",
				Severity:      "critical",
				EntityType:    "database",
				EntityID:      "missing-result",
				Label:         "SQLite quick_check",
				Message:       "SQLite quick_check returned no result",
				RepairActions: []string{"export_diagnostics"},
			})
		}
		for index, message := range snapshot.QuickCheckMessages {
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:      "database_integrity",
				Severity:      "critical",
				EntityType:    "database",
				EntityID:      strconv.Itoa(index),
				Label:         "SQLite quick_check",
				Message:       message,
				RepairActions: []string{"export_diagnostics"},
			})
		}
	}
	for _, violation := range snapshot.ForeignKeyViolations {
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:   "foreign_key_violation",
			Severity:   "critical",
			EntityType: "database_row",
			EntityID:   violation.TableName + ":" + violation.RowID,
			Label:      violation.TableName,
			Message:    fmt.Sprintf("row %s references missing parent table %s", violation.RowID, violation.ParentTable),
			Details: map[string]any{
				"parentTable":  violation.ParentTable,
				"foreignKeyId": violation.ForeignKeyID,
			},
			RepairActions: []string{"export_diagnostics"},
		})
	}

	for _, status := range storageList.Items {
		if status.Status == contracts.LibraryPathStorageStatusOnline {
			continue
		}
		actions := []string{"recheck_storage", "export_diagnostics"}
		if status.Status == contracts.LibraryPathStorageStatusVolumeMismatch {
			actions = append(actions, "rebind_storage")
		}
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:      "storage_unavailable",
			Severity:      "warning",
			EntityType:    "library_path",
			EntityID:      status.LibraryPathID,
			Label:         status.Title,
			Path:          status.Path,
			Message:       status.Message,
			Details:       map[string]any{"status": status.Status},
			RepairActions: actions,
		})
	}

	movieByID := make(map[string]storage.LibraryHealthMovieRecord, len(snapshot.Movies))
	assetTypesByMovie := make(map[string]map[string]bool)
	for _, movie := range snapshot.Movies {
		movieByID[movie.ID] = movie
	}
	for _, attempt := range snapshot.MetadataAttempts {
		movie, exists := movieByID[attempt.MovieID]
		if !exists {
			continue
		}
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:   "metadata_failed",
			Severity:   "warning",
			EntityType: "movie",
			EntityID:   movie.ID,
			Label:      firstNonEmpty(movie.Code, movie.Title),
			Path:       movie.Location,
			Message:    firstNonEmpty(attempt.ErrorMessage, "metadata scrape failed"),
			Details: map[string]any{
				"taskId":        attempt.TaskID,
				"errorCode":     attempt.ErrorCode,
				"errorCategory": attempt.ErrorCategory,
				"provider":      attempt.Provider,
				"finishedAt":    attempt.FinishedAt,
			},
			RepairActions: []string{"rescrape_metadata", "export_diagnostics"},
		})
	}
	for _, asset := range snapshot.Assets {
		if assetTypesByMovie[asset.MovieID] == nil {
			assetTypesByMovie[asset.MovieID] = make(map[string]bool)
		}
		if strings.TrimSpace(asset.LocalPath) != "" {
			assetTypesByMovie[asset.MovieID][strings.ToLower(strings.TrimSpace(asset.Type))] = true
		}
	}

	normalizedCodes := make(map[string][]storage.LibraryHealthMovieRecord)
	normalizedPaths := make(map[string][]storage.LibraryHealthMovieRecord)
	for _, movie := range snapshot.Movies {
		if rootStatus, matched := libraryHealthStatusForPath(movie.Location, storageList.Items); matched && rootStatus.Status != contracts.LibraryPathStorageStatusOnline {
			collector.summary.SkippedOfflineFiles++
		} else {
			addMovieSourceHealthFinding(collector, movie)
		}

		codeKey := strings.ToUpper(strings.TrimSpace(movie.Code))
		if codeKey != "" {
			normalizedCodes[codeKey] = append(normalizedCodes[codeKey], movie)
		}
		pathKey := normalizeLibraryHealthPath(movie.Location)
		if pathKey != "" {
			normalizedPaths[pathKey] = append(normalizedPaths[pathKey], movie)
		}

		if isMovieMetadataMissing(movie) {
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:      "metadata_missing",
				Severity:      "warning",
				EntityType:    "movie",
				EntityID:      movie.ID,
				Label:         movie.Code,
				Path:          movie.Location,
				Message:       "movie still has placeholder metadata",
				RepairActions: []string{"rescrape_metadata", "export_diagnostics"},
			})
		}
		if strings.TrimSpace(movie.CoverURL) == "" && !hasMoviePosterAsset(assetTypesByMovie[movie.ID]) {
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:      "movie_poster_missing",
				Severity:      "info",
				EntityType:    "movie",
				EntityID:      movie.ID,
				Label:         movie.Code,
				Message:       "movie has no remote or registered local poster",
				RepairActions: []string{"rescrape_metadata"},
			})
		}
	}

	addDuplicateMovieFindings(collector, "duplicate_code", normalizedCodes, "multiple active movies normalize to the same catalog code")
	addDuplicateMovieFindings(collector, "duplicate_source", normalizedPaths, "multiple active movies normalize to the same source path")

	for _, asset := range snapshot.Assets {
		path := strings.TrimSpace(asset.LocalPath)
		if path != "" {
			if rootStatus, matched := libraryHealthStatusForPath(path, storageList.Items); matched && rootStatus.Status != contracts.LibraryPathStorageStatusOnline {
				collector.summary.SkippedOfflineFiles++
			} else if err := validateReadableRegularFile(path, false); err != nil {
				movie := movieByID[asset.MovieID]
				collector.add(contracts.LibraryHealthFindingDTO{
					Category:      "asset_missing",
					Severity:      "warning",
					EntityType:    "media_asset",
					EntityID:      asset.ID,
					Label:         firstNonEmpty(movie.Code, asset.Type),
					Path:          path,
					Message:       err.Error(),
					Details:       map[string]any{"movieId": asset.MovieID, "assetType": asset.Type},
					RepairActions: []string{"rescrape_metadata", "export_diagnostics"},
				})
			}
		}
		if strings.TrimSpace(asset.LastError) != "" {
			movie := movieByID[asset.MovieID]
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:   "asset_download_failed",
				Severity:   "warning",
				EntityType: "media_asset",
				EntityID:   asset.ID,
				Label:      firstNonEmpty(movie.Code, asset.Type),
				Path:       path,
				Message:    asset.LastError,
				Details: map[string]any{
					"movieId":        asset.MovieID,
					"assetType":      asset.Type,
					"lastHttpStatus": asset.LastHTTPStatus,
				},
				RepairActions: []string{"rescrape_metadata", "export_diagnostics"},
			})
		}
	}

	for _, actor := range snapshot.Actors {
		localPath := strings.TrimSpace(actor.AvatarLocalPath)
		switch {
		case localPath != "":
			if rootStatus, matched := libraryHealthStatusForPath(localPath, storageList.Items); matched && rootStatus.Status != contracts.LibraryPathStorageStatusOnline {
				collector.summary.SkippedOfflineFiles++
				continue
			}
			if err := validateReadableRegularFile(localPath, false); err != nil {
				collector.add(contracts.LibraryHealthFindingDTO{
					Category:      "actor_avatar_missing",
					Severity:      "info",
					EntityType:    "actor",
					EntityID:      strconv.FormatInt(actor.ID, 10),
					Label:         actor.Name,
					Path:          localPath,
					Message:       err.Error(),
					RepairActions: []string{"rescrape_actor"},
				})
			}
		case strings.TrimSpace(actor.AvatarURL) == "":
			collector.add(contracts.LibraryHealthFindingDTO{
				Category:      "actor_avatar_missing",
				Severity:      "info",
				EntityType:    "actor",
				EntityID:      strconv.FormatInt(actor.ID, 10),
				Label:         actor.Name,
				Message:       "actor has no remote or local avatar",
				RepairActions: []string{"rescrape_actor"},
			})
		}
	}

	for _, orphan := range snapshot.Orphans {
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:      "orphan_user_state",
			Severity:      "warning",
			EntityType:    orphan.TableName,
			EntityID:      orphan.Key,
			Label:         orphan.TableName,
			Message:       "user state references a missing movie",
			RepairActions: []string{"cleanup_orphan_state", "export_diagnostics"},
		})
	}

	for _, status := range storageList.Items {
		if status.Status != contracts.LibraryPathStorageStatusOnline {
			continue
		}
		addImportStagingFindings(collector, status, snapshot.UploadSessionIDs)
	}

	status := "healthy"
	if collector.summary.CriticalFindings > 0 {
		status = "critical"
	} else if collector.summary.WarningFindings > 0 || collector.summary.InfoFindings > 0 {
		status = "attention"
	}
	return contracts.LibraryHealthReportDTO{
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    status,
		Database: contracts.LibraryHealthDatabaseDTO{
			QuickCheckOK:       quickOK,
			QuickCheckMessages: snapshot.QuickCheckMessages,
			ForeignKeyOK:       len(snapshot.ForeignKeyViolations) == 0,
			ForeignKeyCount:    len(snapshot.ForeignKeyViolations),
		},
		StorageStatuses: storageList.Items,
		Summary:         collector.summary,
		Findings:        collector.findings,
		Truncated:       collector.summary.TotalFindings > len(collector.findings),
	}, nil
}

func addMovieSourceHealthFinding(collector *libraryHealthCollector, movie storage.LibraryHealthMovieRecord) {
	err := validateReadableRegularFile(movie.Location, true)
	if err == nil {
		return
	}
	category := "source_unreadable"
	severity := "warning"
	if errors.Is(err, os.ErrNotExist) {
		category = "source_missing"
		severity = "critical"
	} else if errors.Is(err, errLibraryHealthEmptyFile) {
		category = "source_empty"
	}
	collector.add(contracts.LibraryHealthFindingDTO{
		Category:      category,
		Severity:      severity,
		EntityType:    "movie",
		EntityID:      movie.ID,
		Label:         firstNonEmpty(movie.Code, movie.Title),
		Path:          movie.Location,
		Message:       err.Error(),
		RepairActions: []string{"rescan_storage", "export_diagnostics"},
	})
}

var errLibraryHealthEmptyFile = errors.New("source file is empty")

func validateReadableRegularFile(path string, rejectEmpty bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("file path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("path is not a regular file")
	}
	if rejectEmpty && info.Size() == 0 {
		return errLibraryHealthEmptyFile
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func libraryHealthStatusForPath(path string, statuses []contracts.LibraryPathStorageStatusDTO) (contracts.LibraryPathStorageStatusDTO, bool) {
	bestLength := -1
	var best contracts.LibraryPathStorageStatusDTO
	for _, status := range statuses {
		if pathWithinLibraryHealthRoot(status.Path, path) && len(status.Path) > bestLength {
			best = status
			bestLength = len(status.Path)
		}
	}
	return best, bestLength >= 0
}

func pathWithinLibraryHealthRoot(root, path string) bool {
	root = strings.TrimSpace(root)
	path = strings.TrimSpace(path)
	if root == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func normalizeLibraryHealthPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func isMovieMetadataMissing(movie storage.LibraryHealthMovieRecord) bool {
	return strings.EqualFold(strings.TrimSpace(movie.Summary), "Metadata pending scrape.")
}

func hasMoviePosterAsset(types map[string]bool) bool {
	for assetType, present := range types {
		if present && (strings.Contains(assetType, "cover") || strings.Contains(assetType, "poster") || strings.Contains(assetType, "thumb")) {
			return true
		}
	}
	return false
}

func addDuplicateMovieFindings(collector *libraryHealthCollector, category string, groups map[string][]storage.LibraryHealthMovieRecord, message string) {
	keys := make([]string, 0, len(groups))
	for key, movies := range groups {
		if len(movies) > 1 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		movies := groups[key]
		ids := make([]string, 0, len(movies))
		paths := make([]string, 0, len(movies))
		for _, movie := range movies {
			ids = append(ids, movie.ID)
			paths = append(paths, movie.Location)
		}
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:      category,
			Severity:      "warning",
			EntityType:    "movie_group",
			EntityID:      key,
			Label:         key,
			Message:       message,
			Details:       map[string]any{"movieIds": ids, "paths": paths},
			RepairActions: []string{"export_diagnostics"},
		})
	}
}

func addImportStagingFindings(collector *libraryHealthCollector, status contracts.LibraryPathStorageStatusDTO, registered map[string]struct{}) {
	stagingRoot := filepath.Join(status.Path, movieImportUploadStagingDirName)
	entries, err := os.ReadDir(stagingRoot)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:      "import_staging_residue",
			Severity:      "warning",
			EntityType:    "library_path",
			EntityID:      status.LibraryPathID,
			Label:         status.Title,
			Path:          stagingRoot,
			Message:       "staging root cannot be inspected: " + err.Error(),
			RepairActions: []string{"export_diagnostics"},
		})
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(stagingRoot, name)
		info, infoErr := os.Lstat(fullPath)
		if infoErr != nil {
			collector.add(contracts.LibraryHealthFindingDTO{
				Category: "import_staging_residue", Severity: "warning", EntityType: "staging_entry",
				EntityID: name, Label: name, Path: fullPath, Message: infoErr.Error(), RepairActions: []string{"export_diagnostics"},
			})
			continue
		}
		_, registeredSession := registered[name]
		validDir := info.IsDir() && info.Mode()&os.ModeSymlink == 0 && libraryHealthUploadDirPattern.MatchString(name)
		if validDir && registeredSession {
			continue
		}
		message := "unregistered upload staging directory"
		actions := []string{"cleanup_import_staging", "export_diagnostics"}
		if !validDir {
			message = "out-of-policy entry exists under .curated-import; automatic cleanup is disabled"
			actions = []string{"export_diagnostics"}
		}
		collector.add(contracts.LibraryHealthFindingDTO{
			Category:      "import_staging_residue",
			Severity:      "warning",
			EntityType:    "staging_entry",
			EntityID:      name,
			Label:         name,
			Path:          fullPath,
			Message:       message,
			Details:       map[string]any{"registered": registeredSession, "strictUploadDirectory": validDir},
			RepairActions: actions,
		})
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Unknown"
}
