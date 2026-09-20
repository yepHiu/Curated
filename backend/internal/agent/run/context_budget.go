package run

import (
	"curated-backend/internal/config"
	"curated-backend/internal/llm"
	"encoding/json"
)

const historyTextBudget = 24 * 1024

// ContextBudget uses the configured total token window. Until a provider
// tokenizer is available, one serialized UTF-8 byte is charged as one token.
// This is intentionally conservative, not measured provider usage.
type ContextBudget struct {
	Input        int
	Trigger      int
	Output       int
	History      int
	Messages     int
	SummaryInput int
}

func BudgetForContext(window int) ContextBudget {
	if !config.ValidAIContextWindow(window) {
		window = 0
	}
	window = config.EffectiveAIContextWindow(window)
	output := min(32768, window/4)
	input := window - output - window/20
	return ContextBudget{Input: input, Trigger: input * 4 / 5, Output: output,
		History: input / 2, Messages: min(200, maxRecentMessages*window/config.DefaultAIContextWindow),
		SummaryInput: min(MemoryInputBytes, input-2048)}
}

// Byte counts are a conservative, tokenizer-independent estimate, not measured
// provider usage. Never split JSON tool results or the latest user request.
func boundedHistory(history []llm.ChatMessage) ([]llm.ChatMessage, bool) {
	return boundedHistoryWithBudget(history, historyTextBudget, maxRecentMessages)
}

func boundedHistoryWithBudget(history []llm.ChatMessage, budget, maxMessages int) ([]llm.ChatMessage, bool) {
	start := len(history)
	bytes := 0
	for start > 0 && len(history)-start < maxMessages {
		cost := len(history[start-1].Content) + 8
		if start < len(history) && bytes+cost > budget {
			break
		}
		bytes += cost
		start--
	}
	// A retained conversation should not start with a detached assistant answer.
	for start < len(history)-1 && history[start].Role != "user" {
		start++
	}
	return history[start:], start > 0
}

func estimatedRequestBytes(messages []llm.ChatMessage, tools []llm.ToolSpec) int {
	encoded, err := json.Marshal(struct {
		Messages []llm.ChatMessage `json:"messages"`
		Tools    []llm.ToolSpec    `json:"tools"`
	}{messages, tools})
	if err != nil {
		return config.MaxAIContextWindow + 1
	}
	return len(encoded)
}
