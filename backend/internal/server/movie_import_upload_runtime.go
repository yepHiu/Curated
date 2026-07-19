package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const (
	movieImportUploadRecoveryReady         = "ready"
	movieImportUploadRecoveryUnavailable   = "unavailable"
	movieImportUploadRecoveryUnrecoverable = "unrecoverable"
)

func (s *movieImportUploadSessionStore) available() error {
	if s == nil || s.persistence == nil || s.tasks == nil {
		return errors.New("movie import upload persistence is unavailable")
	}
	if s.initErr != nil {
		return fmt.Errorf("movie import upload recovery failed: %w", s.initErr)
	}
	return nil
}

func (s *movieImportUploadSessionStore) currentTime() time.Time {
	now := time.Now
	if s != nil && s.now != nil {
		now = s.now
	}
	return now().UTC()
}

func (s *movieImportUploadSessionStore) persistNew(ctx context.Context, session *movieImportUploadSession) error {
	if err := s.available(); err != nil {
		return err
	}
	return s.persistence.CreateMovieImportUploadSession(ctx, session.storageRecord())
}

func (s *movieImportUploadSessionStore) refreshUnavailable(ctx context.Context, session *movieImportUploadSession) error {
	if session == nil || session.recoveryStatus != movieImportUploadRecoveryUnavailable {
		return nil
	}
	record, err := s.persistence.GetMovieImportUploadSession(ctx, session.uploadID)
	if err != nil {
		return err
	}
	recovered, err := s.reconcileRecord(ctx, record)
	if err != nil {
		return err
	}
	if recovered.recoveryStatus != movieImportUploadRecoveryReady {
		session.recoveryStatus = recovered.recoveryStatus
		session.recoveryError = recovered.recoveryError
		session.state = recovered.state
		return errors.New(recovered.recoveryError)
	}
	session.taskID = recovered.taskID
	session.targetLibraryPathID = recovered.targetLibraryPathID
	session.targetRoot = recovered.targetRoot
	session.stagingDir = recovered.stagingDir
	session.state = recovered.state
	session.chunkSize = recovered.chunkSize
	session.createdAt = recovered.createdAt
	session.updatedAt = recovered.updatedAt
	session.expiresAt = recovered.expiresAt
	session.totalBytes = recovered.totalBytes
	session.bytesReceived = recovered.bytesReceived
	session.recoveryStatus = recovered.recoveryStatus
	session.recoveryError = recovered.recoveryError
	session.files = recovered.files
	session.fileOrder = recovered.fileOrder
	s.tasks.Restore(recoveredMovieImportUploadTask(session))
	return nil
}

func (s *movieImportUploadSessionStore) recover(ctx context.Context) error {
	if s == nil || s.persistence == nil || s.tasks == nil {
		return nil
	}
	records, err := s.persistence.ListRecoverableMovieImportUploadSessions(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		session, reconcileErr := s.reconcileRecord(ctx, record)
		if reconcileErr != nil {
			return fmt.Errorf("reconcile movie import upload %s: %w", record.UploadID, reconcileErr)
		}
		s.put(session)
		task := recoveredMovieImportUploadTask(session)
		if s.tasks.Restore(task) {
			if err := s.persistence.SaveTask(ctx, task); err != nil && s.logger != nil {
				s.logger.Warn("save recovered movie import task failed",
					zap.String("uploadId", session.uploadID),
					zap.String("taskId", session.taskID),
					zap.Error(err))
			}
		}
	}
	return nil
}

func (s *movieImportUploadSessionStore) reconcileRecord(
	ctx context.Context,
	record storage.MovieImportUploadSessionRecord,
) (*movieImportUploadSession, error) {
	session, err := movieImportUploadSessionFromRecord(record)
	if err != nil {
		session = movieImportUploadSessionFromRecordBestEffort(record, s.currentTime())
		return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateUnrecoverable,
			"invalid_persisted_record", err.Error())
	}
	now := s.currentTime()
	if !session.expiresAt.IsZero() && !session.expiresAt.After(now) {
		return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateExpired,
			"upload_expired", "upload session expired before restart recovery")
	}

	if err := validateMovieImportUploadRecordPaths(record); err != nil {
		return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateUnrecoverable,
			"invalid_persisted_paths", err.Error())
	}
	fileBytes, totalBytes, err := validateMovieImportUploadChunkRanges(record)
	if err != nil {
		return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateUnrecoverable,
			"invalid_chunk_ranges", err.Error())
	}
	for fileID, bytesReceived := range fileBytes {
		if file := session.files[fileID]; file != nil {
			file.bytesReceived = bytesReceived
		}
	}
	session.bytesReceived = totalBytes
	session.updatedAt = now
	if err := s.persistence.ReconcileMovieImportUploadCounters(ctx, session.uploadID, fileBytes,
		totalBytes, now, session.expiresAt); err != nil {
		return nil, err
	}

	rootInfo, rootErr := os.Stat(record.TargetRoot)
	if rootErr != nil || !rootInfo.IsDir() {
		session.recoveryStatus = movieImportUploadRecoveryUnavailable
		if rootErr != nil {
			session.recoveryError = fmt.Sprintf("target storage unavailable: %v", rootErr)
		} else {
			session.recoveryError = "target storage path is not a directory"
		}
		if err := s.persistence.TransitionMovieImportUploadState(ctx, session.uploadID,
			[]string{session.state}, session.state, now, formatUploadRuntimeTime(session.expiresAt), "",
			"target_storage_unavailable", session.recoveryError); err != nil {
			return nil, err
		}
		return session, nil
	}

	if record.State == movieImportUploadStateCommitting {
		if err := s.reconcileCommittingFiles(ctx, &record); err != nil {
			return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateUnrecoverable,
				"commit_reconciliation_failed", err.Error())
		}
		session, err = movieImportUploadSessionFromRecord(record)
		if err != nil {
			return nil, err
		}
	}
	if err := validateMovieImportUploadStagingFiles(record); err != nil {
		return s.markRecoveredSessionTerminal(ctx, session, movieImportUploadStateUnrecoverable,
			"staging_validation_failed", err.Error())
	}
	for fileID, bytesReceived := range fileBytes {
		if file := session.files[fileID]; file != nil {
			file.bytesReceived = bytesReceived
		}
	}
	session.bytesReceived = totalBytes
	session.updatedAt = now

	session.recoveryStatus = movieImportUploadRecoveryReady
	session.recoveryError = ""
	return session, nil
}

func (s *movieImportUploadSessionStore) markRecoveredSessionTerminal(
	ctx context.Context,
	session *movieImportUploadSession,
	state string,
	code string,
	message string,
) (*movieImportUploadSession, error) {
	now := s.currentTime()
	if err := s.persistence.TransitionMovieImportUploadState(ctx, session.uploadID,
		[]string{movieImportUploadStateUploading, movieImportUploadStateCommitting}, state,
		now, "", formatUploadRuntimeTime(now.Add(s.terminalCleanupDelay)), code, message); err != nil {
		return nil, err
	}
	session.state = state
	session.updatedAt = now
	session.recoveryStatus = movieImportUploadRecoveryUnrecoverable
	session.recoveryError = message
	return session, nil
}

func (s *movieImportUploadSessionStore) reconcileCommittingFiles(
	ctx context.Context,
	record *storage.MovieImportUploadSessionRecord,
) error {
	for index := range record.Files {
		file := &record.Files[index]
		stagingInfo, stagingErr := os.Lstat(file.StagingPath)
		finalInfo, finalErr := os.Lstat(file.FinalPath)
		stagingExists := stagingErr == nil && stagingInfo.Mode()&os.ModeSymlink == 0 && stagingInfo.Mode().IsRegular()
		finalExists := finalErr == nil && finalInfo.Mode()&os.ModeSymlink == 0 && finalInfo.Mode().IsRegular()
		if stagingErr != nil && !errors.Is(stagingErr, os.ErrNotExist) {
			return fmt.Errorf("inspect staging file %s: %w", file.FileID, stagingErr)
		}
		if finalErr != nil && !errors.Is(finalErr, os.ErrNotExist) {
			return fmt.Errorf("inspect final file %s: %w", file.FileID, finalErr)
		}
		if file.State == storage.MovieImportUploadFileStateCommitted {
			if !finalExists || finalInfo.Size() != file.Size {
				return fmt.Errorf("committed file %s is missing or has unexpected size", file.FileID)
			}
			continue
		}

		switch {
		case stagingExists && !finalExists:
			if stagingInfo.Size() != file.Size {
				return fmt.Errorf("staging file %s has size %d instead of %d", file.FileID, stagingInfo.Size(), file.Size)
			}
			continue
		case !stagingExists && finalExists:
			if finalInfo.Size() != file.Size {
				return fmt.Errorf("partially committed file %s has unexpected final size", file.FileID)
			}
		case stagingExists && finalExists:
			if stagingInfo.Size() != file.Size || finalInfo.Size() != file.Size {
				return fmt.Errorf("partially committed file %s has inconsistent sizes", file.FileID)
			}
			same := os.SameFile(stagingInfo, finalInfo)
			if !same {
				var err error
				same, err = movieImportFilesEqual(file.StagingPath, file.FinalPath)
				if err != nil {
					return fmt.Errorf("compare partially committed file %s: %w", file.FileID, err)
				}
			}
			if !same {
				return fmt.Errorf("final path conflict for partially committed file %s", file.FileID)
			}
			if err := os.Remove(file.StagingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove duplicate staging link for %s: %w", file.FileID, err)
			}
		case !stagingExists && !finalExists:
			return fmt.Errorf("both staging and final file are missing for %s", file.FileID)
		}

		committedAt := s.currentTime()
		if err := s.persistence.MarkMovieImportUploadFileCommitted(ctx, record.UploadID, file.FileID, committedAt); err != nil {
			return err
		}
		file.State = storage.MovieImportUploadFileStateCommitted
		file.CommittedAt = formatUploadRuntimeTime(committedAt)
	}
	return nil
}

func validateMovieImportUploadRecordPaths(record storage.MovieImportUploadSessionRecord) error {
	if !isStrictMovieImportUploadID(record.UploadID) {
		return errors.New("upload id does not match the generated identifier format")
	}
	if !filepath.IsAbs(record.TargetRoot) || !filepath.IsAbs(record.StagingDir) {
		return errors.New("target root and staging directory must be absolute")
	}
	expectedStaging := filepath.Join(record.TargetRoot, movieImportUploadStagingDirName, record.UploadID)
	if !sameUploadRuntimePath(record.StagingDir, expectedStaging) {
		return fmt.Errorf("staging directory is outside the expected upload path")
	}
	for _, file := range record.Files {
		if !isStrictMovieImportFileID(file.FileID) {
			return fmt.Errorf("file id %q does not match the generated identifier format", file.FileID)
		}
		expectedStagingFile := filepath.Join(record.StagingDir, file.FileID+".part")
		if !sameUploadRuntimePath(file.StagingPath, expectedStagingFile) {
			return fmt.Errorf("staging path for %s is outside the upload directory", file.FileID)
		}
		expectedFinal, err := importDestinationPath(record.TargetRoot, file.RelativePath)
		if err != nil || !sameUploadRuntimePath(file.FinalPath, expectedFinal) {
			return fmt.Errorf("final path for %s is outside the target root", file.FileID)
		}
	}
	return nil
}

func validateMovieImportUploadChunkRanges(record storage.MovieImportUploadSessionRecord) (map[string]int64, int64, error) {
	if len(record.Files) == 0 {
		return nil, 0, errors.New("session has no persisted files")
	}
	fileBytes := make(map[string]int64, len(record.Files))
	var total, declaredTotal int64
	for ordinal, file := range record.Files {
		if file.Ordinal != ordinal {
			return nil, 0, fmt.Errorf("file %s has non-contiguous ordinal %d", file.FileID, file.Ordinal)
		}
		if file.Size <= 0 || declaredTotal > record.TotalBytes-file.Size {
			return nil, 0, fmt.Errorf("file %s has invalid manifest size", file.FileID)
		}
		declaredTotal += file.Size
		chunks := append([]storage.MovieImportUploadChunkRecord(nil), file.Chunks...)
		sort.Slice(chunks, func(i, j int) bool {
			if chunks[i].Offset != chunks[j].Offset {
				return chunks[i].Offset < chunks[j].Offset
			}
			return chunks[i].ChunkIndex < chunks[j].ChunkIndex
		})
		var bytesReceived, previousEnd int64
		for chunkIndex, chunk := range chunks {
			if chunk.ChunkIndex < 0 || chunk.Offset < 0 || chunk.Size <= 0 || chunk.Offset > file.Size-chunk.Size {
				return nil, 0, fmt.Errorf("file %s has an out-of-bounds chunk", file.FileID)
			}
			if chunkIndex > 0 && chunk.Offset < previousEnd {
				return nil, 0, fmt.Errorf("file %s has overlapping chunk ranges", file.FileID)
			}
			previousEnd = chunk.Offset + chunk.Size
			bytesReceived += chunk.Size
		}
		if bytesReceived > file.Size {
			return nil, 0, fmt.Errorf("file %s chunk bytes exceed the manifest", file.FileID)
		}
		fileBytes[file.FileID] = bytesReceived
		total += bytesReceived
	}
	if declaredTotal != record.TotalBytes {
		return nil, 0, fmt.Errorf("session file sizes %d do not match total %d", declaredTotal, record.TotalBytes)
	}
	if total > record.TotalBytes {
		return nil, 0, errors.New("session chunk bytes exceed the manifest")
	}
	return fileBytes, total, nil
}

func validateMovieImportUploadStagingFiles(record storage.MovieImportUploadSessionRecord) error {
	stagingInfo, err := os.Lstat(record.StagingDir)
	if err != nil {
		return fmt.Errorf("inspect staging directory: %w", err)
	}
	if stagingInfo.Mode()&os.ModeSymlink != 0 || !stagingInfo.IsDir() {
		return errors.New("staging directory is not a normal directory")
	}
	for _, file := range record.Files {
		if file.State == storage.MovieImportUploadFileStateCommitted {
			info, err := os.Lstat(file.FinalPath)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != file.Size {
				return fmt.Errorf("committed final file %s is unavailable or invalid", file.FileID)
			}
			continue
		}
		info, err := os.Lstat(file.StagingPath)
		if err != nil {
			return fmt.Errorf("inspect staging file %s: %w", file.FileID, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != file.Size {
			return fmt.Errorf("staging file %s is not a regular file of the declared size", file.FileID)
		}
	}
	return nil
}

func movieImportUploadSessionFromRecord(record storage.MovieImportUploadSessionRecord) (*movieImportUploadSession, error) {
	createdAt, err := parseUploadRuntimeTime(record.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse upload createdAt: %w", err)
	}
	updatedAt, err := parseUploadRuntimeTime(record.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse upload updatedAt: %w", err)
	}
	expiresAt, err := parseUploadRuntimeTime(record.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse upload expiresAt: %w", err)
	}
	session := &movieImportUploadSession{
		uploadID:            record.UploadID,
		taskID:              record.TaskID,
		targetLibraryPathID: record.TargetLibraryPathID,
		targetRoot:          record.TargetRoot,
		stagingDir:          record.StagingDir,
		state:               record.State,
		chunkSize:           record.ChunkSize,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
		expiresAt:           expiresAt,
		totalBytes:          record.TotalBytes,
		bytesReceived:       record.BytesReceived,
		recoveryStatus:      movieImportUploadRecoveryReady,
		files:               make(map[string]*movieImportUploadFile, len(record.Files)),
	}
	for _, persistedFile := range record.Files {
		file := &movieImportUploadFile{
			fileID:        persistedFile.FileID,
			relativePath:  persistedFile.RelativePath,
			safeRelPath:   persistedFile.RelativePath,
			size:          persistedFile.Size,
			stagingPath:   persistedFile.StagingPath,
			finalPath:     persistedFile.FinalPath,
			bytesReceived: persistedFile.BytesReceived,
			state:         persistedFile.State,
			chunks:        make(map[int]movieImportUploadedChunk, len(persistedFile.Chunks)),
		}
		if persistedFile.CommittedAt != "" {
			committedAt, err := parseUploadRuntimeTime(persistedFile.CommittedAt)
			if err != nil {
				return nil, fmt.Errorf("parse file %s committedAt: %w", persistedFile.FileID, err)
			}
			file.committedAt = committedAt
		}
		for _, chunk := range persistedFile.Chunks {
			file.chunks[chunk.ChunkIndex] = movieImportUploadedChunk{offset: chunk.Offset, size: chunk.Size}
		}
		session.files[file.fileID] = file
		session.fileOrder = append(session.fileOrder, file.fileID)
	}
	return session, nil
}

func movieImportUploadSessionFromRecordBestEffort(record storage.MovieImportUploadSessionRecord, fallback time.Time) *movieImportUploadSession {
	createdAt, err := parseUploadRuntimeTime(record.CreatedAt)
	if err != nil {
		createdAt = fallback
	}
	updatedAt, err := parseUploadRuntimeTime(record.UpdatedAt)
	if err != nil {
		updatedAt = fallback
	}
	expiresAt, err := parseUploadRuntimeTime(record.ExpiresAt)
	if err != nil {
		expiresAt = fallback
	}
	session := &movieImportUploadSession{
		uploadID:            record.UploadID,
		taskID:              record.TaskID,
		targetLibraryPathID: record.TargetLibraryPathID,
		targetRoot:          record.TargetRoot,
		stagingDir:          record.StagingDir,
		state:               record.State,
		chunkSize:           record.ChunkSize,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
		expiresAt:           expiresAt,
		totalBytes:          record.TotalBytes,
		bytesReceived:       record.BytesReceived,
		files:               make(map[string]*movieImportUploadFile, len(record.Files)),
	}
	for _, persistedFile := range record.Files {
		file := &movieImportUploadFile{
			fileID:        persistedFile.FileID,
			relativePath:  persistedFile.RelativePath,
			safeRelPath:   persistedFile.RelativePath,
			size:          persistedFile.Size,
			stagingPath:   persistedFile.StagingPath,
			finalPath:     persistedFile.FinalPath,
			bytesReceived: persistedFile.BytesReceived,
			state:         persistedFile.State,
			chunks:        make(map[int]movieImportUploadedChunk, len(persistedFile.Chunks)),
		}
		for _, chunk := range persistedFile.Chunks {
			file.chunks[chunk.ChunkIndex] = movieImportUploadedChunk{offset: chunk.Offset, size: chunk.Size}
		}
		session.files[file.fileID] = file
		session.fileOrder = append(session.fileOrder, file.fileID)
	}
	return session
}

func (s *movieImportUploadSession) storageRecord() storage.MovieImportUploadSessionRecord {
	record := storage.MovieImportUploadSessionRecord{
		UploadID:            s.uploadID,
		TaskID:              s.taskID,
		TargetLibraryPathID: s.targetLibraryPathID,
		TargetRoot:          s.targetRoot,
		StagingDir:          s.stagingDir,
		State:               s.state,
		ChunkSize:           s.chunkSize,
		TotalBytes:          s.totalBytes,
		BytesReceived:       s.bytesReceived,
		CreatedAt:           formatUploadRuntimeTime(s.createdAt),
		UpdatedAt:           formatUploadRuntimeTime(s.updatedAt),
		ExpiresAt:           formatUploadRuntimeTime(s.expiresAt),
		Files:               make([]storage.MovieImportUploadFileRecord, 0, len(s.fileOrder)),
	}
	for ordinal, fileID := range s.fileOrder {
		file := s.files[fileID]
		persistedFile := storage.MovieImportUploadFileRecord{
			FileID:        file.fileID,
			Ordinal:       ordinal,
			RelativePath:  file.safeRelPath,
			Size:          file.size,
			StagingPath:   file.stagingPath,
			FinalPath:     file.finalPath,
			BytesReceived: file.bytesReceived,
			State:         file.state,
		}
		if !file.committedAt.IsZero() {
			persistedFile.CommittedAt = formatUploadRuntimeTime(file.committedAt)
		}
		record.Files = append(record.Files, persistedFile)
	}
	return record
}

func (s *movieImportUploadSession) cleanupCandidate(cleanupAfter string) storage.MovieImportUploadCleanupCandidate {
	return storage.MovieImportUploadCleanupCandidate{
		UploadID:     s.uploadID,
		TargetRoot:   s.targetRoot,
		StagingDir:   s.stagingDir,
		State:        s.state,
		ExpiresAt:    formatUploadRuntimeTime(s.expiresAt),
		CleanupAfter: cleanupAfter,
	}
}

func (h *Handler) commitMovieImportUploadFile(
	ctx context.Context,
	session *movieImportUploadSession,
	file *movieImportUploadFile,
) error {
	if file.state == storage.MovieImportUploadFileStateCommitted {
		info, err := os.Lstat(file.finalPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != file.size {
			return fmt.Errorf("committed final file is unavailable or invalid: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(file.finalPath), 0o755); err != nil {
		return err
	}
	stagingInfo, stagingErr := os.Lstat(file.stagingPath)
	finalInfo, finalErr := os.Lstat(file.finalPath)
	stagingExists := stagingErr == nil && stagingInfo.Mode()&os.ModeSymlink == 0 && stagingInfo.Mode().IsRegular()
	finalExists := finalErr == nil && finalInfo.Mode()&os.ModeSymlink == 0 && finalInfo.Mode().IsRegular()
	if stagingErr != nil && !errors.Is(stagingErr, os.ErrNotExist) {
		return stagingErr
	}
	if finalErr != nil && !errors.Is(finalErr, os.ErrNotExist) {
		return finalErr
	}

	switch {
	case stagingExists && !finalExists:
		if stagingInfo.Size() != file.size {
			return fmt.Errorf("staging file size %d does not match %d", stagingInfo.Size(), file.size)
		}
		if err := syncMovieImportStagingFile(file.stagingPath); err != nil {
			return err
		}
		if err := commitMovieImportStagingFile(file.stagingPath, file.finalPath); err != nil {
			return err
		}
	case !stagingExists && finalExists:
		if finalInfo.Size() != file.size {
			return os.ErrExist
		}
	case stagingExists && finalExists:
		if stagingInfo.Size() != file.size || finalInfo.Size() != file.size {
			return os.ErrExist
		}
		same := os.SameFile(stagingInfo, finalInfo)
		if !same {
			var err error
			same, err = movieImportFilesEqual(file.stagingPath, file.finalPath)
			if err != nil {
				return err
			}
		}
		if !same {
			return os.ErrExist
		}
		if err := os.Remove(file.stagingPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	case !stagingExists && !finalExists:
		return os.ErrNotExist
	}

	committedAt := h.importUploads.currentTime()
	if err := h.store.MarkMovieImportUploadFileCommitted(ctx, session.uploadID, file.fileID, committedAt); err != nil {
		return err
	}
	file.state = storage.MovieImportUploadFileStateCommitted
	file.committedAt = committedAt
	return nil
}

func recoveredMovieImportUploadTask(session *movieImportUploadSession) contracts.TaskDTO {
	status := contracts.TaskRunning
	message := "Uploading movies"
	finishedAt := ""
	errorCode := ""
	errorMessage := ""
	if session.state == movieImportUploadStateCommitting {
		message = "Committing uploaded movies"
	}
	if session.recoveryStatus == movieImportUploadRecoveryUnavailable {
		message = "Upload waiting for target storage"
	}
	if session.state == movieImportUploadStateExpired || session.state == movieImportUploadStateUnrecoverable {
		status = contracts.TaskFailed
		message = session.recoveryError
		finishedAt = formatUploadRuntimeTime(session.updatedAt)
		if session.state == movieImportUploadStateExpired {
			errorCode = contracts.ErrorCodeImportUploadExpired
		} else {
			errorCode = contracts.ErrorCodeImportUploadUnrecoverable
		}
		errorMessage = session.recoveryError
	}
	return contracts.TaskDTO{
		TaskID:       session.taskID,
		Type:         contracts.TaskTypeImportMovies,
		Status:       status,
		Progress:     importProgressPercent(session.bytesReceived, session.totalBytes),
		Message:      message,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		CreatedAt:    formatUploadRuntimeTime(session.createdAt),
		StartedAt:    formatUploadRuntimeTime(session.createdAt),
		FinishedAt:   finishedAt,
		Metadata:     session.taskMetadata(session.state),
	}
}

func (s *movieImportUploadSessionStore) startJanitor(ctx context.Context) {
	if s == nil || ctx == nil || s.persistence == nil {
		return
	}
	s.janitorOnce.Do(func() {
		go func() {
			s.runJanitorOnce(ctx)
			interval := s.janitorInterval
			if interval <= 0 {
				interval = 15 * time.Minute
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.runJanitorOnce(ctx)
				}
			}
		}()
	})
}

func (s *movieImportUploadSessionStore) runJanitorOnce(ctx context.Context) {
	if err := s.runJanitor(ctx); err != nil && s.logger != nil && !errors.Is(err, context.Canceled) {
		s.logger.Warn("movie import upload janitor failed", zap.Error(err))
	}
}

func (s *movieImportUploadSessionStore) runJanitor(ctx context.Context) error {
	now := s.currentTime()
	candidates, err := s.persistence.ListMovieImportUploadCleanupCandidates(ctx, now)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if (candidate.State == movieImportUploadStateUploading || candidate.State == movieImportUploadStateCommitting) &&
			movieImportUploadTimeDue(candidate.ExpiresAt, now) {
			previousState := candidate.State
			if err := s.persistence.ExpireMovieImportUploadSession(ctx, candidate.UploadID,
				previousState, now, now.Add(s.terminalCleanupDelay), "upload_expired",
				"upload session expired before completion; committed destination files are never deleted by the upload janitor"); err != nil {
				if errors.Is(err, storage.ErrMovieImportUploadStateConflict) || errors.Is(err, storage.ErrMovieImportUploadNotFound) ||
					errors.Is(err, storage.ErrMovieImportUploadNotDue) {
					continue
				}
				return err
			}
			candidate.State = movieImportUploadStateExpired
			candidate.DiagnosticCode = "upload_expired"
			candidate.DiagnosticMessage = "upload session expired before completion; committed destination files are never deleted by the upload janitor"
			candidate.CleanupAfter = formatUploadRuntimeTime(now.Add(s.terminalCleanupDelay))
			if session, ok := s.get(candidate.UploadID); ok {
				session.mu.Lock()
				session.state = movieImportUploadStateExpired
				session.recoveryStatus = movieImportUploadRecoveryUnrecoverable
				session.recoveryError = candidate.DiagnosticMessage
				task := s.tasks.Fail(session.taskID, contracts.ErrorCodeImportUploadExpired, candidate.DiagnosticMessage)
				_ = s.persistence.SaveTask(ctx, task)
				session.mu.Unlock()
				s.delete(candidate.UploadID)
			}
		}
		if candidate.CleanupAfter != "" && !movieImportUploadTimeDue(candidate.CleanupAfter, now) {
			continue
		}
		if err := s.cleanupPersistedSession(ctx, candidate, "session_"+candidate.State); err != nil && s.logger != nil {
			s.logger.Warn("cleanup movie import upload session failed",
				zap.String("uploadId", candidate.UploadID),
				zap.String("state", candidate.State),
				zap.Error(err))
		}
	}
	return s.cleanupOrphanStagingDirectories(ctx, now)
}

func (s *movieImportUploadSessionStore) cleanupPersistedSession(
	ctx context.Context,
	candidate storage.MovieImportUploadCleanupCandidate,
	reason string,
) error {
	rootInfo, err := os.Stat(candidate.TargetRoot)
	if err != nil || !rootInfo.IsDir() {
		return fmt.Errorf("target root unavailable; cleanup deferred")
	}
	if !filepath.IsAbs(candidate.TargetRoot) || !isStrictMovieImportUploadID(candidate.UploadID) {
		return errors.New("persisted cleanup identity failed safety validation")
	}
	stagingRoot := filepath.Join(candidate.TargetRoot, movieImportUploadStagingDirName)
	expected := filepath.Join(candidate.TargetRoot, movieImportUploadStagingDirName, candidate.UploadID)
	if !sameUploadRuntimePath(candidate.StagingDir, expected) || !uploadRuntimePathDescendsFrom(candidate.StagingDir, stagingRoot) {
		return errors.New("persisted staging directory failed safety validation")
	}
	auditID, err := s.persistence.CreateMovieImportUploadCleanupAudit(ctx, candidate.UploadID,
		reason, candidate.State, candidate.StagingDir, candidate.DiagnosticMessage, s.currentTime())
	if err != nil {
		return err
	}
	outcome := "removed"
	diagnostic := strings.TrimSpace(candidate.DiagnosticMessage)
	if diagnostic == "" {
		diagnostic = "staging directory removed"
	} else {
		diagnostic += "; staging directory removed"
	}
	if err := removeMovieImportUploadStagingDirectory(candidate.StagingDir); err != nil {
		outcome = "failed"
		diagnostic = err.Error()
		_ = s.persistence.CompleteMovieImportUploadCleanupAudit(ctx, auditID, outcome, diagnostic, s.currentTime())
		return err
	}
	if err := s.persistence.CompleteMovieImportUploadCleanupAudit(ctx, auditID, outcome, diagnostic, s.currentTime()); err != nil {
		return err
	}
	if err := s.persistence.DeleteMovieImportUploadSession(ctx, candidate.UploadID); err != nil && !errors.Is(err, storage.ErrMovieImportUploadNotFound) {
		return err
	}
	s.delete(candidate.UploadID)
	return nil
}

func (s *movieImportUploadSessionStore) cleanupOrphanStagingDirectories(ctx context.Context, now time.Time) error {
	sessionIDs, err := s.persistence.ListMovieImportUploadSessionIDs(ctx)
	if err != nil {
		return err
	}
	libraryPaths, err := s.persistence.ListLibraryPaths(ctx)
	if err != nil {
		return err
	}
	for _, libraryPath := range libraryPaths {
		root := filepath.Clean(strings.TrimSpace(libraryPath.Path))
		rootInfo, err := os.Stat(root)
		if err != nil || !rootInfo.IsDir() {
			continue
		}
		stagingRoot := filepath.Join(root, movieImportUploadStagingDirName)
		rootLstat, err := os.Lstat(stagingRoot)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || rootLstat.Mode()&os.ModeSymlink != 0 || !rootLstat.IsDir() {
			continue
		}
		entries, err := os.ReadDir(stagingRoot)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if _, exists := sessionIDs[entry.Name()]; exists || !isStrictMovieImportUploadID(entry.Name()) {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || now.Sub(info.ModTime()) < s.orphanGrace {
				continue
			}
			candidate := filepath.Join(stagingRoot, entry.Name())
			if !sameUploadRuntimePath(candidate, filepath.Join(root, movieImportUploadStagingDirName, entry.Name())) {
				continue
			}
			auditID, err := s.persistence.CreateMovieImportUploadCleanupAudit(ctx, entry.Name(),
				"orphan_staging", "orphan", candidate, "no SQLite upload session exists", now)
			if err != nil {
				return err
			}
			if err := removeMovieImportUploadStagingDirectory(candidate); err != nil {
				_ = s.persistence.CompleteMovieImportUploadCleanupAudit(ctx, auditID, "failed", err.Error(), s.currentTime())
				continue
			}
			if err := s.persistence.CompleteMovieImportUploadCleanupAudit(ctx, auditID, "removed",
				"orphan staging directory removed", s.currentTime()); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeMovieImportUploadStagingDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to remove non-directory or symlink staging path")
	}
	return os.RemoveAll(path)
}

func isStrictMovieImportUploadID(value string) bool {
	return isStrictMovieImportGeneratedID(value, "upload_")
}

func isStrictMovieImportFileID(value string) bool {
	return isStrictMovieImportGeneratedID(value, "file_")
}

func isStrictMovieImportGeneratedID(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(value, prefix)
	if len(suffix) != 16 {
		return false
	}
	for _, char := range suffix {
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return true
}

func uploadRuntimePathDescendsFrom(path, root string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil || relative == "." || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func movieImportFilesEqual(leftPath, rightPath string) (bool, error) {
	left, err := hashMovieImportFile(leftPath)
	if err != nil {
		return false, err
	}
	right, err := hashMovieImportFile(rightPath)
	if err != nil {
		return false, err
	}
	return left == right, nil
}

func hashMovieImportFile(path string) ([sha256.Size]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

func sameUploadRuntimePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func movieImportUploadDestinationKey(path string) string {
	key := filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}

func parseUploadRuntimeTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
}

func formatUploadRuntimeTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func movieImportUploadTimeDue(value string, now time.Time) bool {
	parsed, err := parseUploadRuntimeTime(value)
	return err == nil && !parsed.After(now)
}
