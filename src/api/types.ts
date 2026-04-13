export interface ApiResponse<T> {
  ok: boolean
  data?: T
  error?: ApiError
}

export interface ApiError {
  code: string
  message: string
  retryable: boolean
  details?: Record<string, unknown>
}

export interface HealthDTO {
  name: string
  /** 构建戳：`YYYYMMDD.HHMMSS`（UTC，来自 Git vcs.time 或 CI `-X BuildStamp`）；无则可能 `git.<hash>` / `unknown` */
  version: string
  /** 构建通道：`dev` / `release`；旧后端可能缺省 */
  channel?: string
  /** 正式打包版本号；开发态通常缺省 */
  installerVersion?: string
  transport: string
  databasePath: string
}

export interface DevPerformanceSummaryDTO {
  supported: boolean
  sampledAt?: string
  systemCpuPercent?: number
  backendCpuPercent?: number
}

export interface MovieListItemDTO {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  /** 元数据/刮削标签 */
  tags: string[]
  /** 用户本地标签（与 tags 独立，刮削不覆盖） */
  userTags?: string[]
  runtimeMinutes: number
  rating: number
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  /** 发行日 YYYY-MM-DD，无则省略 */
  releaseDate?: string
  coverUrl?: string
  thumbUrl?: string
  /** 回收站条目为 RFC3339；在库中通常省略 */
  trashedAt?: string
}

export interface MovieDetailDTO extends MovieListItemDTO {
  summary: string
  previewImages?: string[]
  previewVideoUrl?: string
  /** 刮削/站点评分（movies.rating） */
  metadataRating: number
  /** 用户本地评分（movies.user_rating），无覆盖时省略 */
  userRating?: number | null
  /** 演员展示名 -> 头像 URL（SQLite actors.avatar，依赖演员资料刮削） */
  actorAvatarUrls?: Record<string, string>
}

export interface MoviesPageDTO {
  items: MovieListItemDTO[]
  total: number
  limit: number
  offset: number
}

export interface LibraryPathDTO {
  id: string
  path: string
  title: string
  /** 新添加的库根在首次成功扫描完成前为 true */
  firstLibraryScanPending?: boolean
}

export type HardwareEncoderPreference =
  | "auto"
  | "amf"
  | "qsv"
  | "nvenc"
  | "videotoolbox"
  | "software"

export type NativePlayerPreset = "mpv" | "potplayer" | "custom"

export interface PlayerSettingsDTO {
  hardwareDecode: boolean
  hardwareEncoder?: HardwareEncoderPreference
  nativePlayerPreset?: NativePlayerPreset
  nativePlayerEnabled: boolean
  nativePlayerCommand?: string
  streamPushEnabled: boolean
  forceStreamPush?: boolean
  ffmpegCommand?: string
  preferNativePlayer: boolean
  seekForwardStepSec: number
  seekBackwardStepSec: number
}

export interface PatchPlayerSettingsBody {
  hardwareDecode?: boolean
  hardwareEncoder?: HardwareEncoderPreference
  nativePlayerPreset?: NativePlayerPreset
  nativePlayerEnabled?: boolean
  nativePlayerCommand?: string
  streamPushEnabled?: boolean
  forceStreamPush?: boolean
  ffmpegCommand?: string
  preferNativePlayer?: boolean
  seekForwardStepSec?: number
  seekBackwardStepSec?: number
}

/** 与后端 library-config.cfg / GET settings 一致：决定新刮削使用的策略；链与单源列表可保留在切换模式时 */
export type MetadataMovieScrapeMode = "auto" | "specified" | "chain"
export type MetadataMovieStrategy =
  | "auto-global"
  | "auto-cn-friendly"
  | "custom-chain"
  | "specified"

export interface SettingsDTO {
  libraryPaths: LibraryPathDTO[]
  player: PlayerSettingsDTO
  /** 扫描后整理为 番号/番号.ext 并写入 NFO/资产到番号目录 */
  organizeLibrary: boolean
  /**
   * 开启后，在新加入的库根「第一次成功扫描」时，会尝试识别 Curated 清单或外部整理目录布局（仅标注，不改动已有库路径的默认行为）。
   */
  extendedLibraryImport: boolean
  /** 为 true 时库根目录监听新文件并防抖触发扫描（及后续刮削）；与主配置 libraryWatchEnabled 共同生效 */
  autoLibraryWatch: boolean
  autoActorProfileScrape: boolean
  launchAtLogin: boolean
  launchAtLoginSupported: boolean
  /** 空字符串表示自动（全源加权）；非空为 Metatube 影片源注册名 */
  metadataMovieProvider: string
  /** 当前引擎可用的影片源名（排序），供指定模式选择 */
  metadataMovieProviders: string[]
  /** 有序的 Provider 列表；优先于 metadataMovieProvider 使用。空数组表示自动（全源） */
  metadataMovieProviderChain: string[]
  /** 当前生效的刮削策略（旧后端可能缺省，由前端按链/单源推断） */
  metadataMovieScrapeMode?: MetadataMovieScrapeMode
  metadataMovieStrategy?: MetadataMovieStrategy
  /** HTTP 代理配置 */
  proxy: ProxySettingsDTO
  /** 后端进程日志（文件 + 级别）；重启后端后作用于 Zap */
  backendLog: BackendLogSettingsDTO
}

export interface ProxySettingsDTO {
  enabled: boolean
  url?: string
  username?: string
  password?: string
}

/** 后端日志目录与级别（library-config.cfg）；空 logDir 表示使用当前构建的默认日志目录 */
export interface BackendLogSettingsDTO {
  logDir: string
  logFilePrefix?: string
  logMaxAgeDays?: number
  logLevel?: string
}

/** PATCH backendLog 的字段；省略表示不修改 */
export interface PatchBackendLogBody {
  logDir?: string
  logFilePrefix?: string
  logMaxAgeDays?: number
  logLevel?: string
}

/** POST /api/proxy/ping-javbus | ping-google — optional body to test draft proxy without saving */
export interface ProxyJavBusPingRequestBody {
  proxy?: ProxySettingsDTO
}

export interface ProxyJavBusPingResponse {
  ok: boolean
  latencyMs: number
  httpStatus?: number
  message?: string
}

export interface PatchSettingsBody {
  organizeLibrary?: boolean
  extendedLibraryImport?: boolean
  autoLibraryWatch?: boolean
  autoActorProfileScrape?: boolean
  launchAtLogin?: boolean
  player?: PatchPlayerSettingsBody
  /** 未发送则不改；发送 "" 恢复自动；非空须为服务端认可的 provider 名 */
  metadataMovieProvider?: string
  /** 有序的 Provider 列表。发送空数组表示清除（自动模式）。优先级高于 metadataMovieProvider */
  metadataMovieProviderChain?: string[]
  /** 仅切换生效策略，不删除已保存的链或单源字段 */
  metadataMovieScrapeMode?: MetadataMovieScrapeMode
  metadataMovieStrategy?: MetadataMovieStrategy
  /** 代理配置；发送则替换当前配置 */
  proxy?: ProxySettingsDTO
  /** 合并写入后端日志设置 */
  backendLog?: PatchBackendLogBody
}

export interface TaskDTO {
  taskId: string
  type: string
  status: "pending" | "running" | "completed" | "partial_failed" | "failed" | "cancelled"
  createdAt: string
  startedAt?: string
  finishedAt?: string
  progress: number
  message?: string
  errorCode?: string
  errorCategory?: string
  errorMessage?: string
  provider?: string
  metadata?: Record<string, unknown>
}

export interface RecentTasksDTO {
  tasks: TaskDTO[]
}

export interface ActorProfileDTO {
  name: string
  avatarUrl?: string
  avatarRemoteUrl?: string
  avatarLocalUrl?: string
  hasLocalAvatar?: boolean
  summary?: string
  homepage?: string
  provider?: string
  providerActorId?: string
  height?: number
  birthday?: string
  profileUpdatedAt?: string
  /** 演员维度用户标签，与 ActorListItemDTO.userTags 同源 */
  userTags?: string[]
}

/** GET /library/actors 单行；userTags 为演员维度用户标签，与影片 tag 无关 */
export interface ActorListItemDTO {
  name: string
  avatarUrl?: string
  avatarRemoteUrl?: string
  avatarLocalUrl?: string
  hasLocalAvatar?: boolean
  movieCount: number
  userTags?: string[]
}

export interface ActorsListDTO {
  total: number
  actors: ActorListItemDTO[]
}

export interface ListActorsParams {
  /** 子串匹配演员名或演员用户标签（不区分大小写） */
  q?: string
  /** 精确匹配演员用户标签（勿与影片路由 tag= 混用） */
  actorTag?: string
  sort?: "name" | "movieCount"
  limit?: number
  offset?: number
}

export interface ListMoviesParams {
  mode?: string
  q?: string
  /** 精确演员名，与路由 `actor` 一致 */
  actor?: string
  /** 精确厂商名，与路由 `studio` 一致 */
  studio?: string
  limit?: number
  offset?: number
}

export interface StartScanBody {
  paths?: string[]
}

/** POST /library/metadata-scrape — 仅允许已配置的库根路径 */
export interface MetadataScrapeByPathsBody {
  paths: string[]
}

export interface MetadataRefreshQueuedDTO {
  queued: number
  skipped: number
  invalidPaths: string[]
}

export interface AddLibraryPathBody {
  path: string
  title?: string
}

/** POST /library/paths：与 LibraryPathDTO 同字段，成功启动初次扫描时带 scanTask */
export interface AddLibraryPathResultDTO extends LibraryPathDTO {
  scanTask?: TaskDTO
}

export interface UpdateLibraryPathBody {
  title: string
}

/** PATCH /library/movies/{id}；rating 为 null 表示清除用户评分；userTags / metadataTags 出现时整表替换对应标签（NFO 与「我的标签」互不影响） */
export interface PatchMovieBody {
  isFavorite?: boolean
  rating?: number | null
  userTags?: string[]
  /** 元数据/NFO 类标签整表替换；空数组表示清空本地 NFO 标签（下次刮削会再写入） */
  metadataTags?: string[]
  /** 展示用标题覆盖；null 或省略且配合清空语义时由后端清除 user_title，恢复刮削值 */
  userTitle?: string | null
  userStudio?: string | null
  userSummary?: string | null
  /** YYYY-MM-DD；null 清除 user_release_date */
  userReleaseDate?: string | null
  /** 分钟；null 清除 user_runtime_minutes */
  userRuntimeMinutes?: number | null
}

/** GET /playback/progress */
export interface PlaybackProgressItemDTO {
  movieId: string
  positionSec: number
  durationSec: number
  updatedAt: string
}

export interface PlaybackProgressListDTO {
  items: PlaybackProgressItemDTO[]
}

/** Provider health status */
export type ProviderHealthStatus = "ok" | "degraded" | "fail"

/** GET /api/providers/ping or POST /api/providers/ping-all single item */
export interface ProviderHealthDTO {
  name: string
  status: ProviderHealthStatus
  latencyMs: number
  message?: string
  errorCategory?: string
  cooldownUntil?: string
  consecutiveFailures?: number
  avgLatencyMs?: number
}

/** POST /api/providers/ping request body */
export interface PingProviderRequest {
  name: string
}

/** POST /api/providers/ping-all response */
export interface PingAllProvidersResponse {
  providers: ProviderHealthDTO[]
  total: number
  ok: number
  fail: number
}

/** PUT /playback/progress/{movieId} */
export interface PutPlaybackProgressBody {
  positionSec: number
  durationSec: number
}

export type PlaybackMode = "direct" | "hls" | "native"

export interface PlaybackAudioTrackDTO {
  id: string
  label: string
  default: boolean
}

export interface PlaybackSubtitleTrackDTO {
  id: string
  label: string
  kind?: string
  default: boolean
}

export interface PlaybackDescriptorDTO {
  movieId: string
  mode: PlaybackMode
  sessionId?: string
  sessionKind?: string
  url: string
  mimeType?: string
  fileName?: string
  transcodeProfile?: string
  durationSec?: number
  startPositionSec?: number
  resumePositionSec?: number
  canDirectPlay: boolean
  reason?: string
  reasonCode?: string
  reasonMessage?: string
  sourceContainer?: string
  sourceVideoCodec?: string
  sourceAudioCodec?: string
  audioTracks?: PlaybackAudioTrackDTO[]
  subtitleTracks?: PlaybackSubtitleTrackDTO[]
}

export interface CreatePlaybackSessionBody {
  mode?: PlaybackMode
  startPositionSec?: number
}

export interface NativePlaybackLaunchDTO {
  ok: boolean
  command?: string
  target?: string
  mode?: string
  message?: string
  movieId?: string
  startedAt?: string
}

/** 与后端 contracts.MaxMovieCommentRunes 一致 */
export const MAX_MOVIE_COMMENT_RUNES = 10000

/** GET/PUT /library/movies/{movieId}/comment（每部一条可覆盖） */
export interface MovieCommentDTO {
  body: string
  updatedAt: string
}

export interface PutMovieCommentBody {
  body: string
}

/** GET /curated-frames（无图像字节，图用 GET /curated-frames/{id}/image） */
export interface CuratedFrameItemDTO {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  tags: string[]
}

export interface ListCuratedFramesParams {
  q?: string
  actor?: string
  movieId?: string
  tag?: string
  limit?: number
  offset?: number
}

export interface CuratedFramesListDTO {
  items: CuratedFrameItemDTO[]
  total: number
  limit: number
  offset: number
}

/** POST /curated-frames */
export interface CreateCuratedFrameBody {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  tags?: string[]
  imageBase64?: string
}

/** PATCH /curated-frames/{id}/tags */
export interface PatchCuratedFrameTagsBody {
  tags: string[]
}

export interface CuratedFrameStatsDTO {
  total: number
}

export interface CuratedFrameFacetItemDTO {
  name: string
  count: number
}

export interface CuratedFrameFacetListDTO {
  items: CuratedFrameFacetItemDTO[]
}

/** POST /curated-frames/export → WebP/PNG 单文件或 ZIP */
export interface PostCuratedFramesExportBody {
  ids: string[]
  /** 按演员分组导出时传入；须属于每帧的 actors */
  actorName?: string
  /** 默认 webp；png 为带 iTXt 元数据的 PNG */
  format?: "webp" | "png"
}

/** GET /library/played-movies */
export interface PlayedMoviesListDTO {
  movieIds: string[]
}
