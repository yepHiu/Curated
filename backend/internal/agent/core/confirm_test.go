package core

import (
	"testing"
)

func TestHashArgsIgnoresKeyOrderAndNumberFormat(t *testing.T) {
	t.Parallel()
	left := HashArgs([]byte(`{"name":"未看","filters":{"schemaVersion":1,"playState":"unwatched"}}`))
	right := HashArgs([]byte(`{"filters":{"playState":"unwatched","schemaVersion":1.0},"name":"未看"}`))
	if left != right {
		t.Fatalf("canonical hashes differ:\n%s\n%s", left, right)
	}
}
