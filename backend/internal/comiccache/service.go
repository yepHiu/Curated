package comiccache

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"curated-backend/internal/comicarchive"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const thumbnailMaxSide = 420
const maxSourceBytes = 32 << 20
const maxPixels = 24_000_000

var (
	ErrSourceTooLarge           = errors.New("comic thumbnail source exceeds limit")
	ErrSourceDimensionsTooLarge = errors.New("comic thumbnail dimensions exceed limit")
)

type Store interface {
	SaveComicCacheEntry(context.Context, storage.ComicCacheEntryInput) error
	ListComicCacheEntries(context.Context, int) ([]contracts.ComicCacheEntryDTO, error)
	TouchComicCacheEntry(context.Context, string) error
	DeleteComicCacheEntry(context.Context, string) error
}

type Service struct {
	root     string
	maxBytes int64
	store    Store
}

type File struct {
	Path        string
	ContentType string
	SizeBytes   int64
}

// NewService 创建漫画缩略图磁盘缓存；maxBytes < 0 表示不限制占用。
func NewService(root string, maxBytes int64, store Store) *Service {
	root = strings.TrimSpace(root)
	if root == "" {
		root = filepath.Join(os.TempDir(), "curated-comic-cache")
	}
	return &Service{root: filepath.Clean(root), maxBytes: maxBytes, store: store}
}

// GetOrCreateThumbnail 命中磁盘 JPEG 则刷新访问时间；未命中则解码并在超过 maxBytes 时淘汰最久未访问条目。
func (s *Service) GetOrCreateThumbnail(ctx context.Context, comic contracts.ComicBookDetailDTO, page contracts.ComicPageDTO) (File, error) {
	if s == nil || s.store == nil {
		return File{}, errors.New("comic cache service not configured")
	}
	cacheKey := thumbnailCacheKey(comic.ID, page.Index, page.EntryPath)
	targetPath := filepath.Join(s.root, "thumbnails", safeFileSegment(comic.ID), fmt.Sprintf("%06d.jpg", page.Index))

	if info, err := os.Stat(targetPath); err == nil && !info.IsDir() {
		if err := s.store.TouchComicCacheEntry(ctx, cacheKey); err != nil {
			return File{}, err
		}
		return File{Path: targetPath, ContentType: "image/jpeg", SizeBytes: info.Size()}, nil
	}

	body, _, err := comicarchive.OpenPage(ctx, comic.Location, page.EntryPath)
	if err != nil {
		return File{}, err
	}
	defer body.Close()

	img, err := decodeThumbnailSource(body)
	if err != nil {
		return File{}, err
	}
	thumb := resizeForThumbnail(img)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return File{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".thumb-*.jpg")
	if err != nil {
		return File{}, err
	}
	tmpPath := tmp.Name()
	encodeErr := jpeg.Encode(tmp, thumb, &jpeg.Options{Quality: 82})
	closeErr := tmp.Close()
	if encodeErr != nil {
		_ = os.Remove(tmpPath)
		return File{}, encodeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return File{}, closeErr
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return File{}, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return File{}, err
	}
	if err := s.store.SaveComicCacheEntry(ctx, storage.ComicCacheEntryInput{
		CacheKey:  cacheKey,
		ComicID:   comic.ID,
		Kind:      "thumbnail",
		PageIndex: page.Index,
		Path:      targetPath,
		SizeBytes: info.Size(),
	}); err != nil {
		return File{}, err
	}
	if err := s.evictIfOverBudget(ctx, cacheKey); err != nil {
		return File{}, err
	}
	return File{Path: targetPath, ContentType: "image/jpeg", SizeBytes: info.Size()}, nil
}

func (s *Service) Status(ctx context.Context) (contracts.ComicCacheStatusDTO, error) {
	entries, err := s.store.ListComicCacheEntries(ctx, 10000)
	if err != nil {
		return contracts.ComicCacheStatusDTO{}, err
	}
	var used int64
	for _, entry := range entries {
		used += entry.SizeBytes
	}
	return contracts.ComicCacheStatusDTO{
		MaxBytes:   s.maxBytes,
		UsedBytes:  used,
		EntryCount: len(entries),
	}, nil
}

func (s *Service) Cleanup(ctx context.Context) (contracts.ComicCacheStatusDTO, error) {
	entries, err := s.store.ListComicCacheEntries(ctx, 10000)
	if err != nil {
		return contracts.ComicCacheStatusDTO{}, err
	}
	for _, entry := range entries {
		if s.isInsideRoot(entry.Path) {
			if err := os.Remove(entry.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return contracts.ComicCacheStatusDTO{}, err
			}
		}
		if err := s.store.DeleteComicCacheEntry(ctx, entry.CacheKey); err != nil {
			return contracts.ComicCacheStatusDTO{}, err
		}
	}
	return s.Status(ctx)
}

// evictIfOverBudget 按 last_accessed_at 从旧到新删除缓存，直到占用不超过 maxBytes；跳过刚写入的 keepKey。
func (s *Service) evictIfOverBudget(ctx context.Context, keepKey string) error {
	if s.maxBytes < 0 {
		return nil
	}
	entries, err := s.store.ListComicCacheEntries(ctx, 10000)
	if err != nil {
		return err
	}
	var used int64
	for _, entry := range entries {
		used += entry.SizeBytes
	}
	for _, entry := range entries {
		if used <= s.maxBytes {
			return nil
		}
		if entry.CacheKey == keepKey {
			continue
		}
		if s.isInsideRoot(entry.Path) {
			if err := os.Remove(entry.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if err := s.store.DeleteComicCacheEntry(ctx, entry.CacheKey); err != nil {
			return err
		}
		used -= entry.SizeBytes
	}
	return nil
}

// decodeThumbnailSource 限制源字节和像素，过大拒绝而不回退原图。
func decodeThumbnailSource(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSourceBytes {
		return nil, ErrSourceTooLarge
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxPixels {
		return nil, ErrSourceDimensionsTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (s *Service) isInsideRoot(path string) bool {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return false
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func resizeForThumbnail(src image.Image) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	if width <= thumbnailMaxSide && height <= thumbnailMaxSide {
		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
		return dst
	}
	scale := float64(thumbnailMaxSide) / float64(width)
	if height > width {
		scale = float64(thumbnailMaxSide) / float64(height)
	}
	dstW := max(1, int(float64(width)*scale))
	dstH := max(1, int(float64(height)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func thumbnailCacheKey(comicID string, pageIndex int, entryPath string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d:%s", comicID, pageIndex, entryPath)))
	return "comic-thumbnail-" + hex.EncodeToString(sum[:])
}

func safeFileSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
