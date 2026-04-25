package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/config"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func newCuratedFramesP1Server(t *testing.T) (*storage.SQLiteStore, *httptest.Server) {
	t.Helper()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "curated-p1.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHandler(Deps{
		Cfg:    config.Config{},
		Logger: zap.NewNop(),
		Store:  store,
	}).Routes())
	t.Cleanup(srv.Close)
	return store, srv
}

func addMovieForCuratedFramesP1Test(t *testing.T, store *storage.SQLiteStore, code string) string {
	t.Helper()
	root := t.TempDir()
	videoPath := filepath.Join(root, code+".mp4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	outcome, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "t-" + code,
		Path:     videoPath,
		FileName: code + ".mp4",
		Number:   code,
	})
	if err != nil {
		t.Fatal(err)
	}
	return outcome.MovieID
}

func makeTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8((x * 3) % 255), G: uint8((y * 5) % 255), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHandleCuratedFramesQueryStatsAndFacets(t *testing.T) {
	t.Parallel()
	store, srv := newCuratedFramesP1Server(t)
	ctx := context.Background()
	movie1 := addMovieForCuratedFramesP1Test(t, store, "CFP1-001")
	movie2 := addMovieForCuratedFramesP1Test(t, store, "CFP1-002")
	if err := store.InsertCuratedFrame(ctx, storage.CuratedFrameMeta{
		ID: "frame-1", MovieID: movie1, Title: "Alpha Scene", Code: "CFP1-001", Actors: []string{"Mina"}, PositionSec: 12, CapturedAt: "2026-04-11T03:00:00Z", Tags: []string{"closeup"},
	}, []byte("png-1")); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertCuratedFrame(ctx, storage.CuratedFrameMeta{
		ID: "frame-2", MovieID: movie2, Title: "Alpha Scene 2", Code: "CFP1-002", Actors: []string{"Mina", "Airi"}, PositionSec: 24, CapturedAt: "2026-04-11T02:00:00Z", Tags: []string{"closeup", "favorite"},
	}, []byte("png-2")); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertCuratedFrame(ctx, storage.CuratedFrameMeta{
		ID: "frame-3", MovieID: movie1, Title: "Gamma", Code: "ZZZ-003", Actors: []string{"Rin"}, PositionSec: 36, CapturedAt: "2026-04-11T01:00:00Z", Tags: []string{"wide"},
	}, []byte("png-3")); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(srv.URL + "/api/curated-frames?q=alpha&tag=closeup&limit=1&offset=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var listBody struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if listBody.Total != 2 || listBody.Limit != 1 || listBody.Offset != 1 {
		t.Fatalf("list page = %+v", listBody)
	}
	if len(listBody.Items) != 1 || listBody.Items[0].ID != "frame-2" {
		t.Fatalf("list items = %+v", listBody.Items)
	}

	statsResp, err := http.Get(srv.URL + "/api/curated-frames/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer statsResp.Body.Close()
	var statsBody struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(statsResp.Body).Decode(&statsBody); err != nil {
		t.Fatal(err)
	}
	if statsBody.Total != 3 {
		t.Fatalf("stats total = %d, want 3", statsBody.Total)
	}

	actorsResp, err := http.Get(srv.URL + "/api/curated-frames/actors")
	if err != nil {
		t.Fatal(err)
	}
	defer actorsResp.Body.Close()
	var actorsBody struct {
		Items []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(actorsResp.Body).Decode(&actorsBody); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actorsBody.Items, []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}{
		{Name: "Mina", Count: 2},
		{Name: "Airi", Count: 1},
		{Name: "Rin", Count: 1},
	}) {
		t.Fatalf("actors = %+v", actorsBody.Items)
	}

	tagsResp, err := http.Get(srv.URL + "/api/curated-frames/tags")
	if err != nil {
		t.Fatal(err)
	}
	defer tagsResp.Body.Close()
	var tagsBody struct {
		Items []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(tagsResp.Body).Decode(&tagsBody); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tagsBody.Items, []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}{
		{Name: "closeup", Count: 2},
		{Name: "favorite", Count: 1},
		{Name: "wide", Count: 1},
	}) {
		t.Fatalf("tags = %+v", tagsBody.Items)
	}
}

func TestHandlePostCuratedFrameMultipartAndThumbnail(t *testing.T) {
	t.Parallel()
	store, srv := newCuratedFramesP1Server(t)
	movieID := addMovieForCuratedFramesP1Test(t, store, "CFP1-UP")

	meta := map[string]any{
		"id":          "frame-uploaded",
		"movieId":     movieID,
		"title":       "Uploaded Frame",
		"code":        "CFP1-UP",
		"actors":      []string{"Mina"},
		"positionSec": 42.5,
		"capturedAt":  "2026-04-11T10:00:00Z",
		"tags":        []string{"favorite"},
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("metadata", string(metaJSON)); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("image", "frame.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(makeTestPNG(t, 320, 180)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/curated-frames", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("create status = %d, want 204", resp.StatusCode)
	}

	thumbResp, err := http.Get(srv.URL + "/api/curated-frames/frame-uploaded/thumbnail")
	if err != nil {
		t.Fatal(err)
	}
	defer thumbResp.Body.Close()
	if thumbResp.StatusCode != http.StatusOK {
		t.Fatalf("thumbnail status = %d, want 200", thumbResp.StatusCode)
	}
	if got := thumbResp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("thumbnail content-type = %q, want image/png", got)
	}
	data, err := io.ReadAll(thumbResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected thumbnail bytes")
	}
}

func TestHandleCuratedFrameImageAndThumbnailDetectJPEGContentType(t *testing.T) {
	t.Parallel()
	store, srv := newCuratedFramesP1Server(t)
	ctx := context.Background()
	movieID := addMovieForCuratedFramesP1Test(t, store, "CFP1-JPEG")
	jpegBlob := makeTestJPEG(t, 120, 80)
	if err := store.InsertCuratedFrame(ctx, storage.CuratedFrameMeta{
		ID:          "frame-jpeg",
		MovieID:     movieID,
		Title:       "JPEG Frame",
		Code:        "CFP1-JPEG",
		Actors:      []string{"Mina"},
		PositionSec: 18,
		CapturedAt:  "2026-04-11T11:00:00Z",
		Tags:        []string{"jpeg"},
	}, jpegBlob); err != nil {
		t.Fatal(err)
	}

	imageResp, err := http.Get(srv.URL + "/api/curated-frames/frame-jpeg/image")
	if err != nil {
		t.Fatal(err)
	}
	defer imageResp.Body.Close()
	if imageResp.StatusCode != http.StatusOK {
		t.Fatalf("image status = %d, want 200", imageResp.StatusCode)
	}
	if got := imageResp.Header.Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("image content-type = %q, want image/jpeg", got)
	}

	thumbResp, err := http.Get(srv.URL + "/api/curated-frames/frame-jpeg/thumbnail")
	if err != nil {
		t.Fatal(err)
	}
	defer thumbResp.Body.Close()
	if thumbResp.StatusCode != http.StatusOK {
		t.Fatalf("thumbnail status = %d, want 200", thumbResp.StatusCode)
	}
	if got := thumbResp.Header.Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("thumbnail content-type = %q, want image/jpeg", got)
	}
}
