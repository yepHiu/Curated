import { mkdirSync, readFileSync, renameSync, writeFileSync } from "node:fs"
import path from "node:path"
import type { DesktopPlaybackPreferences } from "./playback-contract.js"

export const defaultPlaybackPreferences: DesktopPlaybackPreferences = { preferNative: false, volume: 100, speed: 1 }
export function validatePlaybackPreferences(value: unknown): DesktopPlaybackPreferences {
  if (!value || typeof value !== "object") throw new Error("INVALID_PLAYBACK_PREFERENCES")
  const input = value as DesktopPlaybackPreferences
  if (typeof input.preferNative !== "boolean" || !Number.isFinite(input.volume) || input.volume < 0 || input.volume > 100
    || !Number.isFinite(input.speed) || input.speed < 0.25 || input.speed > 4) throw new Error("INVALID_PLAYBACK_PREFERENCES")
  return { preferNative: input.preferNative, volume: input.volume, speed: input.speed }
}
/** 本机偏好不进入 Server / library-config。旧安装默认继续 Web。 */
export class PlaybackPreferencesStore {
  private readonly file: string
  constructor(directory: string) { this.file = path.join(directory, "playback-settings.json") }
  read(): DesktopPlaybackPreferences {
    try { return validatePlaybackPreferences(JSON.parse(readFileSync(this.file, "utf8"))) }
    catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") return { ...defaultPlaybackPreferences }
      throw error
    }
  }
  write(value: DesktopPlaybackPreferences): void {
    const valid = validatePlaybackPreferences(value)
    mkdirSync(path.dirname(this.file), { recursive: true })
    writeFileSync(`${this.file}.tmp`, JSON.stringify(valid, null, 2), { mode: 0o600 })
    renameSync(`${this.file}.tmp`, this.file)
  }
}
