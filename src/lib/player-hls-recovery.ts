import type { PlaybackDescriptorDTO } from "@/api/types"

export function createHlsRecoveryPolicy() {
  const used = new Set<string>()
  return {
    reset() { used.clear() },
    next(type: string): "network" | "media" | "session" | "failed" {
      const action = type === "networkError" ? "network" : type === "mediaError" ? "media" : "session"
      if (used.has("session")) return "failed"
      if (!used.has(action)) { used.add(action); return action }
      used.add("session")
      return "session"
    },
  }
}

export function canDirectPlaySource(descriptor: PlaybackDescriptorDTO, mp4Codecs: string[]): boolean {
  const name = descriptor.fileName?.toLowerCase() ?? ""
  const video = descriptor.sourceVideoCodec?.toLowerCase()
  const audio = descriptor.sourceAudioCodec?.toLowerCase()
  if (/\.(mp4|m4v)$/.test(name) && video) {
    return mp4Codecs.includes(video) && (!audio || ["aac", "mp3"].includes(audio))
  }
  if (/\.webm$/.test(name) && video) return ["vp8", "vp9", "av1"].includes(video) && (!audio || ["opus", "vorbis"].includes(audio))
  return descriptor.canDirectPlay === true && !video
}
