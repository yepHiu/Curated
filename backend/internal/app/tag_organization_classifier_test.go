package app

import (
	"curated-backend/internal/storage"
	"testing"
)

// TestTopicEvidenceValidation 防止范围外 ID、无证据分类和 NFO 写入字段进入写链路。
func TestTopicEvidenceValidation(t *testing.T) {
	inputs := []storage.TopicMovieInput{{MovieID: "a", Title: "Theme story", MetadataTags: []string{"Theme"}}}
	defs := []storage.TopicDefinition{{Name: "Theme"}}
	valid := []topicClassification{{MovieID: "a", Matches: []topicMatch{{Topic: "Theme", Field: "metadataTags", Quote: "Theme"}}}}
	if _, err := validateTopicClassification(valid, inputs, defs); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []topicClassification{
		{MovieID: "outside"},
		{MovieID: "a", Matches: []topicMatch{{Topic: "Theme", Field: "summary", Quote: "invented"}}},
		{MovieID: "a", Matches: []topicMatch{{Topic: "Other", Field: "title", Quote: "Theme"}}},
	} {
		if _, err := validateTopicClassification([]topicClassification{tc}, inputs, defs); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	var out struct {
		Movies []topicClassification `json:"movies"`
	}
	for _, raw := range []string{`{"movies":[],"metadataTags":["bad"]}`, `{"movies":[]} {"movies":[]}`} {
		if err := decodeTopicJSON(raw, &out); err == nil {
			t.Fatal("accepted invalid JSON shape")
		}
	}
	if err := validateTopicVocabulary([]storage.TopicDefinition{{Name: "A", Aliases: []string{"same"}}, {Name: "B", Aliases: []string{"same"}}}); err == nil {
		t.Fatal("ambiguous alias accepted")
	}
}
