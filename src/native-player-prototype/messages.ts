const zh = {
  title: "Windows 原生播放原型", subtitle: "原始媒体 · mpv", prepare: "连接与引擎", engine: "选择 mpv.exe",
  ready: "引擎已就绪", missing: "尚未选择引擎", server: "Server 地址", connect: "连接", connected: "已连接",
  locked: "Server 已锁定", pin: "解锁 PIN", unlock: "解锁", media: "影片与文件", query: "搜索番号或标题",
  search: "搜索", movie: "影片", selectMovie: "选择影片", file: "文件", selectFile: "选择文件", noMovies: "没有匹配的影片",
  start: "起播位置（秒）", play: "原生播放", pause: "暂停", resume: "继续", stop: "停止", seek: "跳转位置（秒）",
  jump: "跳转", speed: "倍速", volume: "音量", apply: "应用", playback: "播放控制", diagnostics: "播放诊断",
  time: "进度", codec: "视频编码", hwdec: "硬件解码", dropped: "呈现掉帧", decoderDropped: "解码掉帧",
  idle: "待播放", starting: "正在启动", playing: "播放中", paused: "已暂停", ended: "播放结束", stopped: "已停止", error: "播放失败",
  failure: "操作失败", unavailable: "请通过 Desktop 原型命令打开此窗口", saveFailed: "进度保存失败，下次保存时会重试",
  nativeWindow: "视频在独立原生窗口播放", busy: "处理中", engineRequired: "请先选择 mpv.exe", lockedHint: "请先连接并解锁 Server",
}
export type NativeMessage = keyof typeof zh
const en: Record<NativeMessage, string> = {
  title: "Windows native playback prototype", subtitle: "Original media · mpv", prepare: "Connection and engine", engine: "Select mpv.exe",
  ready: "Engine ready", missing: "Engine not selected", server: "Server address", connect: "Connect", connected: "Connected",
  locked: "Server locked", pin: "Unlock PIN", unlock: "Unlock", media: "Movie and file", query: "Search code or title",
  search: "Search", movie: "Movie", selectMovie: "Select a movie", file: "File", selectFile: "Select a file", noMovies: "No matching movies",
  start: "Start position (seconds)", play: "Play natively", pause: "Pause", resume: "Resume", stop: "Stop", seek: "Seek position (seconds)",
  jump: "Seek", speed: "Speed", volume: "Volume", apply: "Apply", playback: "Playback controls", diagnostics: "Playback diagnostics",
  time: "Progress", codec: "Video codec", hwdec: "Hardware decoding", dropped: "Presentation drops", decoderDropped: "Decoder drops",
  idle: "Idle", starting: "Starting", playing: "Playing", paused: "Paused", ended: "Ended", stopped: "Stopped", error: "Playback failed",
  failure: "Action failed", unavailable: "Open this window with the Desktop prototype command", saveFailed: "Progress save failed; the next save will retry",
  nativeWindow: "Video plays in a separate native window", busy: "Working", engineRequired: "Select mpv.exe first", lockedHint: "Connect and unlock the Server first",
}
const ja: Record<NativeMessage, string> = {
  title: "Windows ネイティブ再生プロトタイプ", subtitle: "元のメディア · mpv", prepare: "接続とエンジン", engine: "mpv.exe を選択",
  ready: "エンジン準備完了", missing: "エンジン未選択", server: "Server アドレス", connect: "接続", connected: "接続済み",
  locked: "Server はロック中", pin: "解除 PIN", unlock: "解除", media: "作品とファイル", query: "品番またはタイトルを検索",
  search: "検索", movie: "作品", selectMovie: "作品を選択", file: "ファイル", selectFile: "ファイルを選択", noMovies: "一致する作品がありません",
  start: "開始位置（秒）", play: "ネイティブ再生", pause: "一時停止", resume: "再開", stop: "停止", seek: "移動先（秒）",
  jump: "移動", speed: "再生速度", volume: "音量", apply: "適用", playback: "再生操作", diagnostics: "再生診断",
  time: "進捗", codec: "映像コーデック", hwdec: "ハードウェアデコード", dropped: "表示ドロップ", decoderDropped: "デコードドロップ",
  idle: "待機中", starting: "起動中", playing: "再生中", paused: "一時停止中", ended: "再生終了", stopped: "停止済み", error: "再生失敗",
  failure: "操作失敗", unavailable: "Desktop のプロトタイプコマンドでこの画面を開いてください", saveFailed: "進捗の保存失敗。次回保存時に再試行します",
  nativeWindow: "映像は別のネイティブウィンドウで再生されます", busy: "処理中", engineRequired: "mpv.exe を先に選択してください", lockedHint: "Server に接続して解除してください",
}
export const nativeMessages = { "zh-CN": zh, en, ja }
