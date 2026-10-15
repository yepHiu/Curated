/** Shared pure subtitle parsing and display contract. No renderer or Node dependencies. */
export type SubtitleDisplayMode = "off" | "zh-CN" | "ja" | "bilingual"
export interface SubtitleDisplay { mode: SubtitleDisplayMode; size: number; background: boolean; position: number; offset: number }
export const defaultSubtitleDisplay: SubtitleDisplay = { mode: "zh-CN", size: 28, background: true, position: 8, offset: 0 }
export interface SubtitleCue { begin: number; end: number; text: string; maxEnd: number }
export function validateSubtitleDisplay(value: SubtitleDisplay): SubtitleDisplay {
  if (!value || !["off", "zh-CN", "ja", "bilingual"].includes(value.mode) || typeof value.background !== "boolean"
    || !Number.isFinite(value.size) || value.size < 16 || value.size > 64
    || !Number.isFinite(value.position) || value.position < 0 || value.position > 40
    || !Number.isFinite(value.offset) || Math.abs(value.offset) > 60) throw new Error("INVALID_SUBTITLE_SETTINGS")
  return { mode: value.mode, size: value.size, background: value.background, position: value.position, offset: value.offset }
}
function stamp(raw: string): number {
  const m = raw.trim().match(/^(\d{2,}):(\d{2}):(\d{2})[,.](\d{3})$/)
  if (!m || Number(m[2]) > 59 || Number(m[3]) > 59) return NaN
  return Number(m[1]) * 3600 + Number(m[2]) * 60 + Number(m[3]) + Number(m[4]) / 1000
}
export function parseSRT(raw: string): SubtitleCue[] {
  if (raw.length > 16 * 1024 * 1024) throw new Error("SUBTITLE_TOO_LARGE")
  const cues: SubtitleCue[] = []
  for (const block of raw.replace(/^\uFEFF/, "").replace(/\r/g, "").split(/\n\s*\n/)) {
    const lines = block.trim().split("\n")
    const index = lines.findIndex(line => line.includes("-->"))
    if (index < 0) continue
    const timing = lines[index]!.split("-->")
    const begin = stamp(timing[0] ?? ""), end = stamp(timing[1] ?? "")
    const text = lines.slice(index + 1).join("\n").replace(/<[^>]*>/g, "").replace(/\{[^}]*\}/g, "").trim()
    if (Number.isFinite(begin) && end > begin && text) cues.push({ begin, end, text, maxEnd: end })
  }
  cues.sort((a, b) => a.begin - b.begin)
  let max = 0
  for (const cue of cues) { max = Math.max(max, cue.end); cue.maxEnd = max }
  return cues
}
export function activeSubtitleText(cues: SubtitleCue[], time: number): string {
  let lo = 0, hi = cues.length
  while (lo < hi) { const mid = (lo + hi) >>> 1; if (cues[mid]!.begin <= time) lo = mid + 1; else hi = mid }
  const result: string[] = []
  for (let i = lo - 1; i >= 0 && cues[i]!.maxEnd > time; i--) { const cue = cues[i]!; if (cue.end > time) result.unshift(cue.text) }
  return result.join("\n")
}
function srtStamp(sec: number): string {
  const ms = Math.round(sec * 1000)
  return `${String(Math.floor(ms / 3600000)).padStart(2, "0")}:${String(Math.floor(ms / 60000) % 60).padStart(2, "0")}:${String(Math.floor(ms / 1000) % 60).padStart(2, "0")},${String(ms % 1000).padStart(3, "0")}`
}
export function mergeSubtitleSRT(chinese: string, japanese: string): string {
  const zh = parseSRT(chinese), ja = parseSRT(japanese)
  const originals = new Map(ja.map(cue => [`${cue.begin}:${cue.end}`, cue.text]))
  return zh.map((cue, index) => `${index + 1}\n${srtStamp(cue.begin)} --> ${srtStamp(cue.end)}\n${cue.text}\n${originals.get(`${cue.begin}:${cue.end}`) ?? ""}\n`).join("\n")
}
export function plainSubtitleSRT(raw: string): string {
  return parseSRT(raw).map((cue, index) => `${index + 1}\n${srtStamp(cue.begin)} --> ${srtStamp(cue.end)}\n${cue.text}\n`).join("\n")
}
