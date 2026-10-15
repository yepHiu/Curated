export type RecognitionModel = "qwen-audio-3.1-asr-flash-filetrans" | "qwen3-asr-flash-filetrans"
export type SubtitleModel = "configured-ai" | "qwen-flash" | "qwen-plus"
export type SubtitleMode = "off" | "zh-CN" | "ja" | "bilingual"
export interface SubtitleSettings {
  recognitionModel?: RecognitionModel
  enabled: boolean
  apiKeyConfigured: boolean
  translationModel: SubtitleModel
  configuredModel: string
  canManage: boolean
}
export interface SubtitleAsset { id: string; fileId: string; name: string; language: string; format: string; size: number; hash: string; missing: boolean }
export interface SubtitleJob {
  id: string; movieId: string; fileId: string; fileName: string
  state: string; stage: string; translationModel: SubtitleModel
  completed: number; total: number; error?: string; expiresAt?: string
  createdAt: string; updatedAt: string; assets: SubtitleAsset[]
}
export interface SubtitlePreferences { mode: SubtitleMode; size: number; background: boolean; position: number; offset: number }
export interface SubtitleService {
  revealKey(): Promise<string>
  audioTracks(movieId: string, fileId?: string): Promise<{ index: number; language: string; title: string; codec: string; channels: number }[]>
  testProvider(): Promise<void>
  settings(): Promise<SubtitleSettings>
  saveSettings(patch: { recognitionModel?: RecognitionModel; enabled?: boolean; apiKey?: string; translationModel?: SubtitleModel }): Promise<SubtitleSettings>
  jobs(movieId: string): Promise<SubtitleJob[]>
  create(movieId: string, request: { requestId: string; fileId?: string; translationModel: SubtitleModel; audioTrack: number }): Promise<SubtitleJob>
  action(id: string, action: "cancel" | "retry"): Promise<SubtitleJob>
  assets(movieId: string, fileId?: string): Promise<SubtitleAsset[]>
  content(id: string): Promise<string>
  reveal(id: string): Promise<void>
}
