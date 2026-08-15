/// <reference types="node" />
import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { describe, expect, it } from "vitest"
import { MESSAGE_POLICIES, type MessagePolicyId } from "./message-policy"

function parseCatalogIds(csv: string): string[] {
  const lines = csv.split(/\r?\n/).filter((line) => line.trim().length > 0)
  return lines.slice(1).map((line) => {
    const first = line.split(",")[0]?.replace(/^\ufeff/, "").trim() ?? ""
    return first
  })
}

describe("message policy catalog", () => {
  it("keeps runtime policy IDs aligned with the CSV ledger", () => {
    const csv = readFileSync(resolve(process.cwd(), "docs/prd/message-catalog.csv"), "utf8")
    const catalogIds = parseCatalogIds(csv)
    const runtimeIds = Object.keys(MESSAGE_POLICIES)

    expect(runtimeIds.sort()).toEqual([...catalogIds].sort())
  })

  it("does not let notify policies light the badge", () => {
    const notifyIds = (Object.keys(MESSAGE_POLICIES) as MessagePolicyId[]).filter(
      (id) => MESSAGE_POLICIES[id].level === "notify",
    )
    expect(notifyIds.length).toBeGreaterThan(0)
    for (const id of notifyIds) {
      expect(MESSAGE_POLICIES[id].badge).toBe("none")
      expect(MESSAGE_POLICIES[id].center).toBe("recent")
    }
  })
})
