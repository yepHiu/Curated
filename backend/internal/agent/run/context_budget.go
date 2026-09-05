package run

import (
	"curated-backend/internal/llm"
	"encoding/json"
)

const historyTextBudget = 24 * 1024
const requestTokenEstimateBudget = 64 * 1024

// Byte counts are a conservative, tokenizer-independent estimate, not measured
// provider usage. Never split JSON tool results or the latest user request.
func boundedHistory(history []llm.ChatMessage) ([]llm.ChatMessage, bool) {
	start := len(history)
	bytes := 0
	for start > 0 && len(history)-start < maxRecentMessages {
		cost := len(history[start-1].Content) + 8
		if start < len(history) && bytes+cost > historyTextBudget {
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

func estimatedRequestTokens(messages []llm.ChatMessage, tools []llm.ToolSpec) int {
	encoded, err := json.Marshal(struct {
		Messages []llm.ChatMessage `json:"messages"`
		Tools    []llm.ToolSpec    `json:"tools"`
	}{messages, tools})
	if err != nil {
		return requestTokenEstimateBudget + 1
	}
	return len(encoded)
}
