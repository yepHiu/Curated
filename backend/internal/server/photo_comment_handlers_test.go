package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
	"curated-backend/internal/tasks"
)

// TestPhotoCommentHandlersPersistAndRejectInvalid 验证写真备注 HTTP 读写、过长、缺书和 Beta 关闭。
func TestPhotoCommentHandlersPersistAndRejectInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	book, err := store.UpsertPhotoBook(context.Background(), storage.PhotoBookUpsert{
		Location:       filepath.Join(root, "photo.cbz"),
		Title:          "Photo",
		SourceFileName: "photo.cbz",
		FileModifiedAt: "2026-09-12T00:00:00Z",
		PageCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{
		Cfg:    config.Config{PhotoLibraryEnabled: true},
		Store:  store,
		Logger: zap.NewNop(),
		Tasks:  tasks.NewManager(),
	})
	url := "http://127.0.0.1/api/library/photos/books/" + book.ID + "/comment"
	request := func(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
		// 构造同步 HTTP 请求，复用同一 handler 覆盖读写与拒绝路径。
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}

	empty := request(h.Routes(), http.MethodGet, url, "")
	if empty.Code != http.StatusOK {
		t.Fatalf("empty get status = %d body=%s", empty.Code, empty.Body.String())
	}

	saved := request(h.Routes(), http.MethodPut, url, `{"body":"  photo note  "}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", saved.Code, saved.Body.String())
	}
	var dto contracts.PhotoCommentDTO
	if err := json.Unmarshal(saved.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if dto.Body != "photo note" || dto.UpdatedAt == "" {
		t.Fatalf("saved = %+v", dto)
	}

	reload := request(h.Routes(), http.MethodGet, url, "")
	if err := json.Unmarshal(reload.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if dto.Body != "photo note" {
		t.Fatalf("reload = %+v", dto)
	}

	longBody, _ := json.Marshal(map[string]string{"body": strings.Repeat("あ", contracts.MaxBookCommentRunes+1)})
	if w := request(h.Routes(), http.MethodPut, url, string(longBody)); w.Code != http.StatusBadRequest {
		t.Fatalf("too long status = %d", w.Code)
	}
	if w := request(h.Routes(), http.MethodPut, "http://127.0.0.1/api/library/photos/books/missing/comment", `{"body":"x"}`); w.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", w.Code)
	}

	disabled := NewHandler(Deps{
		Cfg:    config.Config{},
		Store:  store,
		Logger: zap.NewNop(),
		Tasks:  tasks.NewManager(),
	})
	disabledResp := request(disabled.Routes(), http.MethodGet, url, "")
	if disabledResp.Code != http.StatusBadRequest || !bytes.Contains(disabledResp.Body.Bytes(), []byte("PHOTO_LIBRARY_DISABLED")) {
		t.Fatalf("disabled = %d %s", disabledResp.Code, disabledResp.Body.String())
	}
}
