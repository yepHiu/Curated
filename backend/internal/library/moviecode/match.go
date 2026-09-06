package moviecode

import "strings"

const (
	MatchNone    = ""
	MatchExact   = "exact"
	MatchSimilar = "similar"
)

func compactStorageID(normalized string) string {
	return strings.ReplaceAll(normalized, "-", "")
}

func kindRank(kind string) int {
	switch kind {
	case MatchExact:
		return 2
	case MatchSimilar:
		return 1
	default:
		return 0
	}
}

// StrongerKind returns the more specific match kind.
func StrongerKind(a, b string) string {
	if kindRank(a) >= kindRank(b) {
		return a
	}
	return b
}

// Classify compares an incoming catalog code with a library id or code.
// Exact covers normalized equality and hyphen-stripped equality (SSIS-001 vs SSIS001).
// Similar covers hyphen-delimited prefix variants (SSIS-001 vs SSIS-001-CD1).
func Classify(incoming, existing string) string {
	a := NormalizeForStorageID(incoming)
	b := NormalizeForStorageID(existing)
	if a == "" || b == "" {
		return MatchNone
	}
	if a == b {
		return MatchExact
	}
	if compactStorageID(a) == compactStorageID(b) {
		return MatchExact
	}
	if strings.HasPrefix(a, b+"-") || strings.HasPrefix(b, a+"-") {
		return MatchSimilar
	}
	return MatchNone
}
