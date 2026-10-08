import { mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { afterEach, expect, it } from "vitest"
import { PlaybackPreferencesStore } from "./playback-preferences"

const directories: string[] = []
function directory() {
  const value = mkdtempSync(path.join(tmpdir(), "curated-playback-preferences-"))
  directories.push(value)
  return value
}
afterEach(() => { for (const value of directories.splice(0)) rmSync(value, { recursive: true, force: true }) })
it("uses the installation default only when preferences have never been saved", () => {
  const folder = directory()
  const store = new PlaybackPreferencesStore(folder, true)
  expect(store.read().preferNative).toBe(true)
  store.write({ preferNative: false, volume: 38, speed: 1.5 })
  expect(new PlaybackPreferencesStore(folder, true).read()).toEqual({ preferNative: false, volume: 38, speed: 1.5 })
})
it("keeps the existing Web default for other installations", () => {
  expect(new PlaybackPreferencesStore(directory()).read().preferNative).toBe(false)
})
