package tools

import (
	"context"
	"encoding/json"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

type stubWrite struct {
	exists  bool
	comment contracts.MovieCommentDTO
	detail  contracts.MovieDetailDTO
	saved   string
	writes  int
	patched contracts.PatchMovieInput
	views   int
}

func (s *stubWrite) MovieExists(context.Context, string) (bool, error) {
	return s.exists, nil
}
func (s *stubWrite) GetMovieComment(context.Context, string) (contracts.MovieCommentDTO, error) {
	return s.comment, nil
}
func (s *stubWrite) UpsertMovieComment(_ context.Context, _, body string) (contracts.MovieCommentDTO, error) {
	s.writes++
	s.saved = body
	return contracts.MovieCommentDTO{Body: body, UpdatedAt: "t"}, nil
}
func (s *stubWrite) GetMovieDetail(context.Context, string) (contracts.MovieDetailDTO, error) {
	return s.detail, nil
}
func (s *stubWrite) PatchMovieDisplayOverrides(_ context.Context, _ string, patch contracts.PatchMovieInput) (contracts.MovieDetailDTO, error) {
	s.writes++
	s.patched = patch
	out := s.detail
	if patch.UserTitleSet && !patch.UserTitleClear {
		out.Title = patch.UserTitle
	}
	if patch.UserSummarySet && !patch.UserSummaryClear {
		out.Summary = patch.UserSummary
	}
	return out, nil
}
func (s *stubWrite) CreateSavedView(_ context.Context, name string, filters contracts.SavedViewFiltersV1) (contracts.SavedViewDTO, error) {
	s.writes++
	s.views++
	return contracts.SavedViewDTO{ID: "view_1", Name: name, Filters: filters}, nil
}

func TestSaveMovieCommentPreviewDoesNotWrite(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubWrite{exists: true, comment: contracts.MovieCommentDTO{Body: "old"}}
	if err := RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	args := json.RawMessage(`{"movieId":"m1","body":"new note"}`)
	preview := gw.Invoke(context.Background(), core.Call{
		Name: core.SaveMovieCommentName, Args: args, SessionID: "act_1", Channel: core.ChannelAction,
	})
	if !preview.OK || preview.ConfirmToken == "" || len(preview.Changes) != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	if write.writes != 0 {
		t.Fatalf("preview wrote %d times", write.writes)
	}

	applied := gw.Invoke(context.Background(), core.Call{
		Name: core.SaveMovieCommentName, Args: args, SessionID: "act_1", Channel: core.ChannelAction,
		ConfirmTok: preview.ConfirmToken,
	})
	if !applied.OK {
		t.Fatalf("apply = %+v", applied)
	}
	if write.writes != 1 || write.saved != "new note" {
		t.Fatalf("writes=%d saved=%q", write.writes, write.saved)
	}
}

func TestSaveMovieCommentChatCannotApplyWithoutWriteSwitch(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubWrite{exists: true, comment: contracts.MovieCommentDTO{Body: "old"}}
	if err := RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	args := json.RawMessage(`{"movieId":"m1","body":"new"}`)
	preview := gw.Invoke(context.Background(), core.Call{
		Name: core.SaveMovieCommentName, Args: args, SessionID: "ses_1", Channel: core.ChannelChat,
	})
	if preview.ConfirmToken == "" {
		t.Fatalf("expected token: %+v", preview)
	}
	denied := gw.Invoke(context.Background(), core.Call{
		Name: core.SaveMovieCommentName, Args: args, SessionID: "ses_1", Channel: core.ChannelChat,
		ConfirmTok: preview.ConfirmToken,
	})
	if denied.OK || denied.Error == nil || denied.Error.Code != "AI_TOOL_PERMISSION_DENIED" {
		t.Fatalf("chat apply = %+v", denied)
	}
	if write.writes != 0 {
		t.Fatal("chat apply wrote")
	}
}

func TestSaveMovieCommentNoopSkipsToken(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubWrite{exists: true, comment: contracts.MovieCommentDTO{Body: "same"}}
	if err := RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, nil)
	preview := gw.Invoke(context.Background(), core.Call{
		Name:      core.SaveMovieCommentName,
		Args:      json.RawMessage(`{"movieId":"m1","body":"same"}`),
		SessionID: "act_2",
		Channel:   core.ChannelAction,
	})
	if !preview.OK || preview.ConfirmToken != "" {
		t.Fatalf("noop preview = %+v", preview)
	}
}

func TestUpdateMovieDisplayPreviewDoesNotWrite(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubWrite{exists: true, detail: contracts.MovieDetailDTO{MovieListItemDTO: contracts.MovieListItemDTO{Title: "old"}, Summary: "ad filled"}}
	if err := RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	args := json.RawMessage(`{"movieId":"m1","userSummary":"clean plot"}`)
	preview := gw.Invoke(context.Background(), core.Call{
		Name: core.UpdateMovieDisplayOverridesName, Args: args, SessionID: "act_d", Channel: core.ChannelAction,
	})
	if !preview.OK || preview.ConfirmToken == "" || write.writes != 0 {
		t.Fatalf("preview = %+v writes=%d", preview, write.writes)
	}
	applied := gw.Invoke(context.Background(), core.Call{
		Name: core.UpdateMovieDisplayOverridesName, Args: args, SessionID: "act_d", Channel: core.ChannelAction,
		ConfirmTok: preview.ConfirmToken,
	})
	if !applied.OK || write.writes != 1 || !write.patched.UserSummarySet || write.patched.UserSummary != "clean plot" {
		t.Fatalf("apply = %+v writes=%d patch=%+v", applied, write.writes, write.patched)
	}
}

func TestCreateSavedViewPreviewThenApply(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	write := &stubWrite{}
	if err := RegisterWriteTools(reg, write); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, core.NewConfirmStore(), nil, func() core.Settings { return core.Settings{} })
	args := json.RawMessage(`{"name":"睡前短片","filters":{"schemaVersion":1,"playState":"unwatched","userRating":4,"runtime":"short"}}`)
	preview := gw.Invoke(context.Background(), core.Call{
		Name: core.CreateSavedViewName, Args: args, SessionID: "act_v", Channel: core.ChannelAction,
	})
	if !preview.OK || preview.ConfirmToken == "" || write.writes != 0 {
		t.Fatalf("preview = %+v writes=%d", preview, write.writes)
	}
	applied := gw.Invoke(context.Background(), core.Call{
		Name: core.CreateSavedViewName, Args: args, SessionID: "act_v", Channel: core.ChannelAction,
		ConfirmTok: preview.ConfirmToken,
	})
	if !applied.OK || write.views != 1 {
		t.Fatalf("apply = %+v views=%d", applied, write.views)
	}
}
