package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/storage"
)

// topicMatch 引用输入字段中的真实片段，而非模型自报的置信度。
type topicMatch struct {
	Topic string `json:"topic"`
	Field string `json:"field"`
	Quote string `json:"quote"`
}

// topicClassification 是模型唯一可输出的影片决策结构。
type topicClassification struct {
	MovieID string       `json:"movieId"`
	Matches []topicMatch `json:"matches"`
	Reason  string       `json:"reason"`
}

// decodeTopicJSON 拒绝未知字段、尾随 JSON 和无界模型响应。
func decodeTopicJSON(raw string, out any) error {
	if len(raw) > 128*1024 {
		return fmt.Errorf("topic response too large")
	}
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("invalid topic response: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing topic response")
	}
	return nil
}

// validateTopicVocabulary 校验题材和别名，并拒绝含混的跨题材别名映射。
func validateTopicVocabulary(defs []storage.TopicDefinition) error {
	owners := map[string]string{}
	for _, d := range defs {
		if strings.TrimSpace(d.Name) != d.Name || d.Name == "" || utf8.RuneCountInString(d.Name) > 64 || len(d.Description) > 1000 || len(d.Aliases) > 20 {
			return fmt.Errorf("invalid topic vocabulary")
		}
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			key := strings.ToLower(strings.TrimSpace(name))
			if key == "" || utf8.RuneCountInString(name) > 64 {
				return fmt.Errorf("invalid topic alias")
			}
			if old, ok := owners[key]; ok && old != d.Name {
				return fmt.Errorf("ambiguous topic alias")
			}
			owners[key] = d.Name
		}
	}
	return nil
}

// validateTopicClassification 检验范围、规范题材及可复核证据；不允许模型额外写字段。
func validateTopicClassification(results []topicClassification, inputs []storage.TopicMovieInput, defs []storage.TopicDefinition) (map[string][]string, error) {
	if len(results) != len(inputs) {
		return nil, fmt.Errorf("incomplete classification")
	}
	byID := map[string]storage.TopicMovieInput{}
	for _, v := range inputs {
		byID[v.MovieID] = v
	}
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	out := map[string][]string{}
	for _, v := range results {
		input, ok := byID[v.MovieID]
		if !ok {
			return nil, fmt.Errorf("out of scope movie")
		}
		if _, ok = out[v.MovieID]; ok {
			return nil, fmt.Errorf("duplicate movie")
		}
		if len(v.Matches) > 12 || len(v.Reason) > 1000 {
			return nil, fmt.Errorf("unbounded classification")
		}
		out[v.MovieID] = []string{}
		seen := map[string]bool{}
		for _, m := range v.Matches {
			if !names[m.Topic] || strings.TrimSpace(m.Quote) == "" {
				return nil, fmt.Errorf("unsupported topic")
			}
			supported := false
			switch m.Field {
			case "title":
				supported = strings.Contains(input.Title, m.Quote)
			case "summary":
				supported = strings.Contains(input.Summary, m.Quote)
			case "metadataTags":
				for _, tag := range input.MetadataTags {
					if tag == m.Quote {
						supported = true
					}
				}
			}
			if !supported {
				return nil, fmt.Errorf("invalid evidence")
			}
			if !seen[m.Topic] {
				out[v.MovieID] = append(out[v.MovieID], m.Topic)
				seen[m.Topic] = true
			}
		}
	}
	return out, nil
}

// topicVocabularySample is a compact proposal-only view; classification always reads full source fields.
type topicVocabularySample struct {
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	SummaryExcerpt bool     `json:"summaryExcerpt,omitempty"`
	MetadataTags   []string `json:"metadataTags"`
	UserTags       []string `json:"userTags"`
}

// compactVocabularySample limits repeated source input without silently truncating classification evidence.
func compactVocabularySample(input storage.TopicMovieInput) topicVocabularySample {
	summary := []rune(input.Summary)
	excerpt := len(summary) > 600
	if excerpt {
		summary = summary[:600]
	}
	return topicVocabularySample{Title: input.Title, Summary: string(summary), SummaryExcerpt: excerpt, MetadataTags: input.MetadataTags, UserTags: input.UserTags}
}
