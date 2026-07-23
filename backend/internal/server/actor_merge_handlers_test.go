package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"curated-backend/internal/contracts"
)

type stubActorMergeProvider struct {
	preview     contracts.ActorMergePreviewDTO
	previewErr  error
	audit       contracts.ActorMergeAuditDTO
	applyErr    error
	audits      contracts.ActorMergeAuditListDTO
	auditsErr   error
	previewBody contracts.ActorMergePreviewRequest
	applyBody   contracts.ApplyActorMergeRequest
}

func (s *stubActorMergeProvider) PreviewActorMerge(_ context.Context, req contracts.ActorMergePreviewRequest) (contracts.ActorMergePreviewDTO, error) {
	s.previewBody = req
	return s.preview, s.previewErr
}

func (s *stubActorMergeProvider) ApplyActorMerge(_ context.Context, req contracts.ApplyActorMergeRequest) (contracts.ActorMergeAuditDTO, error) {
	s.applyBody = req
	return s.audit, s.applyErr
}

func (s *stubActorMergeProvider) ListActorMergeAudits(_ context.Context, limit, offset int) (contracts.ActorMergeAuditListDTO, error) {
	result := s.audits
	result.Limit = limit
	result.Offset = offset
	return result, s.auditsErr
}

func TestActorMergeHandlersPreviewApplyAndList(t *testing.T) {
	provider := &stubActorMergeProvider{
		preview: contracts.ActorMergePreviewDTO{
			PreviewToken: "preview-token",
			Source:       contracts.ActorMergeActorRefDTO{ID: 1, Name: "Source"},
			Target:       contracts.ActorMergeActorRefDTO{ID: 2, Name: "Target"},
			CanApply:     true,
		},
		audit: contracts.ActorMergeAuditDTO{ID: "amrg_test", SourceName: "Source", TargetName: "Target"},
		audits: contracts.ActorMergeAuditListDTO{
			Items: []contracts.ActorMergeAuditDTO{{ID: "amrg_test"}},
			Total: 1,
		},
	}
	handler := NewHandler(Deps{ActorMergeProvider: provider}).Routes()

	previewRequest := httptest.NewRequest(http.MethodPost, "/api/library/actors/merge-preview", bytes.NewBufferString(`{"sourceName":"Source","targetName":"Target"}`))
	previewRecorder := httptest.NewRecorder()
	handler.ServeHTTP(previewRecorder, previewRequest)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", previewRecorder.Code, previewRecorder.Body.String())
	}
	if provider.previewBody.SourceName != "Source" || provider.previewBody.TargetName != "Target" {
		t.Fatalf("preview request mismatch: %+v", provider.previewBody)
	}

	applyRequest := httptest.NewRequest(http.MethodPost, "/api/library/actors/merge", bytes.NewBufferString(`{"sourceName":"Source","targetName":"Target","previewToken":"preview-token","confirm":true,"profileDecisions":{"summary":"target"}}`))
	applyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}
	if !provider.applyBody.Confirm || provider.applyBody.ProfileDecisions["summary"] != "target" {
		t.Fatalf("apply request mismatch: %+v", provider.applyBody)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/library/actors/merge-audits?limit=12&offset=3", nil)
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var list contracts.ActorMergeAuditListDTO
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || list.Limit != 12 || list.Offset != 3 {
		t.Fatalf("unexpected audit list: %+v", list)
	}
}

func TestActorMergeHandlersRejectUnknownFieldsAndMapStableErrors(t *testing.T) {
	provider := &stubActorMergeProvider{}
	handler := NewHandler(Deps{ActorMergeProvider: provider}).Routes()

	badRequest := httptest.NewRequest(http.MethodPost, "/api/library/actors/merge-preview", bytes.NewBufferString(`{"sourceName":"Source","targetName":"Target","unexpected":true}`))
	badRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badRecorder, badRequest)
	assertActorMergeErrorCode(t, badRecorder, http.StatusBadRequest, contracts.ErrorCodeActorMergeInvalid)

	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "not found", err: contracts.ErrActorMergeNotFound, code: contracts.ErrorCodeActorMergeNotFound},
		{name: "source alias", err: contracts.ErrActorMergeSourceAlias, code: contracts.ErrorCodeActorMergeSourceAlias},
		{name: "self", err: contracts.ErrActorMergeSelf, code: contracts.ErrorCodeActorMergeSelf},
		{name: "stale", err: contracts.ErrActorMergeStalePreview, code: contracts.ErrorCodeActorMergeStalePreview},
		{name: "conflict", err: contracts.ErrActorMergeConflict, code: contracts.ErrorCodeActorMergeConflict},
		{name: "link limit", err: contracts.ErrActorMergeLinkLimit, code: contracts.ErrorCodeActorMergeLinkLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider.applyErr = test.err
			request := httptest.NewRequest(http.MethodPost, "/api/library/actors/merge", bytes.NewBufferString(`{"sourceName":"Source","targetName":"Target","previewToken":"token","confirm":true}`))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			status := http.StatusConflict
			if errors.Is(test.err, contracts.ErrActorMergeNotFound) {
				status = http.StatusNotFound
			}
			assertActorMergeErrorCode(t, recorder, status, test.code)
		})
	}
}

func assertActorMergeErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, status, recorder.Body.String())
	}
	var response contracts.AppError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != code {
		t.Fatalf("error code=%q want=%q body=%s", response.Code, code, recorder.Body.String())
	}
}
