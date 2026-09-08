import { describe, expect, it } from "vitest"
import type { AIChatOutcomeDTO } from "@/api/types"
import { confirmationOutcome } from "./confirmation-outcome"
import { restoreChatHistory } from "./restore-history"
import type { AgentChatEntry } from "./types"

const waiting: AIChatOutcomeDTO = { status: "needs_confirmation", reasonCode: "confirmation_required" }
const legacy: AIChatOutcomeDTO = { status: "completed", reason: "A write preview is waiting for UI confirmation." }
const card = (status: Extract<AgentChatEntry, { kind: "confirm" }>["status"]): AgentChatEntry => ({
  id: "confirm", kind: "confirm", name: "create_saved_view", status, sessionId: "s", confirmToken: "", arguments: {}, changes: [],
})

describe("confirmation outcome feedback", () => {
  it.each([waiting, legacy])("tracks confirmation state for %j", (outcome) => {
    for (const [state, status, reasonCode] of [
      ["pending", "needs_confirmation", "confirmation_required"],
      ["applying", "needs_confirmation", "confirmation_required"],
      ["applied", "completed", "write_applied"],
      ["discarded", "cancelled", "confirmation_discarded"],
      ["archived", "needs_input", "confirmation_unavailable"],
    ] as const) {
      expect(confirmationOutcome([card(state)], 1, outcome)).toEqual({ status, reasonCode })
    }
  })

  it("never replaces an error or borrows a prior turn's confirmation", () => {
    const failed: AIChatOutcomeDTO = { status: "failed", reason: "network" }
    expect(confirmationOutcome([card("applied")], 1, failed)).toBe(failed)
    expect(confirmationOutcome([card("applied"), { id: "user", kind: "user", content: "next" }], 2, waiting)).toBe(waiting)
  })

  it.each([true, false])("restores legacy history using the receipt, applied=%s", (applied) => {
    const entries = restoreChatHistory([{
      id: "message", sessionId: "s", role: "assistant", content: "", seq: 1, createdAt: "",
      events: [{ type: "confirm_required", name: "create_saved_view", applied }, { type: "message_done", outcome: legacy }],
    }])
    const index = entries.findIndex(e => e.kind === "outcome")
    expect(confirmationOutcome(entries, index, legacy).reasonCode).toBe(applied ? "write_applied" : "confirmation_unavailable")
    expect(entries.find(e => e.kind === "confirm")).toMatchObject({ confirmToken: "", arguments: {} })
  })
})
