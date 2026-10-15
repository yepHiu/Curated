import { httpClient } from "@/api/http-client"
import type { SubtitleService } from "@/services/contracts/subtitle-service"
const encoded = encodeURIComponent
export const webSubtitleService: SubtitleService = {
  revealKey: async () => (await httpClient.post<{ apiKey: string }>("/subtitles/provider/key", {})).apiKey,
  audioTracks: (movieId, fileId) => httpClient.get(`/library/movies/${encoded(movieId)}/subtitle-audio-tracks`, { fileId }),
  testProvider: async () => { await httpClient.post("/subtitles/provider/test", {}) },
  settings: () => httpClient.get("/subtitles/settings"),
  async saveSettings(patch) { await httpClient.patch("/settings", { subtitleProvider: patch }); return this.settings() },
  jobs: movieId => httpClient.get(movieId ? `/library/movies/${encoded(movieId)}/subtitle-jobs` : "/subtitle-jobs"),
  create: (movieId, request) => httpClient.post(`/library/movies/${encoded(movieId)}/subtitle-jobs`, request),
  action: (id, action) => httpClient.post(`/subtitle-jobs/${encoded(id)}/${action}`, {}),
  assets: (movieId, fileId) => httpClient.get(`/library/movies/${encoded(movieId)}/subtitles`, { fileId }),
  content: id => httpClient.getText(`/subtitles/${encoded(id)}/content`),
  reveal: async id => { await httpClient.post(`/subtitles/${encoded(id)}/reveal`, {}) },
}
