package subtitles

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/contracts"
	"curated-backend/internal/executil"
	"curated-backend/internal/llm"
	"curated-backend/internal/playback"
	"curated-backend/internal/storage"
)

type RuntimeConfig struct {
	RecognitionModel string
	Enabled          bool
	APIKey           string
	Text             llm.ClientConfig
	DefaultModel     string
}
type Options struct {
	Store     *storage.SQLiteStore
	Root      string
	FFmpeg    string
	Provider  Provider
	Config    func(string) (RuntimeConfig, error)
	Translate func(context.Context, storage.SubtitleRecord, []Cue) ([]Cue, error)
}
type Service struct {
	opt    Options
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	wake   chan struct{}
	active map[string]context.CancelFunc
}
type checkpoint struct {
	Assets   []contracts.SubtitleAsset `json:"assets,omitempty"`
	Version  int                       `json:"version"`
	Duration float64                   `json:"duration"`
	Parts    []part                    `json:"parts"`
}

func (s *Service) AudioTracks(ctx context.Context, movieID, fileID string) ([]contracts.SubtitleAudioTrack, error) {
	d, e := s.opt.Store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(ctx, fileID), movieID)
	if e != nil {
		return nil, e
	}
	return playback.ProbeAudioTracks(ctx, d.Location, s.opt.FFmpeg)
}

type part struct {
	Offset     int64  `json:"offset"`
	TaskID     string `json:"taskId,omitempty"`
	Submitted  bool   `json:"submitted"`
	Recognized bool   `json:"recognized"`
	Cues       []Cue  `json:"cues,omitempty"`
}

func newID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func New(ctx context.Context, opt Options) *Service {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{opt: opt, cancel: cancel, done: make(chan struct{}), wake: make(chan struct{}, 1), active: map[string]context.CancelFunc{}}
	go s.loop(ctx)
	return s
}
func (s *Service) Close() { s.cancel(); <-s.done }
func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func revision(path string) (string, error) {
	i, e := os.Stat(path)
	if e != nil {
		return "", e
	}
	if !i.Mode().IsRegular() {
		return "", fmt.Errorf("source is not a regular file")
	}
	return fmt.Sprintf("%d:%d", i.Size(), i.ModTime().UnixNano()), nil
}
func active(state string) bool { return state == "queued" || state == "running" }
func (s *Service) Create(ctx context.Context, movieID string, req contracts.SubtitleRequest) (contracts.SubtitleJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(req.RequestID) < 8 || len(req.RequestID) > 128 || req.AudioTrack < 0 || req.AudioTrack > 31 {
		return contracts.SubtitleJob{}, fmt.Errorf("invalid subtitle request")
	}
	records, e := s.opt.Store.ListSubtitleRecords(ctx, "")
	if e != nil {
		return contracts.SubtitleJob{}, e
	}
	detail, e := s.opt.Store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(ctx, req.FileID), movieID)
	if e != nil {
		return contracts.SubtitleJob{}, e
	}
	for _, r := range records {
		if r.RequestID == req.RequestID {
			if r.Job.MovieID != movieID || r.Job.FileID != detail.SelectedFileID {
				return contracts.SubtitleJob{}, fmt.Errorf("request ID belongs to another file")
			}
			return r.Job, nil
		}
		if r.Job.FileID == detail.SelectedFileID && active(r.Job.State) {
			return r.Job, nil
		}
	}
	queued := 0
	for _, r := range records {
		if active(r.Job.State) {
			queued++
		}
	}
	if queued >= 8 {
		return contracts.SubtitleJob{}, fmt.Errorf("subtitle queue is full (maximum 8 files)")
	}
	cfg, e := s.opt.Config(req.TranslationModel)
	if e != nil {
		return contracts.SubtitleJob{}, e
	}
	if !cfg.Enabled || cfg.APIKey == "" {
		return contracts.SubtitleJob{}, fmt.Errorf("subtitle provider is not enabled or configured")
	}
	rev, e := revision(detail.Location)
	if e != nil {
		return contracts.SubtitleJob{}, e
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	model := req.TranslationModel
	if model == "" {
		model = cfg.DefaultModel
	}
	r := storage.SubtitleRecord{RequestID: req.RequestID, SourcePath: detail.Location, Revision: rev, AudioTrack: req.AudioTrack, TextBaseURL: cfg.Text.BaseURL, TextModel: cfg.Text.Model, PromptVersion: prompts.SubtitleTranslationPrompt().Version, Job: contracts.SubtitleJob{ID: newID(), MovieID: movieID, FileID: detail.SelectedFileID, FileName: filepath.Base(detail.Location), State: "queued", Stage: "preparing", RecognitionModel: cfg.RecognitionModel, TranslationModel: model, CreatedAt: now, UpdatedAt: now, Assets: []contracts.SubtitleAsset{}}}
	if e = s.opt.Store.SaveSubtitleRecord(ctx, r); e != nil {
		return r.Job, e
	}
	s.signal()
	return r.Job, nil
}
func (s *Service) List(ctx context.Context, movieID string) ([]contracts.SubtitleJob, error) {
	records, e := s.opt.Store.ListSubtitleRecords(ctx, movieID)
	if e != nil {
		return nil, e
	}
	jobs := []contracts.SubtitleJob{}
	for _, r := range records {
		j := r.Job
		for i := range j.Assets {
			_, e := s.AssetPath(ctx, j.Assets[i].ID)
			j.Assets[i].Missing = e != nil
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func (s *Service) Get(ctx context.Context, id string) (contracts.SubtitleJob, error) {
	r, e := s.opt.Store.SubtitleRecord(ctx, id)
	return r.Job, e
}
func (s *Service) Action(ctx context.Context, id, action string) (contracts.SubtitleJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, e := s.opt.Store.SubtitleRecord(ctx, id)
	if e != nil {
		return r.Job, e
	}
	switch action {
	case "cancel":
		if !active(r.Job.State) {
			return r.Job, nil
		}
		r.Job.State = "cancelled"
		r.Job.ExpiresAt = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339Nano)
	case "retry":
		if active(r.Job.State) || r.Job.State == "completed" {
			return r.Job, nil
		}
		if _, ok := s.active[id]; ok {
			return r.Job, fmt.Errorf("task is still stopping; retry shortly")
		}
		if _, e := os.Stat(s.cpPath(id)); e != nil {
			return r.Job, fmt.Errorf("checkpoint expired; start a new generation")
		}
		r.Job.State = "queued"
		r.Job.Error = ""
		r.Job.ExpiresAt = ""
	default:
		return r.Job, fmt.Errorf("invalid subtitle action")
	}
	r.Job.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if e = s.opt.Store.SaveSubtitleRecord(ctx, r); e != nil {
		return r.Job, e
	}
	if cancel := s.active[id]; cancel != nil && action == "cancel" {
		cancel()
	}
	s.signal()
	return r.Job, nil
}
func (s *Service) dir(id string) string    { return filepath.Join(s.opt.Root, id) }
func (s *Service) cpPath(id string) string { return filepath.Join(s.dir(id), "checkpoint.json") }
func writeJSON(path string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	tmp := path + ".tmp"
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, path)
}
func (s *Service) update(ctx context.Context, r *storage.SubtitleRecord, stage string, completed, total int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest, e := s.opt.Store.SubtitleRecord(ctx, r.Job.ID)
	if e != nil {
		return e
	}
	if latest.Job.State == "cancelled" {
		return context.Canceled
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.Job.State = "running"
	r.Job.Stage = stage
	r.Job.Completed = completed
	r.Job.Total = total
	r.Job.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return s.opt.Store.SaveSubtitleRecord(ctx, *r)
}
func (s *Service) loop(ctx context.Context) {
	defer close(s.done)
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		records, e := s.opt.Store.ListSubtitleRecords(ctx, "")
		if e == nil {
			s.cleanupOrphans(records)
			// Oldest queued task first; only this worker invokes paid provider operations.
			for i := len(records) - 1; i >= 0; i-- {
				r := records[i]
				if ctx.Err() != nil {
					return
				}
				if active(r.Job.State) {
					s.run(ctx, r)
				} else if r.Job.State == "completed" {
					_ = os.RemoveAll(s.dir(r.Job.ID))
				} else if r.Job.ExpiresAt != "" {
					if expiry, e := time.Parse(time.RFC3339Nano, r.Job.ExpiresAt); e == nil && time.Now().After(expiry) {
						_ = os.RemoveAll(s.dir(r.Job.ID))
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-tick.C:
		}
	}
}

func (s *Service) cleanupOrphans(records []storage.SubtitleRecord) {
	known := map[string]bool{}
	for _, r := range records {
		known[r.Job.ID] = true
	}
	entries, e := os.ReadDir(s.opt.Root)
	if e != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || known[entry.Name()] || len(entry.Name()) != 32 {
			continue
		}
		if _, e := hex.DecodeString(entry.Name()); e != nil {
			continue
		}
		info, e := entry.Info()
		if e != nil || time.Since(info.ModTime()) < 24*time.Hour {
			continue
		}
		_ = os.RemoveAll(filepath.Join(s.opt.Root, entry.Name()))
	}
}
func (s *Service) run(parent context.Context, r storage.SubtitleRecord) {
	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.active[r.Job.ID] = cancel
	s.mu.Unlock()
	e := s.execute(ctx, &r)
	cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, r.Job.ID)
	if parent.Err() != nil {
		return
	} // Shutdown preserves resumable state.
	latest, readErr := s.opt.Store.SubtitleRecord(parent, r.Job.ID)
	if readErr != nil || latest.Job.State == "cancelled" {
		return
	}
	if e != nil {
		r.Job.State = "failed"
		if r.Job.Stage == "saving" {
			r.Job.State = "blocked"
		}
		r.Job.Error = e.Error()
		r.Job.ExpiresAt = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339Nano)
		r.Job.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		_ = s.opt.Store.SaveSubtitleRecord(parent, r)
	}
}
func (s *Service) validateSource(ctx context.Context, r storage.SubtitleRecord) error {
	detail, e := s.opt.Store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(ctx, r.Job.FileID), r.Job.MovieID)
	if e != nil {
		return e
	}
	rev, e := revision(detail.Location)
	if e != nil {
		return fmt.Errorf("source unavailable")
	}
	if rev != r.Revision || filepath.Clean(detail.Location) != filepath.Clean(r.SourcePath) {
		return fmt.Errorf("source changed or moved; start a new generation")
	}
	return nil
}
func (s *Service) execute(ctx context.Context, r *storage.SubtitleRecord) error {
	if e := s.validateSource(ctx, *r); e != nil {
		return e
	}
	cp := checkpoint{Version: 1}
	raw, e := os.ReadFile(s.cpPath(r.Job.ID))
	if e == nil {
		if e = json.Unmarshal(raw, &cp); e != nil {
			return fmt.Errorf("checkpoint damaged")
		}
		if cp.Version != 1 {
			return fmt.Errorf("unsupported checkpoint version")
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	needsCloud := cp.Duration == 0
	for _, part := range cp.Parts {
		if !part.Recognized {
			needsCloud = true
		}
		for _, cue := range part.Cues {
			if cue.Translation == "" {
				needsCloud = true
			}
		}
	}
	var cfg RuntimeConfig
	if needsCloud {
		if r.PromptVersion != "" && r.PromptVersion != prompts.SubtitleTranslationPrompt().Version {
			return fmt.Errorf("translation prompt changed; start a new generation")
		}
		cfg, e = s.opt.Config(r.Job.TranslationModel)
		if e != nil {
			return e
		}
		if !cfg.Enabled || cfg.APIKey == "" {
			return fmt.Errorf("subtitle provider unavailable")
		}
		if cfg.Text.Model != r.TextModel || cfg.Text.BaseURL != r.TextBaseURL {
			return fmt.Errorf("translation model configuration changed")
		}
	}
	if cp.Duration == 0 {
		if e = s.update(ctx, r, "preparing", 0, 0); e != nil {
			return e
		}
		duration, e := playback.ProbeMediaDuration(ctx, r.SourcePath, s.opt.FFmpeg)
		if e != nil || duration <= 0 || duration > 12*3600 {
			return fmt.Errorf("could not probe media duration (maximum 12 hours)")
		}
		cp.Duration = duration
		for offset := float64(0); offset < duration; offset += 600 {
			cp.Parts = append(cp.Parts, part{Offset: int64(offset * 1000)})
		}
		if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
			return e
		}
	}
	for i := range cp.Parts {
		if e = s.validateSource(ctx, *r); e != nil {
			return e
		}
		p := &cp.Parts[i]
		if !p.Recognized {
			if p.TaskID == "" {
				if p.Submitted {
					return fmt.Errorf("recognition submission result unknown; start a new generation to submit again")
				}
				if e = s.update(ctx, r, "extracting", i, len(cp.Parts)); e != nil {
					return e
				}
				audio := filepath.Join(s.dir(r.Job.ID), fmt.Sprintf("part-%04d.flac", i))
				if e = os.MkdirAll(s.dir(r.Job.ID), 0700); e != nil {
					return e
				}
				extractCtx, stop := context.WithTimeout(ctx, 15*time.Minute)
				cmd := executil.CommandContext(extractCtx, playback.ResolveFFmpegCommand(s.opt.FFmpeg), "-nostdin", "-v", "error", "-y", "-ss", strconv.FormatFloat(float64(p.Offset)/1000, 'f', 3, 64), "-i", r.SourcePath, "-t", "600", "-map", fmt.Sprintf("0:a:%d", r.AudioTrack), "-vn", "-ac", "1", "-ar", "16000", "-c:a", "flac", audio)
				err := cmd.Run()
				stop()
				if err != nil {
					return fmt.Errorf("audio extraction failed (check audio track and FFmpeg)")
				}
				if e = s.update(ctx, r, "uploading", i, len(cp.Parts)); e != nil {
					return e
				}
				uploaded, err := s.opt.Provider.Upload(ctx, cfg.APIKey, audio, r.Job.RecognitionModel)
				if err != nil {
					return err
				}
				p.Submitted = true
				if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
					return e
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				p.TaskID, e = s.opt.Provider.Submit(ctx, cfg.APIKey, uploaded, r.Job.RecognitionModel)
				if e != nil {
					return e
				}
				if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
					return e
				}
			}
			if e = s.update(ctx, r, "recognizing", i, len(cp.Parts)); e != nil {
				return e
			}
			pollCtx, stop := context.WithTimeout(ctx, 60*time.Minute)
			for {
				cues, done, err := s.opt.Provider.Result(pollCtx, cfg.APIKey, p.TaskID)
				if err != nil {
					var failed *RecognitionTaskFailed
					if errors.As(err, &failed) {
						p.TaskID = ""
						p.Submitted = false
						if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
							stop()
							return e
						}
					}
					stop()
					return err
				}
				if done {
					p.Cues = cues
					p.Recognized = true
					for j := range p.Cues {
						p.Cues[j].Begin += p.Offset
						p.Cues[j].End += p.Offset
						p.Cues[j].ID = i*100000 + j + 1
					}
					break
				}
				select {
				case <-pollCtx.Done():
					stop()
					return pollCtx.Err()
				case <-time.After(3 * time.Second):
				}
			}
			stop()
			if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
				return e
			}
			_ = os.Remove(filepath.Join(s.dir(r.Job.ID), fmt.Sprintf("part-%04d.flac", i)))
		}
	}
	total, completed := 0, 0
	for _, p := range cp.Parts {
		total += len(p.Cues)
		for _, c := range p.Cues {
			if c.Translation != "" {
				completed++
			}
		}
	}
	if total == 0 {
		return fmt.Errorf("no speech recognized")
	}
	for pi := range cp.Parts {
		p := &cp.Parts[pi]
		for start := 0; start < len(p.Cues); start += 30 {
			end := start + 30
			if end > len(p.Cues) {
				end = len(p.Cues)
			}
			pending := []Cue{}
			for _, c := range p.Cues[start:end] {
				if c.Translation == "" {
					pending = append(pending, c)
				}
			}
			if len(pending) == 0 {
				continue
			}
			if e = s.update(ctx, r, "translating", completed, total); e != nil {
				return e
			}
			translated, err := s.opt.Translate(ctx, *r, pending)
			if err != nil {
				return err
			}
			values, err := validateTranslations(pending, translated)
			if err != nil {
				return err
			}
			for j := start; j < end; j++ {
				if v, ok := values[p.Cues[j].ID]; ok {
					p.Cues[j].Translation = v
					completed++
				}
			}
			if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
				return e
			}
		}
	}
	if e = s.update(ctx, r, "saving", completed, total); e != nil {
		return e
	}
	if e = s.validateSource(ctx, *r); e != nil {
		return e
	}
	cues := []Cue{}
	for _, p := range cp.Parts {
		cues = append(cues, p.Cues...)
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].Begin < cues[j].Begin })
	s.mu.Lock()
	defer s.mu.Unlock()
	latest, e := s.opt.Store.SubtitleRecord(ctx, r.Job.ID)
	if e != nil {
		return e
	}
	if latest.Job.State == "cancelled" || ctx.Err() != nil {
		return context.Canceled
	}
	assets := cp.Assets
	if len(assets) == 0 {
		assets, e = publish(r.SourcePath, r.Job.FileID, r.Job.ID, cues)
		if e != nil {
			return e
		}
		cp.Assets = assets
		if e = writeJSON(s.cpPath(r.Job.ID), cp); e != nil {
			return e
		}
	} else {
		for _, asset := range assets {
			if filepath.Base(asset.Name) != asset.Name {
				return fmt.Errorf("invalid saved output name")
			}
			raw, err := os.ReadFile(filepath.Join(filepath.Dir(r.SourcePath), "sub", asset.Name))
			if err != nil {
				return fmt.Errorf("saved output missing; restore it before retry")
			}
			digest := sha256.Sum256(raw)
			if hex.EncodeToString(digest[:]) != asset.Hash {
				return fmt.Errorf("saved output changed; original preserved")
			}
		}
	}
	r.Job.Assets = assets
	r.Job.State = "completed"
	r.Job.Stage = "done"
	r.Job.Error = ""
	r.Job.ExpiresAt = ""
	r.Job.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if e = s.opt.Store.SaveSubtitleRecord(ctx, *r); e != nil {
		return e
	}
	_ = os.RemoveAll(s.dir(r.Job.ID))
	return nil
}
func validateTranslations(input, output []Cue) (map[int]string, error) {
	expected := map[int]bool{}
	for _, c := range input {
		expected[c.ID] = true
	}
	values := map[int]string{}
	for _, c := range output {
		if !expected[c.ID] || values[c.ID] != "" || strings.TrimSpace(c.Translation) == "" {
			return nil, fmt.Errorf("translation returned invalid or duplicate cue IDs")
		}
		values[c.ID] = strings.TrimSpace(c.Translation)
	}
	if len(values) != len(input) {
		return nil, fmt.Errorf("translation omitted cues; retry remaining translation")
	}
	return values, nil
}
func timestamp(ms int64) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, (ms/60000)%60, (ms/1000)%60, ms%1000)
}
func outputText(cues []Cue, translated bool, txt bool) []byte {
	var b strings.Builder
	for i, c := range cues {
		value := c.Text
		if translated {
			value = c.Translation
		}
		value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", ""), "\n\n", "\n")
		if txt {
			fmt.Fprintln(&b, value)
		} else {
			fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, timestamp(c.Begin), timestamp(c.End), value)
		}
	}
	return []byte(b.String())
}
func publish(source, fileID, id string, cues []Cue) ([]contracts.SubtitleAsset, error) {
	dir := filepath.Join(filepath.Dir(source), "sub")
	if e := os.MkdirAll(dir, 0755); e != nil {
		return nil, fmt.Errorf("cannot create sub directory")
	}
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	suffixes := []string{".ja.srt", ".zh-CN.srt", ".zh-CN.txt"}
	bodies := [][]byte{outputText(cues, false, false), outputText(cues, true, false), outputText(cues, true, true)}
	// Every regeneration has a distinct name; reserve all three before writing.
	for _, suffix := range suffixes {
		if _, e := os.Lstat(filepath.Join(dir, stem+suffix)); !errors.Is(e, os.ErrNotExist) {
			stem += "." + id[:8]
			break
		}
	}
	paths := []string{}
	assets := []contracts.SubtitleAsset{}
	success := false
	defer func() {
		if !success {
			for _, path := range paths {
				_ = os.Remove(path)
			}
		}
	}()
	for i, suffix := range suffixes {
		name := stem + suffix
		path := filepath.Join(dir, name)
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if e != nil {
			return nil, fmt.Errorf("cannot reserve subtitle output; existing files preserved")
		}
		paths = append(paths, path)
		_, e = f.Write(bodies[i])
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			return nil, fmt.Errorf("cannot save subtitle output")
		}
		hash := sha256.Sum256(bodies[i])
		lang := "zh-CN"
		if i == 0 {
			lang = "ja"
		}
		format := "srt"
		if i == 2 {
			format = "txt"
		}
		assets = append(assets, contracts.SubtitleAsset{ID: id + "-" + strconv.Itoa(i), FileID: fileID, Name: name, Language: lang, Format: format, Size: int64(len(bodies[i])), Hash: hex.EncodeToString(hash[:])})
	}
	success = true
	return assets, nil
}

// AssetPath resolves only metadata-backed IDs against the current selected media directory.
func (s *Service) AssetPath(ctx context.Context, id string) (string, error) {
	if len(id) != 34 || id[32] != '-' {
		return "", fmt.Errorf("invalid subtitle asset ID")
	}
	r, e := s.opt.Store.SubtitleRecord(ctx, id[:32])
	if e != nil {
		return "", e
	}
	detail, e := s.opt.Store.GetMoviePlaybackDetail(contracts.WithMovieFileSelection(ctx, r.Job.FileID), r.Job.MovieID)
	if e != nil {
		return "", e
	}
	if rev, e := revision(detail.Location); e != nil || rev != r.Revision {
		return "", fmt.Errorf("subtitle source changed or unavailable")
	}
	for _, a := range r.Job.Assets {
		if a.ID == id && filepath.Base(a.Name) == a.Name {
			dir := filepath.Join(filepath.Dir(detail.Location), "sub")
			parent, e := os.Lstat(dir)
			if e != nil || !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("subtitle directory unavailable")
			}
			path := filepath.Join(dir, a.Name)
			info, e := os.Lstat(path)
			if e != nil || !info.Mode().IsRegular() || info.Size() != a.Size {
				return "", fmt.Errorf("subtitle asset missing or changed")
			}
			if info.Size() > 16*1024*1024 {
				return "", fmt.Errorf("subtitle file too large")
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				return "", e
			}
			digest := sha256.Sum256(raw)
			if hex.EncodeToString(digest[:]) != a.Hash {
				return "", fmt.Errorf("subtitle asset changed")
			}
			return path, nil
		}
	}
	return "", fmt.Errorf("subtitle asset not found")
}
