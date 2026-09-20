package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"curated-backend/internal/agent/run"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

// prepareAIHistory keeps the visible transcript immutable and advances a
// separate checkpoint only after successful summarization and persistence.
func (a *App) prepareAIHistory(ctx context.Context, sessionID string, streamer llm.Streamer, emit func(contracts.AIChatSSEEvent), contextWindow ...int) ([]llm.ChatMessage, error) {
	windowTokens := 0
	if len(contextWindow) > 0 {
		windowTokens = contextWindow[0]
	}
	budget := run.BudgetForContext(windowTokens)
	rows, err := a.store.ListAIChatContext(ctx, sessionID, 200)
	if err != nil {
		return nil, err
	}
	var history []llm.ChatMessage
	var seqs []int
	for _, row := range rows {
		if strings.TrimSpace(row.Content) == "" {
			continue
		}
		history = append(history, llm.ChatMessage{Role: row.Role, Content: row.Content})
		seqs = append(seqs, row.Seq)
	}
	if len(history) == 0 {
		return nil, nil
	}
	window := run.RecentHistory(history, windowTokens)
	boundary := seqs[len(history)-len(window)]
	through, summary, err := a.store.AIContextCheckpoint(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	status := func(phase string) {
		emit(contracts.AIChatSSEEvent{Type: "context_status", Context: &contracts.AIContextStatusDTO{Phase: phase}})
	}
	started, limited := false, false
	// Legacy backfill and every summary are cancellable; avoid unbounded work
	// before answering when opening a very large old conversation.
	compactCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for through < boundary-1 {
		older, err := a.store.AIContextRange(compactCtx, sessionID, through, boundary)
		if err != nil {
			limited = true
			break
		}
		if len(older) == 0 {
			break
		}
		if !started {
			status("compacting")
			started = true
		}
		var batch []llm.ChatMessage
		lastSeq := through
		for _, row := range older {
			// Large historical messages are explicitly excerpted, never current input.
			message := llm.ChatMessage{Role: row.Role, Content: run.MemoryExcerpt(row.Content, 8*1024)}
			// JSON escaping can multiply byte cost. Bound a single message by
			// the encoded request, so unusual text cannot block backfill forever.
			previousJSON, _ := json.Marshal(summary)
			for {
				raw, _ := json.Marshal(message)
				if len(raw)+len(previousJSON)+128 <= budget.SummaryInput || len(message.Content) <= 256 {
					break
				}
				message.Content = run.MemoryExcerpt(message.Content, len(message.Content)/2)
			}
			candidate := append(append([]llm.ChatMessage(nil), batch...), message)
			raw, _ := json.Marshal(candidate)
			if len(raw)+len(previousJSON)+128 > budget.SummaryInput && len(batch) > 0 {
				break
			}
			batch = candidate
			lastSeq = row.Seq
		}
		next, err := run.SummarizeMemory(compactCtx, streamer, summary, batch, windowTokens)
		if err != nil {
			limited = true
			break
		}
		if err := a.store.SaveAIContextCheckpoint(compactCtx, sessionID, lastSeq, next); err != nil {
			limited = true
			break
		}
		through, summary = lastSeq, next
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if limited {
		status("limited")
		summary += "\nSome older messages could not be included in this checkpoint. Do not assume missing constraints or completed work; retrieve exact facts or ask for clarification when necessary."
	} else if started {
		status("ready")
	}
	if summary != "" {
		return append([]llm.ChatMessage{run.MemoryMessage(summary)}, window...), nil
	}
	return window, nil
}
