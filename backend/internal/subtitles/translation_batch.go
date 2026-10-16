package subtitles

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/llm"
)

// TranslationLimits describe request capacity, not a fixed number of cues.
// OutputBudget is an estimate used for grouping, never sent as max_tokens.
type TranslationLimits struct {
	ContextWindow int
	OutputBudget  int
}

func (l TranslationLimits) normalized() TranslationLimits {
	if l.ContextWindow <= 0 {
		l.ContextWindow = 65536
	}
	if l.OutputBudget <= 0 {
		l.OutputBudget = 32768
	}
	l.OutputBudget = min(l.OutputBudget, l.ContextWindow/2)
	return l
}

// TranslationInput is shared by capacity estimation and the actual API request.
func TranslationInput(cues []Cue) ([]byte, error) {
	type inputCue struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}
	input := make([]inputCue, 0, len(cues))
	for _, c := range cues {
		input = append(input, inputCue{ID: c.ID, Text: c.Text})
	}
	return json.Marshal(input)
}

func translationCost(c Cue) (input, output int) {
	raw, _ := TranslationInput([]Cue{c})
	// Input uses the same conservative UTF-8 byte estimate as agent context
	// budgeting. Output estimates one token per source character plus JSON/ID
	// overhead; actual expansion is handled by explicit truncation fallback.
	return len(raw), utf8.RuneCountInString(c.Text) + 16
}

func translationBatches(cues []Cue, limits TranslationLimits) [][]Cue {
	limits = limits.normalized()
	overhead := len(prompts.SubtitleTranslationPrompt().Text) + 1024
	requestLimit := limits.ContextWindow - limits.ContextWindow/20
	var batches [][]Cue
	start, input, output := 0, overhead, 0
	for i, c := range cues {
		in, out := translationCost(c)
		if i > start && (input+in+output+out > requestLimit || output+out > limits.OutputBudget) {
			batches = append(batches, cues[start:i])
			start, input, output = i, overhead, 0
		}
		input += in
		output += out
	}
	if start < len(cues) {
		batches = append(batches, cues[start:])
	}
	return batches
}

func translationCapacityError(err error) bool {
	return errors.Is(err, llm.ErrOutputTruncated) || llm.IsContextOverflow(err)
}
