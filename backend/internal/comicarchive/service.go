package comicarchive

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedArchive = errors.New("unsupported comic archive")
	ErrArchiveEmpty       = errors.New("comic archive contains no supported image pages")
	ErrPageNotFound       = errors.New("comic page not found")
)

type PageEntry struct {
	Index     int
	EntryPath string
	FileName  string
	ImageExt  string
	SizeBytes int64
}

func IsSupportedArchivePath(p string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(p))) {
	case ".zip", ".cbz":
		return true
	default:
		return false
	}
}

func IsSupportedImageEntry(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(name, "/") {
		return false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func ListPages(ctx context.Context, archivePath string) ([]PageEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !IsSupportedArchivePath(archivePath) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedArchive, archivePath)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	pages := make([]PageEntry, 0)
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file.FileInfo().IsDir() || !IsSupportedImageEntry(file.Name) {
			continue
		}
		pages = append(pages, PageEntry{
			EntryPath: path.Clean(file.Name),
			FileName:  path.Base(file.Name),
			ImageExt:  strings.ToLower(path.Ext(file.Name)),
			SizeBytes: int64(file.UncompressedSize64),
		})
	}
	sort.SliceStable(pages, func(i, j int) bool {
		return naturalPathLess(pages[i].EntryPath, pages[j].EntryPath)
	})
	for i := range pages {
		pages[i].Index = i
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrArchiveEmpty, archivePath)
	}
	return pages, nil
}

// OpenPage 按清理后的 ZIP 条目名打开一页图片，不再先枚举全书。
func OpenPage(ctx context.Context, archivePath string, entryPath string) (io.ReadCloser, PageEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, PageEntry{}, err
	}
	if !IsSupportedArchivePath(archivePath) {
		return nil, PageEntry{}, fmt.Errorf("%w: %s", ErrUnsupportedArchive, archivePath)
	}
	wanted := path.Clean(strings.TrimSpace(entryPath))
	if wanted == "" || wanted == "." || !IsSupportedImageEntry(wanted) {
		return nil, PageEntry{}, fmt.Errorf("%w: %s", ErrPageNotFound, entryPath)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, PageEntry{}, err
	}
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			_ = reader.Close()
			return nil, PageEntry{}, err
		}
		if file.FileInfo().IsDir() || !IsSupportedImageEntry(file.Name) {
			continue
		}
		if path.Clean(file.Name) != wanted {
			continue
		}
		body, err := file.Open()
		if err != nil {
			_ = reader.Close()
			return nil, PageEntry{}, err
		}
		selected := PageEntry{
			EntryPath: wanted,
			FileName:  path.Base(file.Name),
			ImageExt:  strings.ToLower(path.Ext(file.Name)),
			SizeBytes: int64(file.UncompressedSize64),
		}
		return &archivePageReadCloser{ReadCloser: body, archive: reader}, selected, nil
	}
	_ = reader.Close()
	return nil, PageEntry{}, fmt.Errorf("%w: %s", ErrPageNotFound, entryPath)
}

type archivePageReadCloser struct {
	io.ReadCloser
	archive *zip.ReadCloser
}

func (r *archivePageReadCloser) Close() error {
	bodyErr := r.ReadCloser.Close()
	archiveErr := r.archive.Close()
	if bodyErr != nil {
		return bodyErr
	}
	return archiveErr
}

func naturalPathLess(a, b string) bool {
	aa := strings.Split(path.Clean(a), "/")
	bb := strings.Split(path.Clean(b), "/")
	for i := 0; i < len(aa) && i < len(bb); i++ {
		cmp := naturalSegmentCompare(aa[i], bb[i])
		if cmp != 0 {
			return cmp < 0
		}
	}
	return len(aa) < len(bb)
}

func naturalSegmentCompare(a, b string) int {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ra, rb := a[ia], b[ib]
		if isDigit(ra) && isDigit(rb) {
			na, enda := readNumberRun(a, ia)
			nb, endb := readNumberRun(b, ib)
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			ia, ib = enda, endb
			continue
		}
		la := strings.ToLower(string(ra))
		lb := strings.ToLower(string(rb))
		if la < lb {
			return -1
		}
		if la > lb {
			return 1
		}
		ia++
		ib++
	}
	if ia == len(a) && ib == len(b) {
		return 0
	}
	if ia == len(a) {
		return -1
	}
	return 1
}

func readNumberRun(s string, start int) (int64, int) {
	end := start
	for end < len(s) && isDigit(s[end]) {
		end++
	}
	raw := strings.TrimLeft(s[start:end], "0")
	if raw == "" {
		raw = "0"
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, end
	}
	return n, end
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
