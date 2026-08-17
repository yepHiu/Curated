package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

const (
	defaultMovieImportUploadChunkSize   = 32 * 1024 * 1024
	maxMovieImportUploadManifestBytes   = 2 << 20
	maxMovieImportUploadFiles           = 10_000
	maxMovieImportUploadTotalBytes      = int64(^uint64(0) >> 1)
	movieImportUploadStateUploading     = storage.MovieImportUploadStateUploading
	movieImportUploadStateCommitting    = storage.MovieImportUploadStateCommitting
	movieImportUploadStateCommitted     = storage.MovieImportUploadStateCommitted
	movieImportUploadStateAborted       = storage.MovieImportUploadStateAborted
	movieImportUploadStateExpired       = storage.MovieImportUploadStateExpired
	movieImportUploadStateUnrecoverable = storage.MovieImportUploadStateUnrecoverable
	movieImportUploadStagingDirName     = ".curated-import"
)

var errMovieImportUploadChunkInvalid = errors.New("invalid movie import upload chunk")

type movieImportUploadSessionStore struct {
	mu                   sync.RWMutex
	sessions             map[string]*movieImportUploadSession
	persistence          *storage.SQLiteStore
	tasks                *tasks.Manager
	logger               *zap.Logger
	now                  func() time.Time
	activeTTL            time.Duration
	orphanGrace          time.Duration
	terminalCleanupDelay time.Duration
	janitorInterval      time.Duration
	initErr              error
	janitorOnce          sync.Once
}

func newMovieImportUploadSessionStore(persistence *storage.SQLiteStore, taskManager *tasks.Manager, logger *zap.Logger) *movieImportUploadSessionStore {
	return &movieImportUploadSessionStore{
		sessions:             make(map[string]*movieImportUploadSession),
		persistence:          persistence,
		tasks:                taskManager,
		logger:               logger,
		now:                  time.Now,
		activeTTL:            24 * time.Hour,
		orphanGrace:          24 * time.Hour,
		terminalCleanupDelay: time.Minute,
		janitorInterval:      15 * time.Minute,
	}
}

func (s *movieImportUploadSessionStore) put(session *movieImportUploadSession) {
	s.mu.Lock()
	s.sessions[session.uploadID] = session
	s.mu.Unlock()
}

func (s *movieImportUploadSessionStore) get(uploadID string) (*movieImportUploadSession, bool) {
	s.mu.RLock()
	session, ok := s.sessions[uploadID]
	s.mu.RUnlock()
	return session, ok
}

func (s *movieImportUploadSessionStore) delete(uploadID string) {
	s.mu.Lock()
	delete(s.sessions, uploadID)
	s.mu.Unlock()
}

type movieImportUploadSession struct {
	mu                  sync.Mutex
	uploadID            string
	taskID              string
	targetLibraryPathID string
	targetRoot          string
	stagingDir          string
	state               string
	chunkSize           int64
	createdAt           time.Time
	updatedAt           time.Time
	expiresAt           time.Time
	totalBytes          int64
	bytesReceived       int64
	recoveryStatus      string
	recoveryError       string
	files               map[string]*movieImportUploadFile
	fileOrder           []string
}

type movieImportUploadFile struct {
	fileID        string
	relativePath  string
	safeRelPath   string
	size          int64
	stagingPath   string
	finalPath     string
	bytesReceived int64
	state         string
	committedAt   time.Time
	chunks        map[int]movieImportUploadedChunk
}

type movieImportUploadedChunk struct {
	offset int64
	size   int64
}

func (h *Handler) handleCreateMovieImportUpload(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.tasks == nil || h.importUploads == nil || h.importUploads.available() != nil {
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "movie import runtime not available")
		return
	}
	target, targetRoot, err := h.loadDefaultImportTarget(r.Context())
	if err != nil {
		writeImportTargetLoadError(w, err, h.logger)
		return
	}

	var body contracts.CreateMovieImportUploadRequest
	if err := decodeStrictJSONBodyLimit(w, r, &body, maxMovieImportUploadManifestBytes); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid upload manifest")
		return
	}
	if len(body.Files) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "no movie files were provided")
		return
	}
	if len(body.Files) > maxMovieImportUploadFiles {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "upload manifest contains too many files")
		return
	}
	manifestTotalBytes := int64(0)
	for _, manifest := range body.Files {
		if manifestTotalBytes > maxMovieImportUploadTotalBytes-manifest.Size {
			manifestTotalBytes = maxMovieImportUploadTotalBytes
			break
		}
		manifestTotalBytes += manifest.Size
	}
	if err := ensureImportDiskSpace(targetRoot, manifestTotalBytes); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeImportNotEnoughSpace, "not enough disk space on the default import path")
		return
	}

	uploadID := newMovieImportUploadID("upload")
	stagingDir := filepath.Join(targetRoot, movieImportUploadStagingDirName, uploadID)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to create upload staging directory")
		return
	}

	now := h.importUploads.currentTime()
	session := &movieImportUploadSession{
		uploadID:            uploadID,
		targetLibraryPathID: target.ID,
		targetRoot:          targetRoot,
		stagingDir:          stagingDir,
		state:               movieImportUploadStateUploading,
		chunkSize:           defaultMovieImportUploadChunkSize,
		createdAt:           now,
		updatedAt:           now,
		expiresAt:           now.Add(h.importUploads.activeTTL),
		recoveryStatus:      movieImportUploadRecoveryReady,
		files:               make(map[string]*movieImportUploadFile),
	}

	seenDestinations := make(map[string]struct{}, len(body.Files))
	for _, manifest := range body.Files {
		file, err := prepareMovieImportUploadFile(stagingDir, targetRoot, manifest)
		if err != nil {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		destinationKey := movieImportUploadDestinationKey(file.finalPath)
		if _, exists := seenDestinations[destinationKey]; exists {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "upload manifest contains duplicate destination paths")
			return
		}
		seenDestinations[destinationKey] = struct{}{}
		if _, err := os.Stat(file.finalPath); err == nil {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeImportConflict, "target file already exists")
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to prepare import target")
			return
		}
		if err := createSizedStagingFile(file.stagingPath, file.size); err != nil {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to create upload staging file")
			return
		}
		if session.totalBytes > maxMovieImportUploadTotalBytes-file.size {
			_ = os.RemoveAll(stagingDir)
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "upload manifest total size is too large")
			return
		}
		session.files[file.fileID] = file
		session.fileOrder = append(session.fileOrder, file.fileID)
		session.totalBytes += file.size
	}

	task := h.tasks.Create(contracts.TaskTypeImportMovies, session.taskMetadata("uploading"))
	task = h.tasks.Start(task.TaskID, "Uploading movies")
	session.taskID = task.TaskID
	if err := h.importUploads.persistNew(r.Context(), session); err != nil {
		_ = os.RemoveAll(stagingDir)
		h.tasks.Fail(task.TaskID, contracts.ErrorCodeImportUploadPersistFailed, "failed to persist upload session")
		if h.logger != nil {
			h.logger.Error("persist movie import upload session failed", zap.String("uploadId", uploadID), zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeImportUploadPersistFailed, "failed to persist upload session")
		return
	}
	h.saveTaskSnapshot(r.Context(), task)
	h.importUploads.put(session)

	writeJSON(w, http.StatusCreated, session.dto(task))
}

func (h *Handler) handleGetMovieImportUpload(w http.ResponseWriter, r *http.Request) {
	session, ok := h.importUploads.get(strings.TrimSpace(r.PathValue("uploadId")))
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload session not found")
		return
	}
	session.mu.Lock()
	_ = h.importUploads.refreshUnavailable(r.Context(), session)
	task := h.uploadTaskSnapshot(session)
	dto := session.dto(task)
	session.mu.Unlock()
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handlePutMovieImportUploadChunk(w http.ResponseWriter, r *http.Request) {
	session, ok := h.importUploads.get(strings.TrimSpace(r.PathValue("uploadId")))
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload session not found")
		return
	}
	fileID := strings.TrimSpace(r.PathValue("fileId"))
	chunkIndex, err := strconv.Atoi(strings.TrimSpace(r.PathValue("chunkIndex")))
	if err != nil || chunkIndex < 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid chunk index")
		return
	}
	offset, err := parseRequiredInt64Header(r, "X-Curated-Offset")
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}
	declaredSize, err := parseOptionalPositiveInt64Header(r, "X-Curated-Chunk-Size")
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
		return
	}

	session.mu.Lock()
	if session.recoveryStatus != movieImportUploadRecoveryReady {
		_ = h.importUploads.refreshUnavailable(r.Context(), session)
	}
	if session.state != movieImportUploadStateUploading {
		session.mu.Unlock()
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "upload session is not accepting chunks")
		return
	}
	file, ok := session.files[fileID]
	if !ok {
		session.mu.Unlock()
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload file not found")
		return
	}
	if offset < 0 || offset >= file.size {
		session.mu.Unlock()
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "chunk offset is outside file bounds")
		return
	}
	if declaredSize > 0 && offset+declaredSize > file.size {
		session.mu.Unlock()
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "chunk exceeds file size")
		return
	}
	if existing, exists := file.chunks[chunkIndex]; exists {
		if existing.offset != offset || (declaredSize > 0 && existing.size != declaredSize) {
			session.mu.Unlock()
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "chunk index already uploaded with different range")
			return
		}
		task := h.uploadTaskSnapshot(session)
		dto := session.dto(task)
		session.mu.Unlock()
		writeJSON(w, http.StatusOK, dto)
		return
	}
	if session.recoveryStatus != movieImportUploadRecoveryReady {
		recoveryError := session.recoveryError
		session.mu.Unlock()
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeImportTargetUnavailable, recoveryError)
		return
	}

	written, err := writeMovieImportChunk(file.stagingPath, r.Body, offset, declaredSize, file.size-offset)
	if err != nil {
		session.mu.Unlock()
		if errors.Is(err, errMovieImportUploadChunkInvalid) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to write upload chunk")
		return
	}
	now := h.importUploads.currentTime()
	persisted, err := h.store.RecordMovieImportUploadChunk(r.Context(), session.uploadID, file.fileID,
		storage.MovieImportUploadChunkRecord{ChunkIndex: chunkIndex, Offset: offset, Size: written},
		now, now.Add(h.importUploads.activeTTL))
	if err != nil {
		session.mu.Unlock()
		switch {
		case errors.Is(err, storage.ErrMovieImportUploadChunkConflict), errors.Is(err, storage.ErrMovieImportUploadChunkOverlap), errors.Is(err, storage.ErrMovieImportUploadStateConflict):
			writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, err.Error())
		case errors.Is(err, storage.ErrMovieImportUploadNotFound):
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload session not found")
		default:
			if h.logger != nil {
				h.logger.Error("persist movie import upload chunk failed", zap.String("uploadId", session.uploadID), zap.String("fileId", file.fileID), zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeImportUploadPersistFailed, "failed to persist upload chunk")
		}
		return
	}
	file.chunks[chunkIndex] = movieImportUploadedChunk{offset: offset, size: written}
	file.bytesReceived = persisted.FileBytesReceived
	session.bytesReceived = persisted.SessionBytesReceived
	session.updatedAt = now
	session.expiresAt = now.Add(h.importUploads.activeTTL)
	progress := importProgressPercent(session.bytesReceived, session.totalBytes)
	task := h.tasks.ProgressWithMetadata(session.taskID, progress, "Uploading movies", session.taskMetadata("uploading"))
	h.saveTaskSnapshot(r.Context(), task)
	dto := session.dto(task)
	session.mu.Unlock()

	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) handleCommitMovieImportUpload(w http.ResponseWriter, r *http.Request) {
	session, ok := h.importUploads.get(strings.TrimSpace(r.PathValue("uploadId")))
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload session not found")
		return
	}

	session.mu.Lock()
	if session.recoveryStatus != movieImportUploadRecoveryReady {
		_ = h.importUploads.refreshUnavailable(r.Context(), session)
	}
	if session.state != movieImportUploadStateUploading && session.state != movieImportUploadStateCommitting {
		session.mu.Unlock()
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "upload session cannot be committed")
		return
	}
	if session.recoveryStatus != movieImportUploadRecoveryReady {
		recoveryError := session.recoveryError
		session.mu.Unlock()
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeImportTargetUnavailable, recoveryError)
		return
	}
	for _, fileID := range session.fileOrder {
		file := session.files[fileID]
		if file.bytesReceived != file.size {
			session.mu.Unlock()
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "upload is incomplete")
			return
		}
	}
	if session.state == movieImportUploadStateUploading {
		for _, fileID := range session.fileOrder {
			file := session.files[fileID]
			if err := os.MkdirAll(filepath.Dir(file.finalPath), 0o755); err != nil {
				session.mu.Unlock()
				writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to create import destination")
				return
			}
			if _, err := os.Stat(file.finalPath); err == nil {
				session.mu.Unlock()
				writeAppError(w, http.StatusConflict, contracts.ErrorCodeImportConflict, "target file already exists")
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				session.mu.Unlock()
				writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to inspect import destination")
				return
			}
			if err := syncMovieImportStagingFile(file.stagingPath); err != nil {
				session.mu.Unlock()
				writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to flush upload staging file")
				return
			}
		}
		now := h.importUploads.currentTime()
		if err := h.store.TransitionMovieImportUploadState(r.Context(), session.uploadID,
			[]string{movieImportUploadStateUploading}, movieImportUploadStateCommitting,
			now, formatUploadRuntimeTime(session.expiresAt), "", "", ""); err != nil {
			session.mu.Unlock()
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to persist upload commit state")
			return
		}
		session.state = movieImportUploadStateCommitting
		session.updatedAt = now
	}

	for _, fileID := range session.fileOrder {
		file := session.files[fileID]
		if err := h.commitMovieImportUploadFile(r.Context(), session, file); err != nil {
			session.mu.Unlock()
			if os.IsExist(err) || errors.Is(err, os.ErrExist) {
				writeAppError(w, http.StatusConflict, contracts.ErrorCodeImportConflict, "target file already exists")
				return
			}
			if h.logger != nil {
				h.logger.Error("commit movie import upload file failed", zap.String("uploadId", session.uploadID), zap.String("fileId", file.fileID), zap.Error(err))
			}
			writeAppError(w, http.StatusInternalServerError, classifyImportCopyError(err), "failed to commit uploaded file")
			return
		}
	}
	now := h.importUploads.currentTime()
	cleanupAfter := now.Add(h.importUploads.terminalCleanupDelay)
	if err := h.store.TransitionMovieImportUploadState(r.Context(), session.uploadID,
		[]string{movieImportUploadStateCommitting}, movieImportUploadStateCommitted,
		now, "", formatUploadRuntimeTime(cleanupAfter), "", ""); err != nil {
		session.mu.Unlock()
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to finalize upload session")
		return
	}
	session.state = movieImportUploadStateCommitted
	session.updatedAt = now
	finalPatch := session.taskMetadata("completed")
	task := h.tasks.ProgressWithMetadata(session.taskID, 100, "Movie import completed", finalPatch)
	h.saveTaskSnapshot(r.Context(), task)
	targetRoot := session.targetRoot
	uploadID := session.uploadID
	cleanupCandidate := session.cleanupCandidate(formatUploadRuntimeTime(cleanupAfter))
	session.mu.Unlock()
	h.importUploads.delete(uploadID)

	responseTask := task
	if h.scanStarter != nil {
		if scanTask, err := h.scanStarter.StartScan(r.Context(), []string{targetRoot}); err == nil && scanTask.TaskID != "" {
			finalPatch["scanTaskId"] = scanTask.TaskID
		} else if err != nil {
			finalPatch["scanError"] = err.Error()
			responseTask = h.tasks.PartialFail(session.taskID, contracts.ErrorCodeImportScanFailed, "movies uploaded, but scan could not start", finalPatch)
			h.saveTaskSnapshot(r.Context(), responseTask)
		}
	}
	if responseTask.Status != contracts.TaskPartialFailed {
		responseTask = h.tasks.ProgressWithMetadata(session.taskID, 100, "Movie import completed", finalPatch)
		h.saveTaskSnapshot(r.Context(), responseTask)
		responseTask = h.tasks.Complete(session.taskID, "Movie import completed")
		h.saveTaskSnapshot(r.Context(), responseTask)
	}
	if err := h.importUploads.cleanupPersistedSession(r.Context(), cleanupCandidate, "session_committed"); err != nil && h.logger != nil {
		h.logger.Warn("defer committed upload cleanup to janitor", zap.String("uploadId", uploadID), zap.Error(err))
	}
	writeJSON(w, http.StatusAccepted, responseTask)
}

func (h *Handler) handleAbortMovieImportUpload(w http.ResponseWriter, r *http.Request) {
	uploadID := strings.TrimSpace(r.PathValue("uploadId"))
	session, ok := h.importUploads.get(uploadID)
	if !ok {
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "upload session not found")
		return
	}
	session.mu.Lock()
	if session.state != movieImportUploadStateUploading {
		session.mu.Unlock()
		writeAppError(w, http.StatusConflict, contracts.ErrorCodeConflict, "upload session cannot be aborted while committing")
		return
	}
	now := h.importUploads.currentTime()
	cleanupAfter := now.Add(h.importUploads.terminalCleanupDelay)
	if err := h.store.TransitionMovieImportUploadState(r.Context(), session.uploadID,
		[]string{movieImportUploadStateUploading}, movieImportUploadStateAborted,
		now, "", formatUploadRuntimeTime(cleanupAfter), "upload_aborted", "upload cancelled by user"); err != nil {
		session.mu.Unlock()
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to persist upload cancellation")
		return
	}
	session.state = movieImportUploadStateAborted
	task := h.tasks.Fail(session.taskID, contracts.ErrorCodeImportCancelled, "movie import upload cancelled")
	h.saveTaskSnapshot(r.Context(), task)
	cleanupCandidate := session.cleanupCandidate(formatUploadRuntimeTime(cleanupAfter))
	session.mu.Unlock()
	h.importUploads.delete(uploadID)
	if err := h.importUploads.cleanupPersistedSession(r.Context(), cleanupCandidate, "session_aborted"); err != nil && h.logger != nil {
		h.logger.Warn("defer aborted upload cleanup to janitor", zap.String("uploadId", uploadID), zap.Error(err))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) loadDefaultImportTarget(ctx context.Context) (contracts.LibraryPathDTO, string, error) {
	targetID := h.defaultImportLibraryPathID()
	if targetID == "" {
		return contracts.LibraryPathDTO{}, "", errImportTargetNotConfigured
	}
	target, err := h.store.GetLibraryPath(ctx, targetID)
	if err != nil {
		if errors.Is(err, storage.ErrLibraryPathNotFound) {
			return contracts.LibraryPathDTO{}, "", errImportTargetUnavailable
		}
		return contracts.LibraryPathDTO{}, "", fmt.Errorf("load default import library path: %w", err)
	}
	targetRoot := filepath.Clean(strings.TrimSpace(target.Path))
	if targetRoot == "" || targetRoot == "." {
		return contracts.LibraryPathDTO{}, "", errImportTargetUnavailable
	}
	if stat, err := os.Stat(targetRoot); err != nil || !stat.IsDir() {
		return contracts.LibraryPathDTO{}, "", errImportTargetUnavailable
	}
	return target, targetRoot, nil
}

var (
	errImportTargetNotConfigured = errors.New("default import library path is not configured")
	errImportTargetUnavailable   = errors.New("default import library path is unavailable")
)

func writeImportTargetLoadError(w http.ResponseWriter, err error, logger *zap.Logger) {
	switch {
	case errors.Is(err, errImportTargetNotConfigured):
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeImportTargetNotConfigured, "default import library path is not configured")
	case errors.Is(err, errImportTargetUnavailable):
		writeAppError(w, http.StatusNotFound, contracts.ErrorCodeImportTargetUnavailable, "default import library path is unavailable")
	default:
		if logger != nil {
			logger.Warn("load default import library path failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load default import library path")
	}
}

func prepareMovieImportUploadFile(stagingDir, targetRoot string, manifest contracts.MovieImportUploadFileManifest) (*movieImportUploadFile, error) {
	if manifest.Size <= 0 {
		return nil, fmt.Errorf("file size must be positive")
	}
	relPath := sanitizeImportRelativePath(manifest.RelativePath, filepath.Base(manifest.RelativePath))
	if !isSupportedImportVideoPath(relPath) {
		return nil, fmt.Errorf("unsupported video file type")
	}
	finalPath, err := importDestinationPath(targetRoot, relPath)
	if err != nil {
		return nil, fmt.Errorf("invalid destination path")
	}
	fileID := newMovieImportUploadID("file")
	return &movieImportUploadFile{
		fileID:       fileID,
		relativePath: relPath,
		safeRelPath:  relPath,
		size:         manifest.Size,
		stagingPath:  filepath.Join(stagingDir, fileID+".part"),
		finalPath:    finalPath,
		state:        storage.MovieImportUploadFileStatePending,
		chunks:       make(map[int]movieImportUploadedChunk),
	}, nil
}

func createSizedStagingFile(path string, size int64) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := file.Truncate(size); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeMovieImportChunk(path string, src io.Reader, offset int64, declaredSize int64, maxSize int64) (int64, error) {
	file, err := os.OpenFile(path, os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	buf := make([]byte, importCopyBufferSize)
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			if declaredSize > 0 && written+int64(nr) > declaredSize {
				return written, fmt.Errorf("%w: body exceeds declared size", errMovieImportUploadChunkInvalid)
			}
			if maxSize <= 0 || written+int64(nr) > maxSize {
				return written, fmt.Errorf("%w: body exceeds file bounds", errMovieImportUploadChunkInvalid)
			}
			nw, ew := file.WriteAt(buf[:nr], offset+written)
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			return written, er
		}
	}
	if declaredSize > 0 && written != declaredSize {
		return written, fmt.Errorf("%w: body size %d does not match declared size %d", errMovieImportUploadChunkInvalid, written, declaredSize)
	}
	if written <= 0 {
		return 0, fmt.Errorf("%w: body is empty", errMovieImportUploadChunkInvalid)
	}
	if err := file.Sync(); err != nil {
		return written, err
	}
	if err := file.Close(); err != nil {
		return written, err
	}
	return written, nil
}

func syncMovieImportStagingFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func commitMovieImportStagingFile(stagingPath string, finalPath string) error {
	if _, err := os.Stat(finalPath); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Link(stagingPath, finalPath); err == nil {
		return os.Remove(stagingPath)
	} else if os.IsExist(err) || errors.Is(err, os.ErrExist) {
		return os.ErrExist
	}

	src, err := os.Open(stagingPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(finalPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) || errors.Is(err, os.ErrExist) {
			return os.ErrExist
		}
		return err
	}
	cleanupDst := true
	defer func() {
		_ = dst.Close()
		if cleanupDst {
			_ = os.Remove(finalPath)
		}
	}()

	if _, err := io.CopyBuffer(dst, src, make([]byte, importCopyBufferSize)); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	cleanupDst = false
	return os.Remove(stagingPath)
}

func (h *Handler) uploadTaskSnapshot(session *movieImportUploadSession) contracts.TaskDTO {
	if h.tasks == nil {
		return contracts.TaskDTO{}
	}
	task, ok := h.tasks.Get(session.taskID)
	if !ok {
		return contracts.TaskDTO{}
	}
	return task
}

func (s *movieImportUploadSession) taskMetadata(stage string) map[string]any {
	completedFiles := 0
	currentFileName := ""
	for _, fileID := range s.fileOrder {
		file := s.files[fileID]
		if file.bytesReceived == file.size {
			completedFiles++
		} else if currentFileName == "" {
			currentFileName = filepath.Base(file.safeRelPath)
		}
	}
	return map[string]any{
		"uploadId":             s.uploadID,
		"targetLibraryPathId":  s.targetLibraryPathID,
		"targetPath":           s.targetRoot,
		"stage":                stage,
		"totalFiles":           len(s.fileOrder),
		"completedFiles":       completedFiles,
		"failedFiles":          0,
		"copiedBytes":          s.bytesReceived,
		"totalBytes":           s.totalBytes,
		"currentFileName":      currentFileName,
		"resumableUpload":      true,
		"uploadBytesReceived":  s.bytesReceived,
		"uploadTotalBytes":     s.totalBytes,
		"uploadSessionState":   s.state,
		"uploadExpiresAt":      formatUploadRuntimeTime(s.expiresAt),
		"uploadRecoveryStatus": s.recoveryStatus,
		"uploadRecoveryError":  s.recoveryError,
	}
}

func (s *movieImportUploadSession) dto(task contracts.TaskDTO) contracts.MovieImportUploadDTO {
	files := make([]contracts.MovieImportUploadFileDTO, 0, len(s.fileOrder))
	for _, fileID := range s.fileOrder {
		file := s.files[fileID]
		chunks := make([]contracts.MovieImportUploadChunkDTO, 0, len(file.chunks))
		indexes := make([]int, 0, len(file.chunks))
		for index := range file.chunks {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			uploaded := file.chunks[index]
			chunks = append(chunks, contracts.MovieImportUploadChunkDTO{
				Index:  int64(index),
				Offset: uploaded.offset,
				Size:   uploaded.size,
			})
		}
		files = append(files, contracts.MovieImportUploadFileDTO{
			FileID:        file.fileID,
			RelativePath:  file.safeRelPath,
			Size:          file.size,
			BytesReceived: file.bytesReceived,
			Complete:      file.bytesReceived == file.size,
			State:         file.state,
			Chunks:        chunks,
		})
	}
	return contracts.MovieImportUploadDTO{
		UploadID:       s.uploadID,
		TargetPath:     s.targetRoot,
		ChunkSize:      s.chunkSize,
		BytesReceived:  s.bytesReceived,
		TotalBytes:     s.totalBytes,
		State:          s.state,
		ExpiresAt:      formatUploadRuntimeTime(s.expiresAt),
		RecoveryStatus: s.recoveryStatus,
		RecoveryError:  s.recoveryError,
		Files:          files,
		Task:           task,
	}
}

func parseRequiredInt64Header(r *http.Request, name string) (int64, error) {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		return 0, fmt.Errorf("%s header is required", name)
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s header is invalid", name)
	}
	return n, nil
}

func parseOptionalPositiveInt64Header(r *http.Request, name string) (int64, error) {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s header is invalid", name)
	}
	return n, nil
}

func newMovieImportUploadID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%016x", prefix, uint64(time.Now().UnixNano()))
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
