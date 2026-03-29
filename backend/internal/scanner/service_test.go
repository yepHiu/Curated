package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
)

func TestScan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "ABC-123.mp4"))
	mustWriteFile(t, filepath.Join(root, "ignored.txt"))
	mustWriteFile(t, filepath.Join(root, "clips", "fc2ppv123456.mkv"))
	mustWriteFile(t, filepath.Join(root, "clips", "holiday.mov"))

	service := NewService(zap.NewNop())

	recognized := 0
	skipped := 0
	progressCalls := 0

	summary, err := service.Scan(context.Background(), "task-1", []string{root}, Hooks{
		OnProgress: func(processed, total int, message string) {
			progressCalls++
			if total != 3 {
				t.Fatalf("expected total 3 files, got %d", total)
			}
			if processed < 0 || processed > total {
				t.Fatalf("invalid processed count %d of %d", processed, total)
			}
			if message == "" {
				t.Fatal("expected non-empty progress message")
			}
		},
		OnFileDetected: func(result contracts.ScanFileResultDTO) error {
			if result.Number == "" {
				skipped++
				if result.Reason == "" {
					t.Fatal("expected skip reason")
				}
				return nil
			}

			recognized++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}

	if summary.FilesDiscovered != 3 {
		t.Fatalf("expected 3 discovered files, got %d", summary.FilesDiscovered)
	}
	if summary.FilesSkipped != 1 {
		t.Fatalf("expected 1 skipped file, got %d", summary.FilesSkipped)
	}
	if summary.RecognizedNumber != 2 {
		t.Fatalf("expected 2 recognized numbers, got %d", summary.RecognizedNumber)
	}
	if recognized != 2 || skipped != 1 {
		t.Fatalf("unexpected hook counts: recognized=%d skipped=%d", recognized, skipped)
	}
	if progressCalls == 0 {
		t.Fatal("expected progress hooks to be called")
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
