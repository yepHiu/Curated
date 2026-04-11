/// <reference types="node" />

import { describe, expect, it } from "vitest"
import fs from "node:fs/promises"
import os from "node:os"
import path from "node:path"

import {
  allocateNextPatchInFile,
  readVersionState,
  setVersionBaseInFile,
} from "../../scripts/release/versioning.mjs"

async function makeTempVersionFile(initialState: unknown) {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "curated-release-versioning-"))
  const filePath = path.join(tempDir, "version.json")
  await fs.writeFile(filePath, JSON.stringify(initialState, null, 2), "utf8")
  return { tempDir, filePath }
}

describe("release versioning", () => {
  it("bumps patch and persists the new state", async () => {
    const { tempDir, filePath } = await makeTempVersionFile({
      schema: 1,
      current: {
        major: 1,
        minor: 1,
        patch: 0,
      },
    })

    try {
      const result = await allocateNextPatchInFile(filePath)

      expect(result.version).toBe("1.1.1")

      const nextState = await readVersionState(filePath)
      expect(nextState).toEqual({
        schema: 1,
        current: {
          major: 1,
          minor: 1,
          patch: 1,
        },
      })
    } finally {
      await fs.rm(tempDir, { recursive: true, force: true })
    }
  })

  it("updates major/minor and resets patch to zero", async () => {
    const { tempDir, filePath } = await makeTempVersionFile({
      schema: 1,
      current: {
        major: 1,
        minor: 1,
        patch: 4,
      },
    })

    try {
      const result = await setVersionBaseInFile(filePath, 2, 3)

      expect(result.version).toBe("2.3.0")

      const nextState = await readVersionState(filePath)
      expect(nextState).toEqual({
        schema: 1,
        current: {
          major: 2,
          minor: 3,
          patch: 0,
        },
      })
    } finally {
      await fs.rm(tempDir, { recursive: true, force: true })
    }
  })

  it("rejects malformed version files", async () => {
    const { tempDir, filePath } = await makeTempVersionFile({
      schema: 2,
      current: {
        major: 1,
        minor: 1,
        patch: 0,
      },
    })

    try {
      await expect(readVersionState(filePath)).rejects.toThrow(
        /Unsupported release version schema/,
      )
    } finally {
      await fs.rm(tempDir, { recursive: true, force: true })
    }
  })
})
