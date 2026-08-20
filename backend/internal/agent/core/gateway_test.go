package core

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type memoryAudit struct {
	mu      sync.Mutex
	records []AuditRecord
}

func (m *memoryAudit) RecordInvocation(_ context.Context, rec AuditRecord) error {
	m.mu.Lock()
	m.records = append(m.records, rec)
	m.mu.Unlock()
	return nil
}

func boolPtr(v bool) *bool { return &v }

func objectSchema(props map[string]Schema, required ...string) Schema {
	return Schema{Type: "object", Properties: props, Required: required, AdditionalProperties: boolPtr(false)}
}

func TestValidateArgsRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	schema := objectSchema(map[string]Schema{"q": {Type: "string"}})
	if _, err := ValidateArgs(schema, json.RawMessage(`{"q":"a","extra":1}`)); err == nil {
		t.Fatal("expected unknown field error")
	}
	if _, err := ValidateArgs(schema, json.RawMessage(`{"q":"ok"}`)); err != nil {
		t.Fatalf("valid args: %v", err)
	}
}

func TestGatewaySchemaPermissionBudgetConfirmAndAudit(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	readHits := 0
	if err := reg.Register(ToolDefinition{
		Name:         "get_ping",
		Description:  "ping",
		ParamsSchema: objectSchema(nil),
		Permission:   PermissionRead,
		Domain:       DomainQuery,
		Handler: func(ctx context.Context, call Call) (Result, error) {
			readHits++
			return Result{OK: true, Data: map[string]any{"pong": true}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(ToolDefinition{
		Name:         "set_note",
		Description:  "write",
		ParamsSchema: objectSchema(map[string]Schema{"body": {Type: "string"}}, "body"),
		Permission:   PermissionWriteApply,
		Domain:       DomainUserWrite,
		Handler: func(ctx context.Context, call Call) (Result, error) {
			return Result{OK: true, Data: map[string]any{"saved": true}}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	audit := &memoryAudit{}
	gateway := NewGateway(reg, NewConfirmStore(), audit, func() Settings {
		return Settings{WriteEnabled: true, StepLimit: 2, WritePerMinute: 1}
	})

	ok := gateway.Invoke(context.Background(), Call{Name: "get_ping", Args: json.RawMessage(`{}`), SessionID: "s1"})
	if !ok.OK {
		t.Fatalf("read = %+v", ok)
	}
	if loc, _ := json.Marshal(ok.Data); !json.Valid(loc) {
		t.Fatalf("data not json: %v", ok.Data)
	}

	unknown := gateway.Invoke(context.Background(), Call{Name: "get_ping", Args: json.RawMessage(`{"nope":true}`), SessionID: "s1"})
	if unknown.OK || unknown.Error == nil || unknown.Error.Code != "AI_TOOL_INVALID_ARGS" {
		t.Fatalf("unknown field = %+v", unknown)
	}

	missing := gateway.Invoke(context.Background(), Call{Name: "nope", Args: json.RawMessage(`{}`), SessionID: "s1"})
	if missing.OK || missing.Error.Code != "AI_TOOL_NOT_FOUND" {
		t.Fatalf("missing tool = %+v", missing)
	}

	noTok := gateway.Invoke(context.Background(), Call{
		Name: "set_note", Args: json.RawMessage(`{"body":"hi"}`), SessionID: "s2",
	})
	if noTok.OK || noTok.Error.Code != "AI_CONFIRM_REQUIRED" {
		t.Fatalf("missing token = %+v", noTok)
	}

	rec, err := gateway.IssuePreview("s2", "set_note", json.RawMessage(`{"body":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	drift := gateway.Invoke(context.Background(), Call{
		Name: "set_note", Args: json.RawMessage(`{"body":"other"}`), SessionID: "s2", ConfirmTok: rec.Token,
	})
	if drift.OK || drift.Error == nil {
		t.Fatalf("arg drift = %+v", drift)
	}

	rec2, err := gateway.IssuePreview("s2", "set_note", json.RawMessage(`{"body":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	gateway.confirm.now = func() time.Time { return time.Now().Add(20 * time.Minute) }
	expired := gateway.Invoke(context.Background(), Call{
		Name: "set_note", Args: json.RawMessage(`{"body":"hi"}`), SessionID: "s2", ConfirmTok: rec2.Token,
	})
	if expired.OK || expired.Error.Code != "AI_CONFIRM_EXPIRED" {
		t.Fatalf("expired = %+v", expired)
	}
	gateway.confirm.now = time.Now
	gateway.now = time.Now
	gateway.ResetSessionSteps("s2")

	rec3, err := gateway.IssuePreview("s2", "set_note", json.RawMessage(`{"body":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	applied := gateway.Invoke(context.Background(), Call{
		Name: "set_note", Args: json.RawMessage(`{"body":"hi"}`), SessionID: "s2", ConfirmTok: rec3.Token,
	})
	if !applied.OK {
		t.Fatalf("apply = %+v error=%+v", applied, applied.Error)
	}

	limited := gateway.Invoke(context.Background(), Call{
		Name: "get_ping", Args: json.RawMessage(`{}`), SessionID: "limit-session",
	})
	if !limited.OK {
		t.Fatalf("first step = %+v", limited)
	}
	second := gateway.Invoke(context.Background(), Call{
		Name: "get_ping", Args: json.RawMessage(`{}`), SessionID: "limit-session",
	})
	if !second.OK {
		t.Fatalf("second step = %+v", second)
	}
	third := gateway.Invoke(context.Background(), Call{
		Name: "get_ping", Args: json.RawMessage(`{}`), SessionID: "limit-session",
	})
	if third.OK || third.Error.Code != "AI_RATE_LIMITED" {
		t.Fatalf("step limit = %+v", third)
	}

	audit.mu.Lock()
	n := len(audit.records)
	audit.mu.Unlock()
	if n < 6 {
		t.Fatalf("audit records = %d, want at least 6", n)
	}
}

func TestProjectStripsFilesystemFields(t *testing.T) {
	t.Parallel()
	in := map[string]any{"id": "m1", "title": "Hello", "location": "D:\\x.mp4"}
	got := Project(in, SanitizeSanitized).(map[string]any)
	if _, ok := got["location"]; ok {
		t.Fatalf("location leaked: %+v", got)
	}
	if got["title"] != "Hello" {
		t.Fatalf("title = %v", got["title"])
	}
}
