package eval

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunExecutesDeterministicCasesInOrder(t *testing.T) {
	t.Parallel()
	var seen []string
	err := Run(context.Background(), []Case{
		{ID: "EVAL-R0-001", Run: func(context.Context) error { seen = append(seen, "one"); return nil }},
		{ID: "EVAL-R0-002", Run: func(context.Context) error { seen = append(seen, "two"); return nil }},
	})
	if err != nil || strings.Join(seen, ",") != "one,two" {
		t.Fatalf("err=%v seen=%v", err, seen)
	}
}

func TestRunAnnotatesCaseFailure(t *testing.T) {
	t.Parallel()
	err := Run(context.Background(), []Case{{ID: "EVAL-R0-004", Run: func(context.Context) error { return errors.New("tool failed") }}})
	if err == nil || !strings.Contains(err.Error(), "EVAL-R0-004") {
		t.Fatalf("err=%v", err)
	}
}
