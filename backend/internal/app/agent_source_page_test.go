package app

import (
	"context"
	"strings"
	"testing"
)

func TestFetchSourcePageRejectsPrivateAndOffAllowlist(t *testing.T) {
	t.Parallel()
	a := &App{}
	_, err := a.FetchSourcePage(context.Background(), "https://127.0.0.1/secret", []string{"127.0.0.1"})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("loopback = %v", err)
	}
	_, err = a.FetchSourcePage(context.Background(), "https://www.javbus.com/ABC-123", []string{"example.test"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "allowlist") {
		t.Fatalf("off-allowlist host = %v", err)
	}
}
