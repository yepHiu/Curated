package moviecode

import "testing"

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		incoming string
		existing string
		want     string
	}{
		{"SSIS-001", "SSIS-001", MatchExact},
		{" SSIS_001 ", "ssis-001", MatchExact},
		{"SSIS001", "SSIS-001", MatchExact},
		{"SSIS-001", "ssis001", MatchExact},
		{"SSIS-001", "SSIS-001-CD1", MatchSimilar},
		{"SSIS-001-CD1", "SSIS-001", MatchSimilar},
		{"ABC-12", "ABC-123", MatchNone},
		{"SSIS-001", "SSIS-0012", MatchNone},
		{"", "SSIS-001", MatchNone},
		{"SSIS-001", "", MatchNone},
		{"holiday", "SSIS-001", MatchNone},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.incoming+"/"+tc.existing, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tc.incoming, tc.existing); got != tc.want {
				t.Fatalf("Classify(%q, %q) = %q, want %q", tc.incoming, tc.existing, got, tc.want)
			}
		})
	}
}

func TestStrongerKind(t *testing.T) {
	t.Parallel()
	if got := StrongerKind(MatchSimilar, MatchExact); got != MatchExact {
		t.Fatalf("got %q", got)
	}
	if got := StrongerKind(MatchNone, MatchSimilar); got != MatchSimilar {
		t.Fatalf("got %q", got)
	}
}
