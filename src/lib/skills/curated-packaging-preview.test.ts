import { describe, expect, it } from "vitest"
import { buildPackagingPreview } from "./curated-packaging-preview"
import type { PackagingPreviewInput } from "./curated-packaging-preview"

describe("buildPackagingPreview", () => {
  it("predicts installer patch bump without base change", () => {
    const input: PackagingPreviewInput = {
      mode: "installer",
      currentBaseVersion: "1.1.0",
      baseChange: null,
    }

    const preview = buildPackagingPreview(input)

    expect(preview).toEqual({
      mode: "installer",
      currentBaseVersion: "1.1.0",
      predictedVersion: "1.1.1",
      willBumpPatch: true,
      baseVersionAfterChange: "1.1.0",
    })
  })

  it("predicts publish minor base bump and patch bump", () => {
    const input: PackagingPreviewInput = {
      mode: "publish",
      currentBaseVersion: "1.1.0",
      baseChange: { major: null, minor: 2 },
    }

    const preview = buildPackagingPreview(input)

    expect(preview).toEqual({
      mode: "publish",
      currentBaseVersion: "1.1.0",
      predictedVersion: "1.2.1",
      willBumpPatch: true,
      baseVersionAfterChange: "1.2.0",
    })
  })
})
