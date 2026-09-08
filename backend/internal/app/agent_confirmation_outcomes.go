package app

import "curated-backend/internal/contracts"

// History cannot restore confirmation authority. Project the terminal display
// from durable receipts without rewriting the original generation record.
func projectAIConfirmationOutcomes(messages []contracts.AIChatStoredMessageDTO) {
	for i := range messages {
		var confirmation *contracts.AIChatSSEEvent
		for j := range messages[i].Events {
			event := &messages[i].Events[j]
			if event.Type == "confirm_required" {
				confirmation = event
			}
			if event.Type != "message_done" || event.Outcome == nil || confirmation == nil {
				continue
			}
			outcome := event.Outcome
			legacy := outcome.Status == "completed" && outcome.Reason == "A write preview is waiting for UI confirmation."
			if outcome.Status != "needs_confirmation" && !legacy {
				continue
			}
			if confirmation.Applied {
				event.Outcome = &contracts.AIChatOutcomeDTO{Status: "completed", ReasonCode: "write_applied", Reason: "The confirmed changes were saved."}
			} else {
				event.Outcome = &contracts.AIChatOutcomeDTO{Status: "needs_input", ReasonCode: "confirmation_unavailable", Reason: "This historical preview cannot be confirmed. Request a new preview to save the changes."}
			}
		}
	}
}
