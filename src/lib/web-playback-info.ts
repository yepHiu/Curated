import type { PlaybackInfoGroup, PlaybackInfoRow } from "./playback-info"
import { nativePlaybackInfoMessages, type NativeInfoLanguage } from "./native-playback-info"

const fields = {
  "zh-CN": {
    state: "播放状态", mode: "媒体传输", "session-kind": "会话类型", file: "文件", mime: "MIME 类型",
    "transcode-profile": "转码器", "encoder-speed": "编码速度", "written-window": "已写入时长", "last-seek": "最近跳转", "reason-code": "原因代码", reason: "原因",
    "source-format": "源格式", current: "当前 / 总时长", progress: "播放进度", resume: "续播位置", volume: "音量", pip: "画中画", "direct-play": "支持直放",
    "audio-tracks": "音轨数量", "subtitle-tracks": "字幕轨数量", audio: "音频码率", "video-current": "当前视频码率", resolution: "分辨率", fps: "帧率", "video-variant": "变体目标码率", bandwidth: "估算带宽", session: "会话与转码",
  },
  en: {
    state: "Status", mode: "Media transport", "session-kind": "Session type", file: "File", mime: "MIME type",
    "transcode-profile": "Transcoder", "encoder-speed": "Encoder speed", "written-window": "Written duration", "last-seek": "Last seek", "reason-code": "Reason code", reason: "Reason",
    "source-format": "Source format", current: "Current / total", progress: "Progress", resume: "Resume position", volume: "Volume", pip: "Picture-in-picture", "direct-play": "Direct play supported",
    "audio-tracks": "Audio tracks", "subtitle-tracks": "Subtitle tracks", audio: "Audio bitrate", "video-current": "Current video bitrate", resolution: "Resolution", fps: "Frame rate", "video-variant": "Variant target bitrate", bandwidth: "Estimated bandwidth", session: "Session and transcoding",
  },
  ja: {
    state: "再生状態", mode: "メディア転送", "session-kind": "セッション種別", file: "ファイル", mime: "MIME タイプ",
    "transcode-profile": "トランスコーダー", "encoder-speed": "エンコード速度", "written-window": "書き込み済み時間", "last-seek": "直近のシーク", "reason-code": "理由コード", reason: "理由",
    "source-format": "元の形式", current: "現在 / 合計時間", progress: "再生進捗", resume: "再開位置", volume: "音量", pip: "ピクチャーインピクチャー", "direct-play": "直接再生対応",
    "audio-tracks": "音声トラック数", "subtitle-tracks": "字幕トラック数", audio: "音声ビットレート", "video-current": "現在の映像ビットレート", resolution: "解像度", fps: "フレームレート", "video-variant": "バリアント目標ビットレート", bandwidth: "推定帯域幅", session: "セッションと変換",
  },
}
type FieldKey = Exclude<keyof typeof fields.en, "session">
export interface WebPlaybackInfoRow { key: string; label: string; value: string }
const measurements = new Set(["progress", "volume", "audio", "video-current", "fps", "video-variant", "bandwidth", "audio-tracks", "subtitle-tracks", "written-window", "current", "resume"])

export function webPlaybackInfoGroups(rows: WebPlaybackInfoRow[], lang: NativeInfoLanguage): PlaybackInfoGroup[] {
  const labels = nativePlaybackInfoMessages[lang]
  const localize = (row: WebPlaybackInfoRow): PlaybackInfoRow => {
    const value = row.value === "N/A" ? "—" : row.key === "state" ? row.value.toLowerCase() : row.value
    const result: PlaybackInfoRow = { label: fields[lang][row.key as FieldKey] ?? row.label, value }
    if (measurements.has(row.key)) {
      const values = value.split(" / ")
      result.metrics = values.map(part => {
        const match = part.match(/^([+-]?[\d.:]+|—)\s*(.*)$/)
        return { number: match?.[1] ?? part, unit: match?.[2] ?? "", reserveUnit: ["progress", "volume", "audio", "video-current", "fps", "video-variant", "bandwidth"].includes(row.key) }
      })
    }
    return result
  }
  const group = (title: string, keys: string[]): PlaybackInfoGroup => ({ title, rows: rows.filter(row => keys.includes(row.key)).map(localize) })
  return [
    group(labels.playback, ["state", "mode", "file", "mime", "current", "progress", "resume", "volume", "pip", "direct-play"]),
    group(labels.video, ["source-format", "resolution", "fps", "video-current", "video-variant", "subtitle-tracks"]),
    group(labels.audio, ["audio-tracks", "audio"]),
    group(fields[lang].session, ["session-kind", "transcode-profile", "encoder-speed", "written-window", "last-seek", "reason-code", "reason", "bandwidth"]),
  ].filter(group => group.rows.length)
}

export function playbackInfoText(title: string, groups: PlaybackInfoGroup[]): string {
  return [title, ...groups.flatMap(group => ["", group.title, ...group.rows.map(row => `${row.label}: ${row.value}`)])].join("\n")
}
