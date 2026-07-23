package app

import (
	"context"
	"time"

	"curated-backend/internal/contracts"
)

func (a *App) PreviewActorMerge(ctx context.Context, req contracts.ActorMergePreviewRequest) (contracts.ActorMergePreviewDTO, error) {
	return a.store.PreviewActorMerge(ctx, req)
}

func (a *App) ApplyActorMerge(ctx context.Context, req contracts.ApplyActorMergeRequest) (contracts.ActorMergeAuditDTO, error) {
	return a.store.ApplyActorMerge(ctx, req, time.Now())
}

func (a *App) ListActorMergeAudits(ctx context.Context, limit, offset int) (contracts.ActorMergeAuditListDTO, error) {
	return a.store.ListActorMergeAudits(ctx, limit, offset)
}
