package subtitles

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

type testTransport func(*http.Request) (*http.Response, error)

func (t testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return t(r) }
func TestSignedTranscriptRequestHasNoCredentialOrContentType(t *testing.T) {
	client := &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"output":{"task_status":"SUCCEEDED","results":[{"subtask_status":"SUCCEEDED","transcription_url":"https://results.test.aliyuncs.com/transcript?signature=private"}]}}`
		if r.URL.Host == "results.test.aliyuncs.com" {
			if r.Header.Get("Authorization") != "" || r.Header.Get("Content-Type") != "" {
				t.Error("signed GET signature invalidated by added headers")
			}
			body = `{"transcripts":[{"sentences":[{"begin_time":0,"end_time":1000,"text":"fixture"}]}]}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	p := Bailian{Client: client}
	cues, done, e := p.Result(context.Background(), "secret", "known-task")
	if e != nil || !done || len(cues) != 1 {
		t.Fatalf("cues=%d done=%v err=%v", len(cues), done, e)
	}
}

type fakeProvider struct {
	calls    int
	uploaded func(string)
}

func (p *fakeProvider) Upload(_ context.Context, _ string, path string, _ ...string) (string, error) {
	p.calls++
	if p.uploaded != nil {
		p.uploaded(path)
	}
	return "oss://fixture", nil
}
func (p *fakeProvider) Submit(context.Context, string, string, ...string) (string, error) {
	p.calls++
	return "fixture-task", nil
}
func (p *fakeProvider) Result(context.Context, string, string) ([]Cue, bool, error) {
	p.calls++
	return []Cue{{Begin: 100, End: 900, Text: "こんにちは"}}, true, nil
}
func fixture(t *testing.T) (Options, storage.SubtitleRecord) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	store, e := storage.NewSQLiteStore(filepath.Join(root, "test.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = store.Close() })
	if e = store.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(root, "DEMO-001.CD1.mp4")
	if e = os.WriteFile(path, []byte("fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	movie, e := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Number: "DEMO-001", Path: path, FileName: filepath.Base(path)})
	if e != nil {
		t.Fatal(e)
	}
	detail, e := store.GetMoviePlaybackDetail(ctx, movie.MovieID)
	if e != nil {
		t.Fatal(e)
	}
	rev, _ := revision(path)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	r := storage.SubtitleRecord{RequestID: "request-0001", SourcePath: path, Revision: rev, TextBaseURL: "http://model/v1", TextModel: "model", Job: contracts.SubtitleJob{ID: newID(), MovieID: movie.MovieID, FileID: detail.SelectedFileID, State: "queued", Stage: "recognizing", TranslationModel: "configured-ai", CreatedAt: now, UpdatedAt: now, Assets: []contracts.SubtitleAsset{}}}
	if e = store.SaveSubtitleRecord(ctx, r); e != nil {
		t.Fatal(e)
	}
	opt := Options{Store: store, Root: filepath.Join(root, "work"), Provider: &fakeProvider{}, Config: func(string) (RuntimeConfig, error) {
		return RuntimeConfig{Enabled: true, APIKey: "secret", Text: llm.ClientConfig{BaseURL: r.TextBaseURL, Model: r.TextModel}}, nil
	}, Translate: func(ctx context.Context, r storage.SubtitleRecord, c []Cue) ([]Cue, error) {
		for i := range c {
			c[i].Translation = "你好"
		}
		return c, nil
	}}
	return opt, r
}
func waitJob(t *testing.T, s *Service, id, state string) contracts.SubtitleJob {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		j, e := s.Get(context.Background(), id)
		if e != nil {
			t.Fatal(e)
		}
		if j.State == state {
			return j
		}
		if j.State == "failed" {
			t.Fatalf("task failed: %+v", j)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("task did not reach " + state)
	return contracts.SubtitleJob{}
}
func TestRestartResumesTaskWithoutResubmittingAndCleansCheckpoint(t *testing.T) {
	opt, r := fixture(t)
	path := filepath.Join(opt.Root, r.Job.ID, "checkpoint.json")
	if e := writeJSON(path, checkpoint{Version: 1, Duration: 600, Parts: []part{{Offset: 600000, TaskID: "already-submitted", Submitted: true}}}); e != nil {
		t.Fatal(e)
	}
	s := New(context.Background(), opt)
	defer s.Close()
	job := waitJob(t, s, r.Job.ID, "completed")
	if opt.Provider.(*fakeProvider).calls != 1 {
		t.Fatal("resubmitted already known recognition task")
	}
	if len(job.Assets) != 3 {
		t.Fatalf("assets=%+v", job.Assets)
	}
	content, e := os.ReadFile(filepath.Join(filepath.Dir(r.SourcePath), "sub", job.Assets[1].Name))
	if e != nil || !strings.Contains(string(content), "00:10:00,100 --> 00:10:00,900\n你好") {
		t.Fatalf("content=%s err=%v", content, e)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("successful dialogue checkpoint retained")
	}
}
func TestRetryTranslationReusesRecognition(t *testing.T) {
	opt, r := fixture(t)
	r.Job.State = "failed"
	_ = opt.Store.SaveSubtitleRecord(context.Background(), r)
	_ = writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Begin: 0, End: 1000, Text: "こんにちは"}}}}})
	s := New(context.Background(), opt)
	defer s.Close()
	if _, e := s.Action(context.Background(), r.Job.ID, "retry"); e != nil {
		t.Fatal(e)
	}
	waitJob(t, s, r.Job.ID, "completed")
	if opt.Provider.(*fakeProvider).calls != 0 {
		t.Fatal("recognition repeated on translation retry")
	}
}
func TestPublishPreservesExistingFilesAndValidatesTranslationIDs(t *testing.T) {
	source := filepath.Join(t.TempDir(), "test.mp4")
	cues := []Cue{{ID: 1, Begin: 0, End: 1200, Text: "原文", Translation: "译文"}}
	first, e := publish(source, "file", newID(), cues)
	if e != nil {
		t.Fatal(e)
	}
	second, e := publish(source, "file", newID(), cues)
	if e != nil {
		t.Fatal(e)
	}
	if first[0].Name == second[0].Name {
		t.Fatal("overwrote prior generation")
	}
	if _, e = validateTranslations(cues, []Cue{{ID: 2, Translation: "wrong"}}); e == nil {
		t.Fatal("accepted unknown cue ID")
	}
	if _, e = validateTranslations(cues, nil); e == nil {
		t.Fatal("accepted missing cue")
	}
}

func TestCancelDuringTranslationCannotPublish(t *testing.T) {
	opt, r := fixture(t)
	_ = writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Begin: 0, End: 1000, Text: "原文"}}}}})
	started, release := make(chan struct{}), make(chan struct{})
	opt.Translate = func(ctx context.Context, r storage.SubtitleRecord, c []Cue) ([]Cue, error) {
		close(started)
		<-release
		c[0].Translation = "译文"
		return c, nil
	}
	s := New(context.Background(), opt)
	defer s.Close()
	<-started
	if _, e := s.Action(context.Background(), r.Job.ID, "cancel"); e != nil {
		close(release)
		t.Fatal(e)
	}
	close(release)
	waitJob(t, s, r.Job.ID, "cancelled")
	s.mu.Lock()
	s.mu.Unlock()
	if _, e := os.Stat(filepath.Join(filepath.Dir(r.SourcePath), "sub")); !os.IsNotExist(e) {
		t.Fatal("late translation published after cancellation")
	}
}
func TestSaveRetryNeedsNoProviderAndKeepsOldOutputs(t *testing.T) {
	opt, r := fixture(t)
	opt.Config = func(string) (RuntimeConfig, error) {
		t.Error("save retry contacted provider config")
		return RuntimeConfig{}, nil
	}
	_ = writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Begin: 0, End: 1000, Text: "原文", Translation: "译文"}}}}})
	s := New(context.Background(), opt)
	defer s.Close()
	j := waitJob(t, s, r.Job.ID, "completed")
	path, e := s.AssetPath(context.Background(), j.Assets[1].ID)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(path)
	b[len(b)-2] = 'x'
	_ = os.WriteFile(path, b, 0600)
	if _, e = s.AssetPath(context.Background(), j.Assets[1].ID); e == nil {
		t.Fatal("same-size modified output accepted as original")
	}
}

func TestFullExtractionPipelineWithSyntheticMedia(t *testing.T) {
	command, e := exec.LookPath("ffmpeg")
	if e != nil {
		t.Skip("FFmpeg not installed")
	}
	opt, r := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if e = executil.CommandContext(ctx, command, "-nostdin", "-v", "error", "-y", "-f", "lavfi", "-i", "color=size=160x90:rate=10", "-f", "lavfi", "-i", "sine=frequency=500:sample_rate=48000", "-t", "2", "-c:v", "libx264", "-c:a", "aac", r.SourcePath).Run(); e != nil {
		t.Fatal(e)
	}
	r.Revision, e = revision(r.SourcePath)
	if e != nil {
		t.Fatal(e)
	}
	if e = opt.Store.SaveSubtitleRecord(ctx, r); e != nil {
		t.Fatal(e)
	}
	opt.FFmpeg = command
	opt.Provider.(*fakeProvider).uploaded = func(path string) {
		if info, e := os.Stat(path); e != nil || info.Size() == 0 || filepath.Ext(path) != ".flac" {
			t.Error("missing extracted audio")
		}
	}
	s := New(ctx, opt)
	defer s.Close()
	j := waitJob(t, s, r.Job.ID, "completed")
	if len(j.Assets) != 3 || opt.Provider.(*fakeProvider).calls != 3 {
		t.Fatalf("pipeline output=%+v provider calls=%d", j, opt.Provider.(*fakeProvider).calls)
	}
}
func TestBlockedSaveRetryAndRelocationKeepOutputs(t *testing.T) {
	opt, r := fixture(t)
	cp := checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: []Cue{{ID: 1, Begin: 0, End: 1000, Text: "原文", Translation: "译文"}}}}}
	_ = writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), cp)
	sub := filepath.Join(filepath.Dir(r.SourcePath), "sub")
	if e := os.WriteFile(sub, []byte("occupied"), 0600); e != nil {
		t.Fatal(e)
	}
	s := New(context.Background(), opt)
	defer s.Close()
	waitJob(t, s, r.Job.ID, "blocked")
	_ = os.Remove(sub)
	if _, e := s.Action(context.Background(), r.Job.ID, "retry"); e != nil {
		t.Fatal(e)
	}
	j := waitJob(t, s, r.Job.ID, "completed")
	newPath := filepath.Join(filepath.Dir(r.SourcePath), "organized", "DEMO-001.mp4")
	_ = os.MkdirAll(filepath.Dir(newPath), 0755)
	if e := os.Rename(r.SourcePath, newPath); e != nil {
		t.Fatal(e)
	}
	if e := s.RelocateAssets(context.Background(), r.SourcePath, newPath); e != nil {
		t.Fatal(e)
	}
	if e := opt.Store.RelocateMovieFile(context.Background(), r.SourcePath, newPath); e != nil {
		t.Fatal(e)
	}
	path, e := s.AssetPath(context.Background(), j.Assets[1].ID)
	if e != nil || filepath.Dir(path) != filepath.Join(filepath.Dir(newPath), "sub") {
		t.Fatalf("relocated asset path=%s err=%v", path, e)
	}
	if opt.Provider.(*fakeProvider).calls != 0 {
		t.Fatal("save retry made provider calls")
	}
}
func TestBailianSubmissionProtocol(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("X-DashScope-Async") != "enable" || r.Header.Get("X-DashScope-OssResourceResolve") != "enable" {
			t.Error("missing provider headers")
		}
		var b struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Model != ASRModel {
			t.Error("wrong ASR model")
		}
		_, _ = w.Write([]byte(`{"output":{"task_id":"known-task"}}`))
	}))
	defer srv.Close()
	p := Bailian{BaseURL: srv.URL, Client: srv.Client()}
	id, e := p.Submit(context.Background(), "secret", "oss://file")
	if e != nil || id != "known-task" {
		t.Fatalf("id=%s err=%v", id, e)
	}
	for _, u := range []string{"http://result.aliyuncs.com/x", "https://evil.example/x", "https://aliyuncs.com.evil.example/x", "https://result.aliyuncs.com:444/x"} {
		if supplierURL(u) {
			t.Error("unsafe result URL accepted", u)
		}
	}
}

func TestQwen3ProtocolUsesSingularFileAndResult(t *testing.T) {
	client := &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch {
		case strings.Contains(r.URL.Path, "uploads"):
			if r.URL.Query().Get("model") != ASRModel3 {
				t.Error("upload authorization uses wrong model")
			}
			body = `{"data":{"upload_host":"https://upload.test.aliyuncs.com"}}`
		case r.Method == "POST":
			var req struct {
				Model      string         `json:"model"`
				Input      map[string]any `json:"input"`
				Parameters map[string]any `json:"parameters"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Model != ASRModel3 || req.Input["file_url"] != "oss://fixture" || req.Input["file_urls"] != nil || req.Parameters["language"] != "ja" || req.Parameters["language_hints"] != nil {
				t.Fatalf("wrong Qwen3 protocol: %+v", req)
			}
			body = `{"output":{"task_id":"qwen3-task"}}`
		case r.URL.Host == "results.test.aliyuncs.com":
			if r.URL.Scheme != "https" || r.URL.RawQuery != "signature=a%2Bb%2Fc&expires=123" || r.Header.Get("Authorization") != "" || r.Header.Get("Content-Type") != "" {
				t.Fatal("result download must use HTTPS and preserve signature without credential headers")
			}
			body = `{"transcripts":[{"sentences":[{"begin_time":100,"end_time":900,"text":"fixture"}]}]}`
		default:
			body = `{"output":{"task_status":"SUCCEEDED","result":{"transcription_url":"http://results.test.aliyuncs.com/result?signature=a%2Bb%2Fc&expires=123"}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	p := Bailian{Client: client}
	if err := p.TestPolicy(context.Background(), "secret", ASRModel3); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Submit(context.Background(), "secret", "oss://fixture", ASRModel3); err != nil {
		t.Fatal(err)
	}
	cues, done, err := p.Result(context.Background(), "secret", "qwen3-task")
	if err != nil || !done || len(cues) != 1 || cues[0].Begin != 100 {
		t.Fatalf("cues=%+v done=%v err=%v", cues, done, err)
	}
}

func TestTranscriptURLRejectsUntrustedAddresses(t *testing.T) {
	for _, raw := range []string{"http://evil.example/result", "http://aliyuncs.com.evil.example/result", "http://results.aliyuncs.com:80/result", "http://user@results.aliyuncs.com/result", "file:///result", "//results.aliyuncs.com/result"} {
		if _, ok := transcriptURL(raw); ok {
			t.Errorf("accepted untrusted URL: %s", raw)
		}
	}
	const signed = "https://results.aliyuncs.com/a%2Fb?signature=a%2Bb&x=1&x=2"
	if got, ok := transcriptURL(signed); !ok || got != signed {
		t.Fatal("HTTPS signature changed")
	}
}
