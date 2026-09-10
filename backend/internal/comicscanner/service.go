package comicscanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/comicarchive"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

type Store interface {
	GetComicBookByLocation(context.Context, string) (contracts.ComicBookDetailDTO, error)
	UpsertComicBook(context.Context, storage.ComicBookUpsert) (contracts.ComicBookDetailDTO, error)
	ReplaceComicPages(context.Context, string, []storage.ComicPageInput) error
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

func (s *Service) Scan(ctx context.Context, paths []contracts.ComicLibraryPathDTO) (Summary, error) {
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
					ErrorCode: contracts.ErrorCodeComicArchiveReadFailed,
					Message:   walkErr.Error(),
				})
				return nil
			}
			if d.IsDir() || !comicarchive.IsSupportedArchivePath(p) {
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
	_, existingErr := s.store.GetComicBookByLocation(ctx, archivePath)
	alreadyIndexed := existingErr == nil
	if existingErr != nil && !errors.Is(existingErr, storage.ErrComicBookNotFound) {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodeComicArchiveReadFailed,
			Message:   existingErr.Error(),
		})
		return
	}

	pages, err := comicarchive.ListPages(ctx, archivePath)
	if err != nil {
		summary.Skipped++
		code := contracts.ErrorCodeComicArchiveReadFailed
		if errors.Is(err, comicarchive.ErrArchiveEmpty) {
			code = contracts.ErrorCodeComicArchiveEmpty
		} else if errors.Is(err, comicarchive.ErrUnsupportedArchive) {
			code = contracts.ErrorCodeComicArchiveUnsupported
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
			ErrorCode: contracts.ErrorCodeComicArchiveReadFailed,
			Message:   err.Error(),
		})
		return
	}

	book, err := s.store.UpsertComicBook(ctx, storage.ComicBookUpsert{
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
			ErrorCode: contracts.ErrorCodeComicArchiveReadFailed,
			Message:   err.Error(),
		})
		return
	}

	inputs := make([]storage.ComicPageInput, 0, len(pages))
	for _, page := range pages {
		inputs = append(inputs, storage.ComicPageInput{
			Index:     page.Index,
			EntryPath: page.EntryPath,
			FileName:  page.FileName,
			ImageExt:  page.ImageExt,
			SizeBytes: page.SizeBytes,
		})
	}
	if err := s.store.ReplaceComicPages(ctx, book.ID, inputs); err != nil {
		summary.Skipped++
		summary.Errors = append(summary.Errors, FileError{
			Path:      archivePath,
			ErrorCode: contracts.ErrorCodeComicArchiveReadFailed,
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
