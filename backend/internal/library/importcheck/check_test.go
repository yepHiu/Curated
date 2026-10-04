package importcheck

import "testing"

func TestCheckMatchesExactAndSimilarLibraryCodes(t *testing.T) {
	t.Parallel()

	index := []IndexItem{
		{ID: "ssis-001", Code: "SSIS-001", Title: "Exact title"},
		{ID: "ssis-002-cd1", Code: "SSIS-002-CD1", Title: "Disc one"},
		{ID: "abc-999", Code: "ABC-999", Title: "Unrelated"},
	}
	items := Check([]string{
		"489155.com@SSIS-001-C.mp4",
		"folder\\SSIS-002.mkv",
		"holiday.mp4",
		"  ",
	}, index)
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].ExtractedCode != "SSIS-001" || len(items[0].Matches) != 1 || items[0].Matches[0].MatchKind != "exact" {
		t.Fatalf("exact item = %+v", items[0])
	}
	if items[1].ExtractedCode != "SSIS-002" || len(items[1].Matches) != 1 || items[1].Matches[0].MatchKind != "similar" {
		t.Fatalf("similar item = %+v", items[1])
	}
	if items[2].ExtractedCode != "" || len(items[2].Matches) != 0 {
		t.Fatalf("unrecognized item = %+v", items[2])
	}
}

// TestValidateNames 验证本次浏览类别及文件归属的兼容行为。
func TestValidateNames(t *testing.T) {
	t.Parallel()
	if got := ValidateNames(nil); got == "" {
		t.Fatal("expected error for empty names")
	}
	if got := ValidateNames([]string{"  "}); got == "" {
		t.Fatal("expected error for blank names")
	}
	tooMany := make([]string, MaxNames+1)
	for i := range tooMany {
		tooMany[i] = "ABC-001.mp4"
	}
	if got := ValidateNames(tooMany); got == "" {
		t.Fatal("expected error for too many names")
	}
	if got := ValidateNames([]string{"ABC-001.mp4"}); got != "" {
		t.Fatalf("unexpected error %q", got)
	}
}

// TestCheckDistinguishesAdditionalPart preserves movie matches while reporting file-level novelty.
func TestCheckDistinguishesAdditionalPart(t *testing.T) {
	index := []IndexItem{{ID: "fc2-1234567", Code: "FC2-1234567", PartIndexes: []int{1, 3}}}
	items := Check([]string{"FC2-1234567-CD1.mp4", "FC2-1234567_2.mp4", "FC2-1234567.mp4"}, index)
	for i, want := range []string{"part-exists", "new-part", "same-code"} {
		if items[i].FileStatus != want {
			t.Fatalf("item=%+v want=%s", items[i], want)
		}
	}
}
