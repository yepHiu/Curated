package app

import (
	"testing"

	"curated-backend/internal/contracts"
)

func TestHistoricalConfirmationOutcomes(t *testing.T) {
	for _, status := range []string{"completed", "needs_confirmation", "failed", "cancelled"} {
		for _, applied := range []bool{true, false} {
			original := &contracts.AIChatOutcomeDTO{Status: status, Reason: "A write preview is waiting for UI confirmation."}
			messages := []contracts.AIChatStoredMessageDTO{{Events: []contracts.AIChatSSEEvent{
				{Type: "confirm_required", Applied: applied},
				{Type: "message_done", Outcome: original},
			}}}
			projectAIConfirmationOutcomes(messages)
			got := messages[0].Events[1].Outcome
			if status == "failed" || status == "cancelled" {
				if got != original {
					t.Fatalf("overrode %s outcome", status)
				}
			} else if applied && (got.Status != "completed" || got.ReasonCode != "write_applied") {
				t.Fatalf("applied history: %+v", got)
			} else if !applied && (got.Status != "needs_input" || got.ReasonCode != "confirmation_unavailable") {
				t.Fatalf("unapplied history: %+v", got)
			}
			if original.Status != status {
				t.Fatal("mutated recorded generation outcome")
			}
		}
	}
}
