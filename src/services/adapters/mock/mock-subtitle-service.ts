import type { SubtitleService, SubtitleSettings } from "@/services/contracts/subtitle-service"
let settings: SubtitleSettings = { enabled: false, apiKeyConfigured: false, translationModel: "configured-ai", configuredModel: "", canManage: true }
/** Mock never fabricates cloud progress or retains an entered credential. */
export const mockSubtitleService: SubtitleService = {
  async revealKey() { throw new Error("Mock mode: no saved credential") },
  async audioTracks() { return [{ index: 0, language: "ja", title: "", codec: "aac", channels: 2 }] },
  async testProvider() { throw new Error("Mock mode: connect to Server to test credentials") },
  async settings() { return { ...settings } },
  async saveSettings(patch) { settings = { ...settings, enabled: patch.enabled ?? settings.enabled, translationModel: patch.translationModel ?? settings.translationModel, apiKeyConfigured: patch.apiKey === undefined ? settings.apiKeyConfigured : !!patch.apiKey }; return { ...settings } },
  async jobs() { return [] },
  async assets() { return [] },
  async create() { throw new Error("Mock mode: connect to Server to generate subtitles") },
  async action() { throw new Error("Mock mode: no subtitle tasks") },
  async content() { throw new Error("Mock mode: no subtitle files") },
  async reveal() { throw new Error("Mock mode: no subtitle files") },
}
