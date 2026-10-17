package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func (a *App) StartBookOrganization(ctx context.Context, req contracts.BookOrganizationRequest) (contracts.BookOrganizationJob, error) {
	if err := a.aiPermission(true); err != nil {
		return contracts.BookOrganizationJob{}, err
	}
	if _, err := normalizeAIProviderConfig(a.currentAIProviderConfig()); err != nil {
		return contracts.BookOrganizationJob{}, err
	}
	if strings.TrimSpace(a.currentAIProviderConfig().Model) == "" {
		return contracts.BookOrganizationJob{}, ErrAIProviderNotConfigured
	}
	if req.Locale == "" || req.Locale == "zh" {
		req.Locale = "zh-CN"
	}
	if (req.Kind != "comic" && req.Kind != "photo") || (req.Locale != "zh-CN" && req.Locale != "en" && req.Locale != "ja") || len(req.BookIDs) < 1 || len(req.BookIDs) > 600 {
		return contracts.BookOrganizationJob{}, &core.ToolError{Code: "BAD_REQUEST", Message: "invalid book selection or locale"}
	}
	// Check the enabled library and scope before creating a durable task.
	for _, id := range req.BookIDs {
		if _, err := a.GetBookMetadataState(ctx, req.Kind, id); err != nil {
			return contracts.BookOrganizationJob{}, err
		}
	}
	id, err := a.store.CreateBookOrganization(ctx, newAgentID("book-org-"), req)
	if err != nil {
		return contracts.BookOrganizationJob{}, err
	}
	a.wakeTagOrganization()
	return a.store.GetBookOrganization(ctx, id)
}

func (a *App) ControlBookOrganization(ctx context.Context, id, operation string) error {
	if operation == "retry" {
		if err := a.aiPermission(true); err != nil {
			return err
		}
	}
	if err := a.store.ControlBookOrganization(ctx, id, operation); err != nil {
		return err
	}
	if operation == "cancel" {
		a.tagRT.mu.Lock()
		if a.tagRT.runningID == id && a.tagRT.cancelJob != nil {
			a.tagRT.cancelJob()
		}
		a.tagRT.mu.Unlock()
	} else {
		a.wakeTagOrganization()
	}
	return nil
}

// The existing organization worker owns scheduling, cancellation and restart recovery.
func (a *App) runBookOrganization(ctx context.Context, id string) {
	a.processBookOrganization(ctx, id, func(ctx context.Context, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
		return a.RunAIAction(ctx, prompts.ActionRecognizeBookMetadata, req)
	})
}

func (a *App) processBookOrganization(ctx context.Context, id string, recognize func(context.Context, contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error)) {
	job, err := a.store.GetBookOrganization(ctx, id)
	if err != nil {
		return
	}
	if err = a.store.SetBookOrganizationRunning(ctx, id); err != nil {
		return
	}
	for offset := 0; offset < job.Total; offset += 25 {
		items, err := a.store.BookOrganizationItems(ctx, id, 25, offset)
		if err != nil {
			return
		}
		for _, item := range items {
			if ctx.Err() != nil {
				return
			}
			if item.Status != "pending" {
				continue
			}
			req := contracts.AIActionRequest{TargetLocale: job.Locale}
			if job.Kind == "comic" {
				req.ComicID = item.BookID
			} else {
				req.PhotoID = item.BookID
			}
			preview, err := recognize(ctx, req)
			if ctx.Err() != nil {
				return
			}
			var args tools.BookMetadataArgs
			status, reason := "ready", ""
			if err == nil {
				err = json.Unmarshal(preview.Arguments, &args)
			}
			if err != nil {
				status, reason = "failed", "BOOK_RECOGNITION_FAILED"
				var toolErr *core.ToolError
				if errors.As(err, &toolErr) {
					switch toolErr.Code {
					case "BOOK_METADATA_INVALID_RESULT", "BOOK_METADATA_PROVIDER_FAILED", "BOOK_METADATA_TRUNCATED":
						reason = toolErr.Code
					}
				}
			} else if preview.Noop {
				status = "unchanged"
			}
			if err = a.store.SaveBookOrganizationProposal(ctx, id, item.BookID, status, args.Snapshot, args.Metadata, reason); err != nil {
				return
			}
		}
	}
}

// Confirmation uses the persisted revision and source snapshot, never a model call or an expired ticket.
func (a *App) ReviewBookOrganization(ctx context.Context, id string, req contracts.BookOrganizationReview) error {
	if err := a.aiPermission(true); err != nil {
		return err
	}
	if req.Decision != "apply" && req.Decision != "discard" {
		return &core.ToolError{Code: "BAD_REQUEST", Message: "invalid review decision"}
	}
	job, err := a.store.GetBookOrganization(ctx, id)
	if err != nil {
		return err
	}
	item, err := a.store.BookOrganizationItem(ctx, id, req.BookID)
	if err != nil {
		return err
	}
	if item.Revision != req.Revision {
		return storage.ErrAIWriteConflict
	}
	if (item.Status == "applied" && req.Decision == "apply") || (item.Status == "discarded" && req.Decision == "discard") {
		return nil
	}
	if item.Status != "ready" {
		return storage.ErrAIWriteConflict
	}
	if req.Decision == "discard" {
		return a.store.FinishBookReview(ctx, id, item.BookID, item.Revision, "discarded", "")
	}
	snapshot, _ := json.Marshal(item.Before)
	args, _ := json.Marshal(tools.BookMetadataArgs{Kind: job.Kind, ID: item.BookID, Metadata: item.Metadata, Snapshot: string(snapshot)})
	session := newAgentID("review_")
	result := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: tools.UpdateBookMetadataName, Args: args, SessionID: session, Channel: core.ChannelAction, Sanitize: core.SanitizeFull})
	err = result.Error
	if result.Error == nil {
		if result.ConfirmToken == "" {
			return a.store.FinishBookReview(ctx, id, item.BookID, item.Revision, "unchanged", "")
		}
		_, err = a.ApplyAITool(storage.WithBookOrganizationReview(ctx, id, item.BookID, item.Revision), contracts.AIToolApplyRequest{Name: tools.UpdateBookMetadataName, Arguments: args, SessionID: session, ConfirmToken: result.ConfirmToken})
	}
	if err != nil {
		var toolErr *core.ToolError
		if errors.Is(err, storage.ErrAIWriteConflict) || (errors.As(err, &toolErr) && toolErr.Code == "AI_WRITE_CONFLICT") {
			// A concurrent successful review is a replay, not a source conflict.
			latest, e := a.store.BookOrganizationItem(ctx, id, item.BookID)
			if e == nil && latest.Status == "applied" && latest.Revision == item.Revision {
				return nil
			}
			_ = a.store.FinishBookReview(ctx, id, item.BookID, item.Revision, "conflict", "AI_WRITE_CONFLICT")
		}
	}
	return err
}
