package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

type metadataCompleter func(context.Context, []llm.ChatMessage, int) (string, error)

func (f metadataCompleter) Complete(ctx context.Context, messages []llm.ChatMessage, tokens int) (string, error) {
	return f(ctx, messages, tokens)
}

func TestBookMetadataPreviewApplyConflictAndReceipt(t *testing.T) {
	for _, kind := range []string{"comic", "photo"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			dbPath := filepath.Join(t.TempDir(), "metadata.db")
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			if err = store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			a := &App{store: store, cfg: enabledAITestConfig(), comicLibraryEnabled: true, photoLibraryEnabled: true}
			source := "作者:山田 花子 夏の記録 02"
			metadata := contracts.BookMetadata{CleanTitle: "夏の記録 02", TranslatedTitle: "夏日记录 02", Authors: []contracts.BookCredit{{Name: "山田 花子", Evidence: "作者:山田 花子"}}}
			if kind == "photo" {
				source = "モデル:山田 花子 夏の記録 02"
				metadata.Models = []contracts.BookCredit{{Name: "山田 花子", Evidence: "モデル:山田 花子"}}
				metadata.Authors = nil
			}
			location := filepath.Join(t.TempDir(), "private", "sample.zip")
			var id string
			upsert := func() {
				if kind == "comic" {
					b, e := store.UpsertComicBook(ctx, storage.ComicBookUpsert{Title: source, SourceFileName: source + ".zip", Location: location, PageCount: 1})
					id, err = b.ID, e
				} else {
					b, e := store.UpsertPhotoBook(ctx, storage.PhotoBookUpsert{Title: source, SourceFileName: source + ".zip", Location: location, PageCount: 1})
					id, err = b.ID, e
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			upsert()
			req := contracts.AIActionRequest{TargetLocale: "zh-CN"}
			if kind == "comic" {
				req.ComicID = id
			} else {
				req.PhotoID = id
			}
			encoded, _ := json.Marshal(metadata)
			model := metadataCompleter(func(_ context.Context, messages []llm.ChatMessage, maxTokens int) (string, error) {
				if maxTokens != 0 {
					t.Fatalf("book recognition must not cap reasoning and JSON output: %d", maxTokens)
				}
				if len(messages) != 2 || strings.Contains(messages[1].Content, "private") || !strings.Contains(messages[1].Content, source) {
					t.Fatalf("unexpected model input: %+v", messages)
				}
				return string(encoded), nil
			})
			preview, err := a.runBookMetadataAction(ctx, model, req)
			if err != nil || preview.ConfirmToken == "" {
				t.Fatalf("preview: %+v %v", preview, err)
			}
			before, err := a.GetBookMetadataState(ctx, kind, id)
			if err != nil || before.Title != source || before.Metadata.CleanTitle != "" {
				t.Fatalf("preview wrote data: %+v %v", before, err)
			}
			apply := contracts.AIToolApplyRequest{Name: preview.Name, SessionID: preview.SessionID, ConfirmToken: preview.ConfirmToken, Arguments: preview.Arguments}
			if _, err = a.ApplyAITool(ctx, apply); err != nil {
				t.Fatal(err)
			}
			upsert() // A rescan must preserve confirmed metadata and the title overlay.
			after, err := a.GetBookMetadataState(ctx, kind, id)
			if err != nil || after.Title != metadata.TranslatedTitle || after.Metadata.CleanTitle != metadata.CleanTitle || after.OriginalTitle != source {
				t.Fatalf("rescan lost data: %+v %v", after, err)
			}
			if kind == "comic" {
				page, e := store.ListComicBooks(ctx, contracts.ListComicBooksRequest{Query: "山田 花子"})
				if e != nil || page.Total != 1 || len(page.Items[0].Metadata.Authors) != 1 {
					t.Fatalf("search: %+v %v", page, e)
				}
			} else {
				page, e := store.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Query: "夏日记录"})
				if e != nil || page.Total != 1 || len(page.Items[0].Metadata.Models) != 1 {
					t.Fatalf("search: %+v %v", page, e)
				}
			}
			// A later edit must survive a replay, including a fresh gateway after restart.
			manual := "人工标题"
			if kind == "comic" {
				_, err = store.PatchComicBook(ctx, id, contracts.PatchComicBookRequest{Title: &manual})
			} else {
				_, err = store.PatchPhotoBook(ctx, id, contracts.PatchPhotoBookRequest{Title: &manual})
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = storage.NewSQLiteStore(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			a = &App{store: store, cfg: enabledAITestConfig(), comicLibraryEnabled: true, photoLibraryEnabled: true}
			replayed, err := a.ApplyAITool(ctx, apply)
			if err != nil || !replayed.Replayed {
				t.Fatalf("replay: %+v %v", replayed, err)
			}
			after, _ = a.GetBookMetadataState(ctx, kind, id)
			if after.Title != manual {
				t.Fatal("replay overwrote manual edit")
			}
			preview, err = a.runBookMetadataAction(ctx, model, req)
			if err != nil {
				t.Fatal(err)
			}
			source += " 改訂版"
			upsert()
			_, err = a.ApplyAITool(ctx, contracts.AIToolApplyRequest{Name: preview.Name, SessionID: preview.SessionID, ConfirmToken: preview.ConfirmToken, Arguments: preview.Arguments})
			toolErr, ok := err.(*core.ToolError)
			if !ok || toolErr.Code != "AI_WRITE_CONFLICT" {
				t.Fatalf("source conflict not rejected: %v", err)
			}
		})
	}
}

func TestBookMetadataRejectsUnfoundedCredits(t *testing.T) {
	source := contracts.BookMetadataState{OriginalTitle: "作者:山田 花子 夏の記録", SourceFileName: "sample.zip"}
	valid := contracts.BookMetadata{CleanTitle: "夏の記録", TranslatedTitle: "夏日记录", Authors: []contracts.BookCredit{{Name: "山田 花子", Evidence: "作者:山田 花子"}}}
	if err := contracts.ValidateBookMetadata("comic", valid, source); err != nil {
		t.Fatal(err)
	}
	for _, credit := range []contracts.BookCredit{{Name: "不存在", Evidence: "不存在"}, {Name: "另一人", Evidence: "作者:山田 花子"}, {Name: "山田 花子", Evidence: ""}} {
		invalid := valid
		invalid.Authors = []contracts.BookCredit{credit}
		if contracts.ValidateBookMetadata("comic", invalid, source) == nil {
			t.Fatalf("accepted unsupported credit: %+v", credit)
		}
	}
	if contracts.ValidateBookMetadata("photo", valid, source) == nil {
		t.Fatal("accepted comic author as photo role")
	}
}

func TestBookMetadataCorrectsInvalidOutputOnce(t *testing.T) {
	valid := `{"cleanTitle":"夏の記録","translatedTitle":"夏日记录","models":[{"name":"花子","evidence":"モデル:花子"}]}`
	wrongRole := `{"cleanTitle":"夏の記録","translatedTitle":"夏日记录","authors":[{"name":"花子","evidence":"モデル:花子"}]}`
	for _, tc := range []struct {
		name, first, second, code string
		providerErr               error
		calls                     int
	}{
		{name: "photo role correction", first: wrongRole, second: valid, calls: 2},
		{name: "JSON correction", first: "```json\n" + valid + "\n```", second: valid, calls: 2},
		{name: "invalid roles remain rejected", first: wrongRole, second: wrongRole, code: "BOOK_METADATA_INVALID_RESULT", calls: 2},
		{name: "invented evidence remains rejected", first: wrongRole, second: `{"cleanTitle":"夏","translatedTitle":"夏","models":[{"name":"Alice","evidence":"Alice"}]}`, code: "BOOK_METADATA_INVALID_RESULT", calls: 2},
		{name: "provider failure is not corrected", providerErr: errors.New("connection failed"), code: "BOOK_METADATA_PROVIDER_FAILED", calls: 1},
		{name: "truncated response", providerErr: llm.ErrOutputTruncated, code: "BOOK_METADATA_TRUNCATED", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "book.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err = store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			a := &App{store: store, cfg: enabledAITestConfig(), photoLibraryEnabled: true}
			book, err := store.UpsertPhotoBook(ctx, storage.PhotoBookUpsert{Title: "モデル:花子 夏の記録", SourceFileName: "sample.zip", Location: filepath.Join(t.TempDir(), "sample.zip"), PageCount: 1})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.CreateBookOrganization(ctx, "job", contracts.BookOrganizationRequest{Kind: "photo", BookIDs: []string{book.ID}, Locale: "zh-CN", RequestID: "request"}); err != nil {
				t.Fatal(err)
			}
			calls := 0
			model := metadataCompleter(func(_ context.Context, messages []llm.ChatMessage, maxTokens int) (string, error) {
				if maxTokens != 0 {
					t.Fatalf("recognition and correction must omit the output token ceiling: %d", maxTokens)
				}
				calls++
				if calls == 1 {
					return tc.first, tc.providerErr
				}
				if calls > 2 || len(messages) != 4 || messages[2].Content != tc.first || !strings.Contains(messages[3].Content, "failed validation") {
					t.Fatalf("unexpected correction: %+v", messages)
				}
				return tc.second, nil
			})
			a.processBookOrganization(ctx, "job", func(ctx context.Context, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
				return a.runBookMetadataAction(ctx, model, req)
			})
			if calls != tc.calls {
				t.Fatalf("calls = %d", calls)
			}
			item, err := store.BookOrganizationItem(ctx, "job", book.ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.code == "" {
				if item.Status != "ready" || len(item.Metadata.Models) != 1 || len(item.Metadata.Authors) != 0 {
					t.Fatalf("corrected proposal: %+v", item)
				}
			} else if item.Status != "failed" || item.Reason != tc.code {
				t.Fatalf("wrong error classification: %+v", item)
			}
			state, err := a.GetBookMetadataState(ctx, "photo", book.ID)
			if err != nil || state.Title != book.Title || state.Metadata.CleanTitle != "" {
				t.Fatalf("recognition wrote metadata: %+v %v", state, err)
			}
		})
	}
}
