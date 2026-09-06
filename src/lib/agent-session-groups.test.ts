import { describe, expect, it } from "vitest"
import { groupAgentSessions } from "./agent-session-groups"

function session(id: string, localYear: number, localMonth: number, localDay: number, title = id) {
  const updatedAt = new Date(localYear, localMonth - 1, localDay, 15, 0, 0).toISOString()
  return { id, title, createdAt: updatedAt, updatedAt }
}

describe("groupAgentSessions", () => {
  const now = new Date(2026, 7, 20, 15, 0, 0)

  it("groups by local calendar recency", () => {
    const groups = groupAgentSessions(
      [
        session("today", 2026, 8, 20),
        session("yesterday", 2026, 8, 19),
        session("week", 2026, 8, 16),
        session("old", 2026, 7, 1),
      ],
      now,
    )
    expect(groups.map((group) => group.key)).toEqual(["today", "yesterday", "previous7Days", "older"])
    expect(groups[0]!.items.map((item) => item.id)).toEqual(["today"])
    expect(groups[1]!.items.map((item) => item.id)).toEqual(["yesterday"])
    expect(groups[2]!.items.map((item) => item.id)).toEqual(["week"])
    expect(groups[3]!.items.map((item) => item.id)).toEqual(["old"])
  })

  it("omits empty groups", () => {
    const groups = groupAgentSessions([session("today", 2026, 8, 20)], now)
    expect(groups).toHaveLength(1)
    expect(groups[0]!.key).toBe("today")
  })
})
