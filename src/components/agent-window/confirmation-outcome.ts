import type { AIChatOutcomeDTO } from "@/api/types"
import type { AgentChatEntry } from "./types"

/** Keep the terminal feedback in sync with this turn's confirmation card. */
export function confirmationOutcome(entries: AgentChatEntry[], index: number, outcome: AIChatOutcomeDTO): AIChatOutcomeDTO {
  const legacy = outcome.status === "completed" && outcome.reason === "A write preview is waiting for UI confirmation."
  if (outcome.status !== "needs_confirmation" && !legacy) return outcome
  for (let i = index - 1; i >= 0; i--) {
    const entry = entries[i]
    if (!entry || entry.kind === "user" || entry.kind === "outcome") break
    if (entry.kind !== "confirm") continue
    switch (entry.status) {
      case "applied": return { status: "completed", reasonCode: "write_applied" }
      case "archived": return { status: "needs_input", reasonCode: "confirmation_unavailable" }
      case "discarded": return { status: "cancelled", reasonCode: "confirmation_discarded" }
      default: return { status: "needs_confirmation", reasonCode: "confirmation_required" }
    }
  }
  return outcome
}
