package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

func (a *App) GetBookMetadataState(ctx context.Context, kind, id string) (contracts.BookMetadataState, error) {
	switch kind {
	case "comic":
		d, err := a.GetComicBookDetail(ctx, id)
		return contracts.BookMetadataState{Title: d.Title, OriginalTitle: d.OriginalTitle, SourceFileName: d.SourceFileName, Metadata: d.Metadata}, err
	case "photo":
		d, err := a.GetPhotoBookDetail(ctx, id)
		return contracts.BookMetadataState{Title: d.Title, OriginalTitle: d.OriginalTitle, SourceFileName: d.SourceFileName, Metadata: d.Metadata}, err
	}
	return contracts.BookMetadataState{}, fmt.Errorf("invalid book kind")
}

func (a *App) ApplyBookMetadata(ctx context.Context, kind, id string, metadata contracts.BookMetadata, expected string) (contracts.BookMetadataState, error) {
	if _, err := a.GetBookMetadataState(ctx, kind, id); err != nil {
		return contracts.BookMetadataState{}, err
	}
	return a.store.ApplyBookMetadata(ctx, kind, id, metadata, expected)
}

func (a *App) runBookMetadataAction(ctx context.Context, completer commentCompleter, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
	comicID, photoID := strings.TrimSpace(req.ComicID), strings.TrimSpace(req.PhotoID)
	if (comicID == "") == (photoID == "") || strings.TrimSpace(req.MovieID) != "" {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("exactly one comicId or photoId is required")
	}
	kind, id := "comic", comicID
	if photoID != "" {
		kind, id = "photo", photoID
	}
	state, err := a.GetBookMetadataState(ctx, kind, id)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	// Deliberately exclude location, personal notes and reading history from the model.
	source, _ := json.Marshal(map[string]string{"originalTitle": state.OriginalTitle, "sourceFileName": state.SourceFileName})
	if len(source) > 16000 {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("book source title too long")
	}
	locale := strings.TrimSpace(req.TargetLocale)
	if locale == "" {
		locale = strings.TrimSpace(req.Locale)
	}
	raw, err := completer.Complete(ctx, []llm.ChatMessage{{Role: "system", Content: prompts.BookMetadataPrompt(kind, locale)}, {Role: "user", Content: string(source)}}, 4096)
	if err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	if len(raw) > 32000 {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("book metadata result too long")
	}
	var metadata contracts.BookMetadata
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(raw)))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&metadata); err != nil {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("invalid book metadata JSON: %w", err)
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return contracts.AIActionPreviewDTO{}, fmt.Errorf("invalid trailing book metadata")
	}
	if err = contracts.ValidateBookMetadata(kind, metadata, state); err != nil {
		return contracts.AIActionPreviewDTO{}, err
	}
	snapshot, _ := json.Marshal(state)
	args, _ := json.Marshal(tools.BookMetadataArgs{Kind: kind, ID: id, Metadata: metadata, Snapshot: string(snapshot)})
	sessionID := newAgentID("act_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: tools.UpdateBookMetadataName, Args: args, SessionID: sessionID, Channel: core.ChannelAction, Sanitize: core.SanitizeFull})
	if result.Error != nil {
		return contracts.AIActionPreviewDTO{}, result.Error
	}
	return contracts.AIActionPreviewDTO{Action: prompts.ActionRecognizeBookMetadata, Name: tools.UpdateBookMetadataName, SessionID: sessionID, OriginalText: state.Title, ProposedText: metadata.TranslatedTitle, Changes: confirmChangeDTOs(result.Changes), ConfirmToken: result.ConfirmToken, ExpiresAt: result.ExpiresAt, Arguments: args, Noop: result.ConfirmToken == ""}, nil
}
