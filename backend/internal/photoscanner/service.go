package photoscanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/photoarchive"
	"curated-backend/internal/storage"
)

type Store interface {
	GetPhotoBookByLocation(context.Context, string) (contracts.PhotoBookDetailDTO, error)
	UpsertPhotoBook(context.Context, storage.PhotoBookUpsert) (contracts.PhotoBookDetailDTO, error)
	ReplacePhotoPages(context.Context, string, []storage.PhotoPageInput) error
}

type Service struct {
	store Store
}

type Summary struct {
	FilesDiscovered int
	Imported        int
	Updated         int
	Skipped         int
	Errors          []FileError
}

type FileError struct {
	Path      string
	ErrorCode string
	Message   string
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Scan(ctx context.Context, paths []contracts.PhotoLibraryPathDTO) (Summary, error) {
	var summary Summary
	for _, libraryPath := range paths {
		root := filepath.Clean(strings.TrimSpace(libraryPath.Path))
		if root == "" || root == "." {
			continue
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				summary.Skipped++
				summary.Errors = append(summary.Errors, FileError{
					Path:      p,
					ErrorCode: contracts.ErrorCodePhotoArchiveReadFailed,
					Message:   walkErr.Error(),
				})
				return nil
			}
			if d.IsDir() || !photoarchive.IsSupportedArchivePath(p) {
				return nil
			}
			summary.FilesDiscovered++
			s.scanArchive(ctx, libraryPath.ID, p, &summary)
			return nil
		})
		if err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func (s *Service) scanArchive(ctx context.Context, libraryPathID string, archivePath string, summary *Summary) {
	archivePath = filepath.Clean(archivePath)
	_, existingErr := s.store.GetPhotoBookByLocation(ctx, archivePath)
	alreadyIndexed := existingErr == nil
	if existingErr != nil && !errors.Is(existingErr, storage.ErrPhotoBookNotFound) {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodePhotoArchiveReadFailed,
			Message:   existingErr.Error(),
		})
		return
	}

	pages, err := photoarchive.ListPages(ctx, archivePath)
	if err != nil {
		summary.Skipped++
		code := contracts.ErrorCodePhotoArchiveReadFailed
		if errors.Is(err, photoarchive.ErrArchiveEmpty) {
			code = contracts.ErrorCodePhotoArchiveEmpty
		} else if errors.Is(err, photoarchive.ErrUnsupportedArchive) {
			code = contracts.ErrorCodePhotoArchiveUnsupported
		}
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: code,
			Message:   err.Error(),
		})
		return
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodePhotoArchiveReadFailed,
			Message:   err.Error(),
		})
		return
	}

	book, err := s.store.UpsertPhotoBook(ctx, storage.PhotoBookUpsert{
		LibraryPathID:  libraryPathID,
		Location:       archivePath,
		SourceFileName: filepath.Base(archivePath),
		Title:          strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath)),
		FileSize:       info.Size(),
		FileModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
		PageCount:      len(pages),
	})
	if err != nil {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodePhotoArchiveReadFailed,
			Message:   err.Error(),
		})
		return
	}

	inputs := make([]storage.PhotoPageInput, 0, len(pages))
	for _, page := range pages {
		inputs = append(inputs, storage.PhotoPageInput{
			Index:     page.Index,
			EntryPath: page.EntryPath,
			FileName:  page.FileName,
			ImageExt:  page.ImageExt,
			SizeBytes: page.SizeBytes,
		})
	}
	if err := s.store.ReplacePhotoPages(ctx, book.ID, inputs); err != nil {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodePhotoArchiveReadFailed,
			Message:   err.Error(),
		})
		return
	}
	if alreadyIndexed {
		summary.Updated++
	} else {
		summary.Imported++
	}
}
