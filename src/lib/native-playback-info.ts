import type { NativePlayerState } from "../../electron/native-player-contract"

const zh = {
  title: "播放信息", close: "关闭播放信息", copy: "复制信息", save: "保存信息", copied: "播放信息已复制", saved: "播放信息已保存", failed: "操作失败，请重试",
  playback: "播放", video: "视频", audio: "音频", network: "网络与缓存", player: "播放器", transport: "媒体传输", mime: "MIME 类型", container: "容器格式",
  status: "播放状态", speed: "播放倍速", resolution: "源分辨率", fps: "源帧率", displayFps: "显示刷新率", codec: "编码", pixel: "解码像素格式", bitrate: "码率（估算）",
  hwdec: "硬件解码", output: "视频输出", dropped: "呈现 / 解码掉帧", frames: "预计文件帧数", samplerate: "采样率", channels: "声道", avsync: "音画偏差",
  host: "媒体来源", transfer: "接收速度", received: "本次接收流量", requests: "媒体 / Range 请求", cache: "前向缓存", buffer: "缓冲进度", waiting: "等待缓存",
  yes: "是", no: "否", idle: "待播放", starting: "正在启动", playing: "播放中", paused: "已暂停", ended: "播放结束", stopped: "已停止", error: "播放失败",
  history: "最近 30 秒媒体接收速度", software: "软件解码",
}
type InfoKey = keyof typeof zh
const en: Record<InfoKey, string> = {
  title: "Playback information", close: "Close playback information", copy: "Copy information", save: "Save information", copied: "Playback information copied", saved: "Playback information saved", failed: "Action failed; try again",
  playback: "Playback", video: "Video", audio: "Audio", network: "Network and cache", player: "Player", transport: "Media transport", mime: "MIME type", container: "Container",
  status: "Status", speed: "Playback speed", resolution: "Source resolution", fps: "Source frame rate", displayFps: "Display refresh rate", codec: "Codec", pixel: "Decoded pixel format", bitrate: "Bitrate (estimated)",
  hwdec: "Hardware decoding", output: "Video output", dropped: "Presentation / decoder drops", frames: "Estimated file frames", samplerate: "Sample rate", channels: "Channels", avsync: "A/V offset",
  host: "Media source", transfer: "Receive speed", received: "Session received", requests: "Media / Range requests", cache: "Forward cache", buffer: "Buffering", waiting: "Waiting for cache",
  yes: "Yes", no: "No", idle: "Idle", starting: "Starting", playing: "Playing", paused: "Paused", ended: "Ended", stopped: "Stopped", error: "Playback failed",
  history: "Media receive speed over the last 30 seconds", software: "Software decoding",
}
const ja: Record<InfoKey, string> = {
  title: "再生情報", close: "再生情報を閉じる", copy: "情報をコピー", save: "情報を保存", copied: "再生情報をコピーしました", saved: "再生情報を保存しました", failed: "操作に失敗しました。再試行してください",
  playback: "再生", video: "映像", audio: "音声", network: "ネットワークとキャッシュ", player: "プレイヤー", transport: "メディア転送", mime: "MIME タイプ", container: "コンテナ",
  status: "再生状態", speed: "再生速度", resolution: "元の解像度", fps: "元のフレームレート", displayFps: "画面更新頻度", codec: "コーデック", pixel: "デコード後のピクセル形式", bitrate: "ビットレート（推定）",
  hwdec: "ハードウェアデコード", output: "映像出力", dropped: "表示 / デコード落ち", frames: "推定ファイルフレーム数", samplerate: "サンプルレート", channels: "チャンネル", avsync: "音声映像のずれ",
  host: "メディア配信元", transfer: "受信速度", received: "今回の受信量", requests: "メディア / Range 要求", cache: "前方キャッシュ", buffer: "バッファリング", waiting: "キャッシュ待ち",
  yes: "はい", no: "いいえ", idle: "待機中", starting: "起動中", playing: "再生中", paused: "一時停止", ended: "再生終了", stopped: "停止", error: "再生失敗",
  history: "直近 30 秒のメディア受信速度", software: "ソフトウェアデコード",
}
export const nativePlaybackInfoMessages = { "zh-CN": zh, en, ja }
export type NativeInfoLanguage = keyof typeof nativePlaybackInfoMessages
const missing = "—"
export function metric(value: number | null | undefined, unit = "", digits = 2): string {
  return typeof value === "number" && Number.isFinite(value) && value >= 0 ? `${value.toFixed(digits)}${unit ? ` ${unit}` : ""}` : missing
}
export function bytes(value: number | null | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) return missing
  const index = value === 0 ? 0 : Math.min(3, Math.floor(Math.log(value) / Math.log(1024)))
  return metric(value / 1024 ** Math.max(0, index), ["B", "KiB", "MiB", "GiB"][Math.max(0, index)])
}
export function bitrate(value: number | null | undefined): string {
  if (value == null) return missing
  return metric(value / (value >= 1e6 ? 1e6 : 1000), value >= 1e6 ? "Mbps" : "Kbps")
}
export function avOffset(value: number | null | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return missing
  const ms = Number((value * 1000).toFixed(2))
  return `${ms < 0 ? "" : "+"}${ms.toFixed(2)} ms`
}
interface NativeInfoMetric { number: string; unit: string; reserveUnit: boolean }
interface NativeInfoRow { label: string; value: string; metrics?: NativeInfoMetric[] }
/** 固定 30 秒横轴与自适应纵轴，闲置时保留真实零值。 */
export function transferPolyline(history: number[]): string {
  const values = history.slice(-30).map(value => Number.isFinite(value) && value >= 0 ? value : 0)
  const max = Math.max(1, ...values)
  return values.map((value, index) => `${((30 - values.length + index) * 240 / 29).toFixed(1)},${(34 - value / max * 32).toFixed(1)}`).join(" ")
}
export function playbackInfoGroups(state: NativePlayerState | undefined, lang: NativeInfoLanguage) {
  const t = nativePlaybackInfoMessages[lang], d = state?.diagnostics, n = d?.network
  const row = (key: InfoKey, value: string | null | undefined): NativeInfoRow => ({ label: t[key], value: value || missing })
  // 数字与单位独立占位；缺失、整数进位、正负号或单位切换均不改变后续文字的位置。
  const measured = (key: InfoKey, values: string[], reserveUnit = true): NativeInfoRow => ({ ...row(key, values.join(" / ")),
    metrics: values.map(value => {
      const space = value.indexOf(" ")
      return { number: space < 0 ? value : value.slice(0, space), unit: space < 0 ? "" : value.slice(space + 1), reserveUnit }
    }) })
  return [
    { title: t.playback, rows: [row("player", d?.engineVersion ? `mpv · ${d.engineVersion.replace(/^mpv\s+/i, "")}` : "mpv"),
      row("transport", n ? `${n.protocol.toUpperCase()} / Range` : null), row("mime", n?.mime), row("container", d?.container),
      row("status", state ? t[state.status] : null), measured("speed", [metric(state?.speed, "×")])] },
    { title: t.video, rows: [row("resolution", d?.width && d.height ? `${d.width} × ${d.height}` : null), measured("fps", [metric(d?.sourceFps, "fps", 3)]),
      row("codec", state?.codec), row("pixel", d?.pixelFormat), measured("bitrate", [bitrate(d?.videoBitrate)]),
      row("hwdec", state?.hwdec === "no" ? t.software : state?.hwdec), row("output", d?.videoOutput), measured("displayFps", [metric(d?.displayFps, "Hz", 3)]),
      measured("dropped", [metric(state?.droppedFrames, "", 0), metric(state?.decoderDroppedFrames, "", 0)], false), measured("frames", [metric(d?.estimatedFrames, "", 0)], false)] },
    { title: t.audio, rows: [row("codec", d?.audioCodec), measured("samplerate", [metric(d?.audioSampleRate == null ? null : d.audioSampleRate / 1000, "kHz")]),
      row("channels", d?.audioChannels), measured("bitrate", [bitrate(d?.audioBitrate)]), measured("avsync", [avOffset(d?.avSyncSec)])] },
    { title: t.network, rows: [row("host", n?.sourceHost), measured("transfer", [n ? `${bytes(n.bytesPerSec)}/s` : missing]), measured("received", [bytes(n?.receivedBytes)]),
      measured("requests", [metric(n?.requests, "", 0), metric(n?.rangeRequests, "", 0)], false), measured("cache", [metric(d?.cacheDurationSec, "s"), bytes(d?.cacheBytes)]),
      measured("buffer", [metric(d?.bufferingPercent, "%")]), row("waiting", d?.pausedForCache == null ? null : d.pausedForCache ? t.yes : t.no)] },
  ]
}
