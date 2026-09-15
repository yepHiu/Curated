package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

type stubBookQuery struct {
	comicEnabled bool
	photoEnabled bool
	comics       contracts.ComicBooksPageDTO
	photos       contracts.PhotoBooksPageDTO
	comic        contracts.ComicBookDetailDTO
	photo        contracts.PhotoBookDetailDTO
	comicNote    contracts.ComicCommentDTO
	photoNote    contracts.PhotoCommentDTO
}

func (s stubBookQuery) ComicLibraryEnabled() bool { return s.comicEnabled }
func (s stubBookQuery) PhotoLibraryEnabled() bool { return s.photoEnabled }
func (s stubBookQuery) ListComicBooks(context.Context, contracts.ListComicBooksRequest) (contracts.ComicBooksPageDTO, error) {
	return s.comics, nil
}
func (s stubBookQuery) GetComicBookDetail(context.Context, string) (contracts.ComicBookDetailDTO, error) {
	return s.comic, nil
}
func (s stubBookQuery) GetComicComment(context.Context, string) (contracts.ComicCommentDTO, error) {
	return s.comicNote, nil
}
func (s stubBookQuery) ListPhotoBooks(context.Context, contracts.ListPhotoBooksRequest) (contracts.PhotoBooksPageDTO, error) {
	return s.photos, nil
}
func (s stubBookQuery) GetPhotoBookDetail(context.Context, string) (contracts.PhotoBookDetailDTO, error) {
	return s.photo, nil
}
func (s stubBookQuery) GetPhotoComment(context.Context, string) (contracts.PhotoCommentDTO, error) {
	return s.photoNote, nil
}

func TestSearchComicsOmitsLocationAndDisabledLibrary(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubBookQuery{
		comicEnabled: true,
		comics: contracts.ComicBooksPageDTO{
			Total: 1,
			Items: []contracts.ComicBookListItemDTO{{
				ID:       "comic-1",
				Title:    "Sample Comic",
				Tags:     []string{"tag"},
				CoverURL: "/api/library/comics/books/comic-1/pages/0/thumbnail",
				Location: "C:\\secret\\comic.zip",
			}},
		},
	}
	if err := RegisterBookQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	shown := gateway.Invoke(context.Background(), core.Call{
		Name: "search_comics",
		Args: json.RawMessage(`{"q":"Sample","limit":20}`),
	})
	if !shown.OK {
		t.Fatalf("search comics = %+v", shown)
	}
	raw, _ := json.Marshal(shown.Data)
	if !strings.Contains(string(raw), "comic-1") || strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "movieId") {
		t.Fatalf("unsafe comic projection: %s", raw)
	}

	closed := core.NewRegistry()
	if err := RegisterBookQueryTools(closed, stubBookQuery{}); err != nil {
		t.Fatal(err)
	}
	denied := core.NewGateway(closed, nil, nil, nil).Invoke(context.Background(), core.Call{
		Name: "search_comics",
		Args: json.RawMessage(`{"q":"Sample"}`),
	})
	if denied.OK || denied.Error == nil || denied.Error.Code != contracts.ErrorCodeComicLibraryDisabled {
		t.Fatalf("disabled comic library = %+v", denied)
	}
}

func TestPresentComicsRequiresThisTurnRead(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubBookQuery{
		comicEnabled: true,
		comics: contracts.ComicBooksPageDTO{
			Total: 1,
			Items: []contracts.ComicBookListItemDTO{{ID: "comic-1", Title: "Sample Comic", CoverURL: "/cover"}},
		},
	}
	if err := RegisterBookQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := RegisterBookPresentTools(reg, gateway.BookRefs()); err != nil {
		t.Fatal(err)
	}
	unknown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.PresentComicsName,
		Args:      json.RawMessage(`{"items":[{"comicId":"comic-1"}]}`),
		SessionID: "ses_book",
	})
	if unknown.OK {
		t.Fatalf("present before search should fail: %+v", unknown)
	}
	search := gateway.Invoke(context.Background(), core.Call{
		Name:      "search_comics",
		Args:      json.RawMessage(`{"q":"Sample"}`),
		SessionID: "ses_book",
	})
	if !search.OK {
		t.Fatalf("search = %+v", search)
	}
	gateway.RememberBookRefs("ses_book", core.ExtractBookRefs(search))
	shown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.PresentComicsName,
		Args:      json.RawMessage(`{"items":[{"comicId":"comic-1"}]}`),
		SessionID: "ses_book",
	})
	if !shown.OK {
		t.Fatalf("present after search = %+v", shown)
	}
	raw, _ := json.Marshal(shown.Data)
	if !strings.Contains(string(raw), `"kind":"comic"`) || strings.Contains(string(raw), "movieId") {
		t.Fatalf("book card projection: %s", raw)
	}
}

type stubBookWrite struct {
	comicEnabled bool
	photoEnabled bool
	comic        contracts.ComicBookDetailDTO
	photo        contracts.PhotoBookDetailDTO
	comicWrites  int
	photoWrites  int
}

func (s *stubBookWrite) ComicLibraryEnabled() bool { return s.comicEnabled }
func (s *stubBookWrite) PhotoLibraryEnabled() bool { return s.photoEnabled }
func (s *stubBookWrite) GetComicBookDetail(context.Context, string) (contracts.ComicBookDetailDTO, error) {
	return s.comic, nil
}
func (s *stubBookWrite) GetComicComment(context.Context, string) (contracts.ComicCommentDTO, error) {
	return contracts.ComicCommentDTO{}, nil
}
func (s *stubBookWrite) UpsertComicComment(context.Context, string, string, ...string) (contracts.ComicCommentDTO, error) {
	return contracts.ComicCommentDTO{}, nil
}
func (s *stubBookWrite) PatchComicBook(_ context.Context, _ string, patch contracts.PatchComicBookRequest) (contracts.ComicBookDetailDTO, error) {
	// 记录确认后的漫画标题写入次数。
	s.comicWrites++
	if patch.Title != nil {
		s.comic.Title = *patch.Title
	}
	return s.comic, nil
}
func (s *stubBookWrite) GetPhotoBookDetail(context.Context, string) (contracts.PhotoBookDetailDTO, error) {
	return s.photo, nil
}
func (s *stubBookWrite) GetPhotoComment(context.Context, string) (contracts.PhotoCommentDTO, error) {
	return contracts.PhotoCommentDTO{}, nil
}
func (s *stubBookWrite) UpsertPhotoComment(context.Context, string, string, ...string) (contracts.PhotoCommentDTO, error) {
	return contracts.PhotoCommentDTO{}, nil
}
func (s *stubBookWrite) PatchPhotoBook(_ context.Context, _ string, patch contracts.PatchPhotoBookRequest) (contracts.PhotoBookDetailDTO, error) {
	// 记录确认后的写真标题写入次数。
	s.photoWrites++
	if patch.Title != nil {
		s.photo.Title = *patch.Title
	}
	return s.photo, nil
}

// TestUpdateComicTitlePreviewDoesNotWrite 确认漫画标题写工具预览不落库，确认后才写入。
func TestUpdateComicTitlePreviewDoesNotWrite(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubBookWrite{
		comicEnabled: true,
		comic:        contracts.ComicBookDetailDTO{ComicBookListItemDTO: contracts.ComicBookListItemDTO{ID: "comic-1", Title: "旧标题"}},
	}
	if err := RegisterBookWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	args := json.RawMessage(`{"comicId":"comic-1","title":"展示标题"}`)
	preview := gw.Invoke(context.Background(), core.Call{
		Name: core.UpdateComicTitleName, Args: args, SessionID: "act_book", Channel: core.ChannelAction,
	})
	if !preview.OK || preview.ConfirmToken == "" || write.comicWrites != 0 {
		t.Fatalf("preview = %+v writes=%d", preview, write.comicWrites)
	}
	applied := gw.Invoke(context.Background(), core.Call{
		Name: core.UpdateComicTitleName, Args: args, SessionID: "act_book", Channel: core.ChannelAction,
		ConfirmTok: preview.ConfirmToken,
	})
	if !applied.OK || write.comicWrites != 1 || write.comic.Title != "展示标题" {
		t.Fatalf("apply = %+v writes=%d title=%q", applied, write.comicWrites, write.comic.Title)
	}
}

