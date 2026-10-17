package subtitles

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func dialogue(count int, text string) []Cue {
	cues := make([]Cue, count)
	for i := range cues {
		cues[i] = Cue{ID: i + 1, Begin: int64(i * 1000), End: int64(i*1000 + 900), Text: text}
	}
	return cues
}

func TestTranslationUsesOneRequestAcrossRecognitionParts(t *testing.T) {
	opt, r := fixture(t)
	cues := dialogue(181, "こんにちは")
	cues[0].Translation = "保留的译文"
	cp := checkpoint{Version: 1, Duration: 1200, Parts: []part{
		{Recognized: true, Cues: cues[:90]}, {Recognized: true, Offset: 600000, Cues: cues[90:]},
	}}
	if err := writeJSON(filepath.Join(opt.Root, r.Job.ID, "checkpoint.json"), cp); err != nil {
		t.Fatal(err)
	}
	calls := 0
	opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		calls++
		if len(batch) != 180 || batch[0].ID != 2 || batch[len(batch)-1].ID != 181 {
			t.Fatalf("request lost pending dialogue: %d cues", len(batch))
		}
		for i := range batch {
			batch[i].Translation = "你好"
		}
		return batch, nil
	}
	s := &Service{opt: opt}
	if err := s.execute(context.Background(), &r); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || r.Job.Completed != 181 || r.Job.State != "completed" || opt.Provider.(*fakeProvider).calls != 0 {
		t.Fatalf("calls=%d job=%+v", calls, r.Job)
	}
	content, err := os.ReadFile(filepath.Join(filepath.Dir(r.SourcePath), "sub", r.Job.Assets[1].Name))
	if err != nil || !strings.Contains(string(content), "保留的译文") || strings.Count(string(content), "你好") != 180 {
		t.Fatalf("published translations lost: %v", err)
	}
}

func TestTranslationBatchesUseCapacityInsteadOfCueCount(t *testing.T) {
	short := dialogue(500, "はい")
	if batches := translationBatches(short, TranslationLimits{}); len(batches) != 1 {
		t.Fatalf("short dialogue split into %d requests", len(batches))
	}
	long := dialogue(50, strings.Repeat("日本語", 200))
	limits := TranslationLimits{ContextWindow: 32768, OutputBudget: 4096}
	batches := translationBatches(long, limits)
	if len(batches) <= 1 {
		t.Fatal("long dialogue exceeded capacity without splitting")
	}
	var got []Cue
	for _, batch := range batches {
		got = append(got, batch...)
	}
	if !reflect.DeepEqual(got, long) {
		t.Fatal("capacity split lost or reordered cues")
	}
	if translationBatches(nil, limits) != nil {
		t.Fatal("empty pending list created a request")
	}
}

func TestCapacityFallbackCheckpointsAndResumesSuccessfulHalf(t *testing.T) {
	opt, r := fixture(t)
	cp := checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: dialogue(4, "原文")}}}
	calls := []int{}
	blocked := errors.New("provider unavailable")
	opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		calls = append(calls, len(batch))
		if len(batch) > 2 {
			return nil, &TranslationError{Cause: llm.ErrOutputTruncated}
		}
		if batch[0].ID == 3 {
			return nil, blocked
		}
		for i := range batch {
			batch[i].Translation = "成功"
		}
		return batch, nil
	}
	s := &Service{opt: opt}
	completed := 0
	err := s.translatePending(context.Background(), &r, &cp, TranslationLimits{}, &completed, 4)
	if !errors.Is(err, blocked) || completed != 2 || !reflect.DeepEqual(calls, []int{4, 2, 2}) {
		t.Fatalf("calls=%v completed=%d err=%v", calls, completed, err)
	}
	raw, err := os.ReadFile(s.cpPath(r.Job.ID))
	if err != nil {
		t.Fatal(err)
	}
	var restored checkpoint
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	s.opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		if len(batch) != 2 || batch[0].ID != 3 {
			t.Fatalf("resubmitted completed translation: %+v", batch)
		}
		for i := range batch {
			batch[i].Translation = "重试"
		}
		return batch, nil
	}
	if err := s.translatePending(context.Background(), &r, &restored, TranslationLimits{}, &completed, 4); err != nil || completed != 4 {
		t.Fatalf("completed=%d err=%v", completed, err)
	}
}

func TestTranslationOnlySplitsOnExplicitCapacityErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		err   error
		split bool
	}{
		{"truncated", &TranslationError{Cause: llm.ErrOutputTruncated}, true},
		{"context", &llm.HTTPError{Status: 400, Detail: "context_length_exceeded"}, true},
		{"policy", &llm.HTTPError{Status: 400, Detail: "data_inspection_failed"}, false},
		{"auth", &llm.HTTPError{Status: 401, Detail: "invalid key"}, false},
		{"rate", &llm.HTTPError{Status: 429, Detail: "rate limit"}, false},
		{"timeout", context.DeadlineExceeded, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			opt, r := fixture(t)
			cp := checkpoint{Parts: []part{{Cues: dialogue(2, "原文")}}}
			calls := 0
			opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
				calls++
				return nil, test.err
			}
			s := &Service{opt: opt}
			completed := 0
			err := s.translatePending(context.Background(), &r, &cp, TranslationLimits{}, &completed, 2)
			want := 1
			if test.split {
				want = 2
			} // one-cue failure stops further requests
			if !errors.Is(err, test.err) || calls != want || completed != 0 {
				t.Fatalf("calls=%d completed=%d err=%v", calls, completed, err)
			}
		})
	}
}

func TestCapacityFallbackSurvivesInterruptionBeforeAnyTranslation(t *testing.T) {
	opt, r := fixture(t)
	cp := checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: dialogue(8, "原文")}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls []int
	opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		calls = append(calls, len(batch))
		if len(batch) > 2 {
			return nil, &TranslationError{Cause: llm.ErrOutputTruncated}
		}
		cancel()
		return nil, context.Canceled
	}
	s := &Service{opt: opt}
	completed := 0
	err := s.translatePending(ctx, &r, &cp, TranslationLimits{}, &completed, 8)
	if !errors.Is(err, context.Canceled) || completed != 0 || !reflect.DeepEqual(calls, []int{8, 4, 2}) {
		t.Fatalf("calls=%v completed=%d err=%v", calls, completed, err)
	}
	raw, err := os.ReadFile(s.cpPath(r.Job.ID))
	if err != nil {
		t.Fatal(err)
	}
	var restored checkpoint
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.TranslationBatchLimit != 2 {
		t.Fatalf("lost learned capacity before first successful result: %+v", restored)
	}
	calls = nil
	var ids []int
	s.opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		calls = append(calls, len(batch))
		for i := range batch {
			ids = append(ids, batch[i].ID)
			batch[i].Translation = "完成"
		}
		return batch, nil
	}
	if err := s.translatePending(context.Background(), &r, &restored, TranslationLimits{}, &completed, 8); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []int{2, 2, 2, 2}) || !reflect.DeepEqual(ids, []int{1, 2, 3, 4, 5, 6, 7, 8}) || completed != 8 {
		t.Fatalf("resume replayed oversized request or lost dialogue: calls=%v ids=%v completed=%d", calls, ids, completed)
	}
}

func TestOmittedCuesAreCheckpointedAndOnlyMissingIDsAreRepaired(t *testing.T) {
	opt, r := fixture(t)
	cp := checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: dialogue(4, "原文")}}}
	s := &Service{opt: opt}
	requests := [][]int{}
	s.opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		ids := []int{}
		for _, c := range batch {
			ids = append(ids, c.ID)
		}
		requests = append(requests, ids)
		if len(requests) == 1 {
			// The real app returns a validated subset plus an explicit error.
			return []Cue{{ID: 1, Translation: "第一条"}, {ID: 3, Translation: "第三条"}}, &IncompleteTranslationError{Expected: 4, Received: 2}
		}
		var saved checkpoint
		raw, err := os.ReadFile(s.cpPath(r.Job.ID))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Parts[0].Cues[0].Translation != "第一条" || saved.Parts[0].Cues[2].Translation != "第三条" || r.Job.Completed != 2 {
			t.Fatal("repair began before partial results and progress were saved")
		}
		for i := range batch {
			batch[i].Translation = "补译"
		}
		return batch, nil
	}
	completed := 0
	if err := s.translatePending(context.Background(), &r, &cp, TranslationLimits{}, &completed, 4); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requests, [][]int{{1, 2, 3, 4}, {2, 4}}) || completed != 4 || r.Job.Completed != 4 {
		t.Fatalf("requests=%v completed=%d record=%d", requests, completed, r.Job.Completed)
	}
}

func TestOmissionRepairIsBoundedAndRetryRetainsProgress(t *testing.T) {
	opt, r := fixture(t)
	cp := checkpoint{Version: 1, Duration: 10, Parts: []part{{Recognized: true, Cues: dialogue(5, "原文")}}}
	s := &Service{opt: opt}
	calls := 0
	s.opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		calls++
		// Callbacks with nil error still receive service-boundary validation.
		return []Cue{{ID: batch[0].ID, Translation: "已译"}}, nil
	}
	completed := 0
	err := s.translatePending(context.Background(), &r, &cp, TranslationLimits{}, &completed, 5)
	var incomplete *IncompleteTranslationError
	if !errors.As(err, &incomplete) || calls != 3 || completed != 3 || r.Job.Completed != 3 {
		t.Fatalf("repair loop not bounded or progress lost: calls=%d completed=%d error=%v", calls, completed, err)
	}
	raw, err := os.ReadFile(s.cpPath(r.Job.ID))
	if err != nil {
		t.Fatal(err)
	}
	var restored checkpoint
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	s.opt.Translate = func(_ context.Context, _ storage.SubtitleRecord, batch []Cue) ([]Cue, error) {
		if len(batch) != 2 || batch[0].ID != 4 || batch[1].ID != 5 {
			t.Fatalf("retry lost progress: %+v", batch)
		}
		for i := range batch {
			batch[i].Translation = "完成"
		}
		return batch, nil
	}
	if err := s.translatePending(context.Background(), &r, &restored, TranslationLimits{}, &completed, 5); err != nil || completed != 5 {
		t.Fatalf("retry failed: completed=%d error=%v", completed, err)
	}
}

func TestInvalidOrEmptyResultsDoNotTriggerOmissionRepair(t *testing.T) {
	for _, result := range [][]Cue{
		nil,
		{{ID: 1, Translation: "有效"}, {ID: 1, Translation: "重复"}},
		{{ID: 1, Translation: "有效"}, {ID: 99, Translation: "未知"}},
		{{ID: 1, Translation: "有效"}, {ID: 2, Translation: " "}},
	} {
		opt, r := fixture(t)
		cp := checkpoint{Parts: []part{{Cues: dialogue(2, "原文")}}}
		calls := 0
		opt.Translate = func(context.Context, storage.SubtitleRecord, []Cue) ([]Cue, error) { calls++; return result, nil }
		s := &Service{opt: opt}
		completed := 0
		if err := s.translatePending(context.Background(), &r, &cp, TranslationLimits{}, &completed, 2); err == nil || calls != 1 || completed != 0 || cp.Parts[0].Cues[0].Translation != "" {
			t.Fatalf("invalid result repaired or persisted: calls=%d completed=%d error=%v", calls, completed, err)
		}
	}
}
