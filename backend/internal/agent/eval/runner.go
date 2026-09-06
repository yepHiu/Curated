// Package eval provides a tiny deterministic runner for cross-layer Agent
// capability cases. It deliberately has no network or provider dependency;
// callers supply scripted model turns and synthetic library adapters.
package eval

import (
	"context"
	"fmt"
	"strings"
)

// Case is one stable, named Agent behavior contract.
type Case struct {
	ID  string
	Run func(context.Context) error
}

// Run executes cases in declaration order and annotates failures with their
// stable id so CI output identifies the broken behavior contract directly.
func Run(ctx context.Context, cases []Case) error {
	for _, testCase := range cases {
		id := strings.TrimSpace(testCase.ID)
		if id == "" || testCase.Run == nil {
			return fmt.Errorf("invalid agent eval case")
		}
		if err := testCase.Run(ctx); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
	}
	return nil
}
