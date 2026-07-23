import { beforeEach, describe, expect, it } from "vitest"
import type { ActorMergeAuditDTO } from "@/api/types"
import {
  ACTOR_MERGE_STORAGE_KEY,
  emptyLocalActorMergeState,
  loadLocalActorMergeState,
  saveLocalActorMergeState,
} from "@/lib/actor-merge-local-storage"

function audit(): ActorMergeAuditDTO {
  return {
    id: "amrg_test",
    sourceActorId: 1,
    targetActorId: 2,
    sourceName: "Source",
    targetName: "Target",
    previewToken: "token",
    appliedAt: "2026-07-21T00:00:00Z",
    summary: {
      movies: { sourceCount: 1, targetCount: 1, duplicateCount: 0, resultCount: 2 },
      userTags: ["tag"],
      externalLinks: ["https://example.com"],
      aliases: ["Source"],
      recommendationFeedback: {
        sourceCount: 0,
        targetCount: 0,
        duplicateCount: 0,
        resultCount: 0,
      },
      curatedFramesAffected: 0,
      profileDecisions: { summary: "target" },
    },
  }
}

describe("actor merge localStorage", () => {
  beforeEach(() => localStorage.clear())

  it("round-trips aliases and audits without sharing mutable arrays", () => {
    const state = {
      aliases: { source: { alias: "Source", canonicalName: "Target" } },
      audits: [audit()],
    }
    saveLocalActorMergeState(state, localStorage)
    const loaded = loadLocalActorMergeState(localStorage)
    expect(loaded).toEqual(state)
    loaded.audits[0]?.summary.aliases.push("changed")
    expect(loadLocalActorMergeState(localStorage).audits[0]?.summary.aliases).toEqual(["Source"])
  })

  it("falls back safely for malformed or future schemas", () => {
    localStorage.setItem(ACTOR_MERGE_STORAGE_KEY, "not-json")
    expect(loadLocalActorMergeState(localStorage)).toEqual(emptyLocalActorMergeState())
    localStorage.setItem(
      ACTOR_MERGE_STORAGE_KEY,
      JSON.stringify({ schemaVersion: 2, aliases: {}, audits: [] }),
    )
    expect(loadLocalActorMergeState(localStorage)).toEqual(emptyLocalActorMergeState())
    localStorage.setItem(
      ACTOR_MERGE_STORAGE_KEY,
      JSON.stringify({ schemaVersion: 1, aliases: { source: 123 }, audits: [] }),
    )
    expect(loadLocalActorMergeState(localStorage)).toEqual(emptyLocalActorMergeState())
  })

  it("rekeys persisted aliases with the current Unicode comparison form", () => {
    localStorage.setItem(
      ACTOR_MERGE_STORAGE_KEY,
      JSON.stringify({
        schemaVersion: 1,
        aliases: {
          "straße": { alias: "Straße", canonicalName: "Canonical" },
        },
        audits: [],
      }),
    )

    expect(loadLocalActorMergeState(localStorage).aliases).toEqual({
      strasse: { alias: "Straße", canonicalName: "Canonical" },
    })
  })
})
