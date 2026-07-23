import { beforeEach, describe, expect, it, vi } from "vitest"
import { ACTOR_MERGE_STORAGE_KEY } from "@/lib/actor-merge-local-storage"

async function loadMockService() {
  const module = await import("@/services/adapters/mock/mock-library-service")
  return module.mockLibraryService
}

describe("mock actor canonical merges", () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
  })

  it("previews, applies, resolves aliases, persists audits, and survives reload", async () => {
    const service = await loadMockService()
    const preview = await service.previewActorMerge({
      sourceName: "Mina Kaze",
      targetName: "Rin Asuka",
    })
    expect(preview.canApply).toBe(true)
    expect(preview.previewToken).not.toBe("")
    expect(preview.movies.sourceCount).toBeGreaterThan(0)

    const audit = await service.applyActorMerge({
      sourceName: preview.source.name,
      targetName: preview.target.name,
      previewToken: preview.previewToken,
      confirm: true,
    })
    expect(audit.sourceName).toBe("Mina Kaze")
    expect(audit.targetName).toBe("Rin Asuka")
    await expect(service.getActorProfile("Mina Kaze")).resolves.toMatchObject({
      name: "Rin Asuka",
    })
    expect((await service.listActors({ q: "Mina Kaze" })).actors).toEqual([
      expect.objectContaining({ name: "Rin Asuka" }),
    ])
    await expect(service.listActorMergeAudits()).resolves.toMatchObject({ total: 1 })
    expect(localStorage.getItem(ACTOR_MERGE_STORAGE_KEY)).toContain("Mina Kaze")

    vi.resetModules()
    const reloaded = await loadMockService()
    await expect(reloaded.getActorProfile("Mina Kaze")).resolves.toMatchObject({
      name: "Rin Asuka",
    })
    await expect(reloaded.listActorMergeAudits()).resolves.toMatchObject({ total: 1 })
  })

  it("rejects stale previews after actor state changes", async () => {
    const service = await loadMockService()
    const preview = await service.previewActorMerge({
      sourceName: "Airi Sena",
      targetName: "Emi Kisaragi",
    })
    await service.patchActorUserTags("Airi Sena", ["changed-after-preview"])

    await expect(
      service.applyActorMerge({
        sourceName: preview.source.name,
        targetName: preview.target.name,
        previewToken: preview.previewToken,
        confirm: true,
      }),
    ).rejects.toMatchObject({ apiError: { code: "ACTOR_MERGE_STALE_PREVIEW" } })
  })
})
