package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
)

// TestComicCommentHandlersPersistAndRejectInvalid 验证漫画备注 HTTP 读写、过长、缺书和 Beta 关闭。
func TestComicCommentHandlersPersistAndRejectInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := newImportTestStore(t, root)
	detail, _ := indexComicFixture(t, store, root)
	cfg := config.Default()
	cfg.CacheDir = filepath.Join(root, "cache")
	h := NewHandler(Deps{
		Cfg:              cfg,
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: true},
	})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	url := srv.URL + "/api/library/comics/books/" + detail.ID + "/comment"

	getResp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("empty get status = %d", getResp.StatusCode)
	}
	var empty contracts.ComicCommentDTO
	if err := json.NewDecoder(getResp.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if empty.Body != "" {
		t.Fatalf("expected empty body, got %+v", empty)
	}

	putReq, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(`{"body":"  comic note  "}`))
	if err != nil {
		t.Fatal(err)
	}
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put status = %d", putResp.StatusCode)
	}
	var saved contracts.ComicCommentDTO
	if err := json.NewDecoder(putResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Body != "comic note" || saved.UpdatedAt == "" {
		t.Fatalf("saved = %+v", saved)
	}

	reload, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer reload.Body.Close()
	var got contracts.ComicCommentDTO
	if err := json.NewDecoder(reload.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Body != "comic note" {
		t.Fatalf("reload = %+v", got)
	}

	longBody, _ := json.Marshal(map[string]string{"body": strings.Repeat("あ", contracts.MaxBookCommentRunes+1)})
	tooLong, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(longBody))
	if err != nil {
		t.Fatal(err)
	}
	tooLongResp, err := http.DefaultClient.Do(tooLong)
	if err != nil {
		t.Fatal(err)
	}
	defer tooLongResp.Body.Close()
	if tooLongResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("too long status = %d", tooLongResp.StatusCode)
	}

	missing, err := http.NewRequest(http.MethodPut, srv.URL+"/api/library/comics/books/missing/comment", bytes.NewBufferString(`{"body":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	missingResp, err := http.DefaultClient.Do(missing)
	if err != nil {
		t.Fatal(err)
	}
	defer missingResp.Body.Close()
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d", missingResp.StatusCode)
	}

	disabled := NewHandler(Deps{
		Cfg:              config.Default(),
		Logger:           zap.NewNop(),
		Store:            store,
		ComicSettingsCtl: &stubComicSettingsCtl{enabled: false},
	})
	disabledSrv := httptest.NewServer(disabled.Routes())
	t.Cleanup(disabledSrv.Close)
	disabledResp, err := http.Get(disabledSrv.URL + "/api/library/comics/books/" + detail.ID + "/comment")
	if err != nil {
		t.Fatal(err)
	}
	defer disabledResp.Body.Close()
	if disabledResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("disabled status = %d", disabledResp.StatusCode)
	}
	var appErr contracts.AppError
	if err := json.NewDecoder(disabledResp.Body).Decode(&appErr); err != nil {
		t.Fatal(err)
	}
	if appErr.Code != contracts.ErrorCodeComicLibraryDisabled {
		t.Fatalf("disabled code = %q", appErr.Code)
	}
}
