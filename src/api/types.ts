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

export type ConnectedClientDeviceType =
  | "desktop"
  | "laptop"
  | "mobile"
  | "tablet"
  | "tool"
  | "unknown"

export type ConnectedClientAccessKind = "local" | "remote"

export interface ConnectedClientDTO {
  key: string
  ip: string
  port?: number
  hostname?: string
  userAgent?: string
  browser: string
  browserVersion?: string
  os: string
  osVersion?: string
  deviceType: ConnectedClientDeviceType
  accessKind: ConnectedClientAccessKind
  isLocalMachine: boolean
  firstSeen: string
  lastSeen: string
  requestCount: number
}

export interface ConnectedClientsDTO {
  clients: ConnectedClientDTO[]
  total: number
  localCount: number
  remoteCount: number
  sampledAt: string
}

export interface AuthStatusDTO {
  pinEnabled: boolean
  unlocked: boolean
  setupRequired: boolean
  pinLength: number
  sessionExpiresAt?: string
  trustedForever: boolean
  sessionTtlMinutes: number
  lanRequiresPin: boolean
  lockOnRestart: boolean
}

export interface SetupPinBody {
  pin: string
  confirmPin: string
  sessionTtlMinutes?: number
  lockOnRestart?: boolean
  trustedForever?: boolean
}

export interface UnlockPinBody {
  pin: string
  trustedForever?: boolean
}

export interface ChangePinBody {
  currentPin: string
  newPin: string
  confirmPin: string
}

export interface PatchAuthSettingsBody {
  pinEnabled?: boolean
  sessionTtlMinutes?: number
  lockOnRestart?: boolean
}

export interface AuthSessionDTO {
  publicId: string
  userAgent?: string
  ip?: string
  createdAt: string
  lastSeenAt: string
  trustedForever: boolean
  current: boolean
}

export interface AuthSessionsDTO {
  items: AuthSessionDTO[]
}

export interface AppUpdateStatusDTO {
  supported: boolean
  status: "unsupported" | "up-to-date" | "update-available" | "error"
  installedVersion?: string
  latestVersion?: string
  hasUpdate?: boolean
  checkedAt?: string
  publishedAt?: string
  releaseName?: string
  releaseUrl?: string
  installerDownloadUrl?: string
  installerSha256?: string
  artifactStatus?: "downloading" | "downloaded" | "verified" | "failed" | "installing" | "install-launched" | ""
  downloadedVersion?: string
  downloadedFileName?: string
  downloadedBytes?: number
  totalBytes?: number
  downloadProgress?: number
  signatureStatus?: string
  installReady?: boolean
  lastInstallAttemptAt?: string
  lastInstallError?: string
  downloadTaskId?: string
  releaseNotesSnippet?: string
  source?: string
  errorMessage?: string
}

export interface AppUpdateInstallBody {
  mode?: "interactive" | "silent" | "verysilent"
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
  /** 用户本地评分；列表筛选必须与站点/刮削评分区分 */
  userRating?: number | null
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
  /** 写出当前元数据的刮削源；尚未刮削时省略 */
  metadataProvider?: string
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

export type LibraryPathStorageStatus =
  | "online"
  | "offline"
  | "volume_mismatch"
  | "path_missing"
  | "permission_denied"
  | "unknown"

export interface LibraryPathStorageStatusDTO {
  libraryPathId: string
  path: string
  title: string
  status: LibraryPathStorageStatus
  message: string
  checkedAt: string
  rootPath?: string
  driveType?: string
  volumeLabel?: string
  fileSystem?: string
  identityConfidence?: string
  expectedVolumeId?: string
  currentVolumeId?: string
  canRescan: boolean
  canImport: boolean
}

export interface LibraryPathStorageStatusListDTO {
  items: LibraryPathStorageStatusDTO[]
}

export interface CheckLibraryPathStorageStatusBody {
  libraryPathIds?: string[]
}

export interface ComicLibraryPathDTO {
  id: string
  path: string
  title: string
  firstLibraryScanPending?: boolean
}

export interface AddComicLibraryPathBody {
  path: string
  title?: string
}

export type AddComicLibraryPathResultDTO = ComicLibraryPathDTO

export interface UpdateComicLibraryPathBody {
  title: string
}

export type ComicReadStatus = "unread" | "reading" | "read"

export interface ComicBookListItemDTO {
  id: string
  title: string
  tags: string[]
  rating?: number | null
  isFavorite: boolean
  readStatus: ComicReadStatus | string
  pageCount: number
  currentPageIndex: number
  coverUrl?: string
  sourceFileName: string
  location: string
  addedAt: string
  updatedAt: string
  lastReadAt?: string
  completedAt?: string
}

export interface ComicBooksPageDTO {
  items: ComicBookListItemDTO[]
  total: number
  limit: number
  offset: number
}

export interface ListComicBooksParams {
  q?: string
  tag?: string
  favorite?: boolean
  readStatus?: string
  limit?: number
  offset?: number
}

export interface ComicPageDTO {
  comicId: string
  index: number
  entryPath: string
  fileName: string
  imageExt?: string
  width?: number
  height?: number
  imageUrl?: string
  thumbUrl?: string
}

export interface ComicBookDetailDTO extends ComicBookListItemDTO {
  pages: ComicPageDTO[]
}

export type ComicReaderMode = "page" | "scroll"
export type ComicFitMode = "contain" | "width"
export type ComicReadingDirection = "ltr" | "rtl"

export interface ComicReaderSettingsDTO {
  mode: ComicReaderMode
  fit: ComicFitMode
  direction: ComicReadingDirection
}

export interface ComicCacheSettingsDTO {
  maxBytes: number
}

export interface PhotoLibraryPathDTO {
  id: string
  path: string
  title: string
  firstLibraryScanPending?: boolean
}

export interface AddPhotoLibraryPathBody {
  path: string
  title?: string
}

export interface AddPhotoLibraryPathResultDTO extends PhotoLibraryPathDTO {
  scanTask?: TaskDTO
}

export interface UpdatePhotoLibraryPathBody {
  title: string
}

export interface PhotoBookListItemDTO {
  id: string
  title: string
  tags: string[]
  rating?: number | null
  isFavorite: boolean
  pageCount: number
  currentPageIndex: number
  coverUrl?: string
  sourceFileName: string
  location: string
  addedAt: string
  updatedAt: string
  lastViewedAt?: string
  completedAt?: string
}

export interface PhotoBooksPageDTO {
  items: PhotoBookListItemDTO[]
  total: number
  limit: number
  offset: number
}

export interface ListPhotoBooksParams {
  q?: string
  tag?: string
  favorite?: boolean
  limit?: number
  offset?: number
}

export interface PhotoPageDTO {
  photoId: string
  index: number
  entryPath: string
  fileName: string
  imageExt?: string
  width?: number
  height?: number
  imageUrl?: string
  thumbUrl?: string
}

export interface PhotoBookDetailDTO extends PhotoBookListItemDTO {
  pages: PhotoPageDTO[]
}

export type PhotoViewerMode = "page" | "scroll"
export type PhotoFitMode = "contain" | "width"
export type PhotoViewingDirection = "ltr" | "rtl"

export interface PhotoViewerSettingsDTO {
  mode: PhotoViewerMode
  fit: PhotoFitMode
  direction: PhotoViewingDirection
}

export interface PhotoCacheSettingsDTO {
  maxBytes: number
}

export interface ComicCacheStatusDTO {
  maxBytes: number
  usedBytes: number
  entryCount: number
}

export interface ComicReadingProgressDTO {
  comicId: string
  pageIndex: number
  completed: boolean
  updatedAt: string
}

export interface ComicReadingPreferencesDTO {
  comicId?: string
  mode: ComicReaderMode
  fit: ComicFitMode
  direction: ComicReadingDirection
  updatedAt?: string
}

export interface PatchComicBookBody {
  title?: string
  tags?: string[]
  favorite?: boolean
  ratingSet?: boolean
  ratingClear?: boolean
  rating?: number
}

/** 写真展示标题与本地评分 PATCH body；清除评分时同时带 ratingSet 与 ratingClear。 */
export interface PatchPhotoBookBody {
  title?: string
  favorite?: boolean
  ratingSet?: boolean
  ratingClear?: boolean
  rating?: number
}

export interface PutComicProgressBody {
  pageIndex: number
  completed: boolean
}

export interface PutComicReadingPreferencesBody {
  mode?: ComicReaderMode
  fit?: ComicFitMode
  direction?: ComicReadingDirection
}

/** 与后端 contracts.MaxBookTitleRunes 一致 */
export const MAX_BOOK_TITLE_RUNES = 500

/** 与后端 contracts.MaxBookCommentRunes 一致 */
export const MAX_BOOK_COMMENT_RUNES = 10000

/** GET/PUT 漫画或写真详情个人备注（每本一条可覆盖） */
export interface BookCommentDTO {
  body: string
  updatedAt: string
}

export interface PutBookCommentBody {
  body: string
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
export type CuratedFrameExportFormat = "jpg" | "webp" | "png"
export type CuratedFrameExportMode = "raw" | "watermarked"

export interface SettingsDTO {
  libraryPaths: LibraryPathDTO[]
  /** Library path id used as the target for top-bar movie imports. Empty or missing means not configured. */
  defaultImportLibraryPathId?: string
  /** Remembered directory for newly generated backup package filenames. Empty means not configured. */
  backupDirectory: string
  comicLibraryEnabled: boolean
  autoComicLibraryWatch: boolean
  comicLibraryPaths: ComicLibraryPathDTO[]
  defaultComicImportLibraryPathId?: string
  comicReader: ComicReaderSettingsDTO
  comicCache: ComicCacheSettingsDTO
  photoLibraryEnabled: boolean
  autoPhotoLibraryWatch: boolean
  photoLibraryPaths: PhotoLibraryPathDTO[]
  defaultPhotoImportLibraryPathId?: string
  photoViewer: PhotoViewerSettingsDTO
  photoCache: PhotoCacheSettingsDTO
  player: PlayerSettingsDTO
  /** 扫描后整理为 番号/番号.ext 并写入 NFO/资产到番号目录 */
  organizeLibrary: boolean
  /**
   * 开启后，在新加入的库根「第一次成功扫描」时，会尝试识别 Curated 清单或外部整理目录布局（仅标注，不改动已有库路径的默认行为）。
   */
  /** 为 true 时库根目录监听新文件并防抖触发扫描（及后续刮削）；与主配置 libraryWatchEnabled 共同生效 */
  autoLibraryWatch: boolean
  autoActorProfileScrape: boolean
  autoDownloadUpdates: boolean
  launchAtLogin: boolean
  launchAtLoginSupported: boolean
  /** 是否允许局域网设备访问本机 HTTP 服务；改监听需完全退出后重新打开 */
  browserPluginEnabled?: boolean
  discoveryEnabled?: boolean
  lanEnabled: boolean
  /** 当前进程是否已经绑定非 loopback 地址 */
  lanListening: boolean
  /** 可供局域网设备打开的 http://私网IPv4:端口 候选 */
  lanAccessUrls: string[]
  curatedFrameExportFormat: CuratedFrameExportFormat
  curatedFrameExportMode: CuratedFrameExportMode
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
  /** 实验性 Agent provider 配置（library-config.cfg） */
  aiProvider: AIProviderSettingsDTO
  aiGovernance?: import("@/services/contracts/ai-governance-service").AIGovernanceSettings
  /** 后端进程日志（文件 + 级别）；重启后端后作用于 Zap */
  backendLog: BackendLogSettingsDTO
}

export interface ProxySettingsDTO {
  enabled: boolean
  url?: string
  username?: string
  password?: string
}

/** 实验：Agent LLM provider 配置（OpenAI 兼容）；baseUrl/model 为空表示未配置 */
export interface AIProviderSettingsDTO {
  /** Total model window in tokens; absent on older servers. */
  contextWindow?: number
  kind: string
  baseUrl: string
  apiKey?: string
  model: string
}

/** 实验：Agent provider 局部更新；未发送字段保持不变，空字符串清除 */
export interface PatchAIProviderBody {
  contextWindow?: number
  baseUrl?: string
  apiKey?: string
  model?: string
}

/** POST /api/ai/provider/test — 可选传入草稿配置而未保存时先测试 */
export interface AIProviderTestRequestBody {
  provider?: AIProviderSettingsDTO
}

export interface AIProviderTestResponse {
  ok: boolean
  latencyMs: number
  message?: string
}

/** POST /api/ai/chat 的消息行 */
export interface AIChatMessageDTO {
  role: "system" | "user" | "assistant"
  content: string
}

export interface AIChatMentionDTO {
  kind: "movie" | "actor" | "tag" | "comic" | "photo"
  id: string
  label: string
}

/** Bounded, one-turn projection of the active library filters. */
export interface AIChatActiveFiltersDTO {
  query?: string
  tag?: string
  actor?: string
  playState?: "all" | "unwatched" | "in-progress" | "completed"
  runtime?: "short" | "standard" | "long"
  favorite?: boolean
  readStatus?: "unread" | "reading" | "read"
}

export interface AIChatContextDTO {
  /** Omit for the legacy page-context shape. Version 1 enables explicit selections and filters. */
  contextVersion?: 1
  route?: string
  movieId?: string
  comicId?: string
  photoId?: string
  actorName?: string
  query?: string
  mentions?: AIChatMentionDTO[]
  selectedMovieIds?: string[]
  /** Actor public list rows use canonical names, not a stable actor ID. */
  selectedActors?: string[]
  selectedComicIds?: string[]
  selectedPhotoIds?: string[]
  activeFilters?: AIChatActiveFiltersDTO
}

export interface AIChatSessionDTO {
  id: string
  title?: string
  createdAt: string
  updatedAt: string
}

export interface AIChatSessionListDTO {
  items: AIChatSessionDTO[]
}

export interface AIChatStoredMessageDTO {
  events?: AIChatStoredEventDTO[]
  id: string
  sessionId: string
  role: string
  content: string
  toolName?: string
  toolCallId?: string
  seq: number
  createdAt: string
}

export interface AIChatSessionDetailDTO extends AIChatSessionDTO {
  nextCursor?: string
  messages: AIChatStoredMessageDTO[]
}

export interface AIAgentMovieCardDTO {
  movieId: string
  title?: string
  code?: string
  actors?: string[]
  coverUrl?: string
  thumbUrl?: string
  reason?: string
}

export interface AIAgentBookCardDTO {
  kind: "comic" | "photo"
  comicId?: string
  photoId?: string
  title?: string
  coverUrl?: string
  tags?: string[]
}

/** Public lifecycle metadata; private checkpoint text stays on the server. */
export interface AIContextStatusDTO {
  phase: "compacting" | "ready" | "limited"
}

/** Persisted result events; confirmation tokens are deliberately not stored. */
export interface AIChatStoredEventDTO {
  context?: AIContextStatusDTO
  answerEvidence?: AIAnswerEvidenceDTO
  receiptId?: string
  applied?: boolean
  type: string
  toolCallId?: string
  name?: string
  ok?: boolean
  summary?: string
  movies?: AIAgentMovieCardDTO[]
  books?: AIAgentBookCardDTO[]
  providerRows?: AIAgentProviderTitleDTO[]
  evidence?: AIEvidenceDTO
  resolution?: AIEntityResolutionDTO
  outcome?: AIChatOutcomeDTO
  changes?: AIConfirmChangeDTO[]
}

/** Published record snapshots; these never grant authority to later requests. */
export interface AIAnswerEvidenceDTO {
  version: 1
  items: {
    refId: string
    movieId?: string
    source: "local" | "provider"
    tool: string
    retrievedAt: string
    truncated?: boolean
    fields: Record<string, unknown>
  }[]
}

export interface AIAgentProviderTitleDTO {
  code?: string
  title?: string
  provider?: string
  score?: number
  homepage?: string
  inLibrary: boolean
  movieId?: string
}

export interface AIEntityCandidateDTO {
  kind: "movie" | "actor"
  movieId?: string
  actorName?: string
  title?: string
  code?: string
  aliases?: string[]
  reason?: string
}

export interface AIEntityResolutionDTO {
  query: string
  kind: "auto" | "movie" | "actor"
  status: "matched" | "ambiguous" | "unmatched"
  candidates: AIEntityCandidateDTO[]
  reason?: string
}

export interface AIEvidenceDTO {
  source: "local" | "provider" | "source_page"
  retrievedAt: string
  filters?: Record<string, string>
  truncated?: boolean
  nextCursor?: string
  failed?: boolean
  errorCode?: string
}

export interface AIChatOutcomeDTO {
  status: "completed" | "partial" | "needs_input" | "needs_confirmation" | "cancelled" | "failed"
  reasonCode?: string
  reason?: string
  retryable?: boolean
}

export interface AIConfirmChangeDTO {
  path: string
  before?: unknown
  after?: unknown
}

export interface AIActionRequestBody {
  movieId?: string
  comicId?: string
  photoId?: string
  body?: string
  locale?: string
  targetLocale?: string
  range?: string
  timezone?: string
}

export interface AIActionPreviewDTO {
  action: string
  name: string
  sessionId: string
  originalText?: string
  proposedText?: string
  changes?: AIConfirmChangeDTO[]
  confirmToken?: string
  expiresAt?: string
  arguments?: Record<string, unknown>
  noop?: boolean
}

export interface AIToolApplyRequestBody {
  sessionId: string
  name: string
  arguments: Record<string, unknown>
  confirmToken: string
}

export interface AIToolApplyDTO {
  /** Existing committed result; data is a historical snapshot. */
  replayed?: boolean
  ok: boolean
  name: string
  data?: unknown
}

/** 后端日志目录与级别（library-config.cfg）；空 logDir 表示使用当前构建的默认日志目录 */
export interface BackendLogSettingsDTO {
  logDir: string
  logFilePrefix?: string
  logMaxAgeDays?: number
  logLevel?: string
}

export interface BackupScopeDTO {
  databaseIncluded: boolean
  libraryConfigIncluded: boolean
  userAssetsIncluded: boolean
  wishlistAssetsIncluded?: boolean
  mediaFilesIncluded: boolean
}

export interface BackupFileDTO {
  kind: string
  path: string
  sizeBytes: number
  sha256: string
}

export interface BackupManifestDTO {
  format: string
  formatVersion: number
  createdAt: string
  appVersion: string
  appChannel: string
  scope: BackupScopeDTO
  schemaMigrations: string[]
  files: BackupFileDTO[]
}

export interface BackupIntegrityDTO {
  quickCheck: string
  foreignKeyViolations: number
}

export interface BackupVerificationDTO {
  valid: boolean
  checkedAt: string
  manifest?: BackupManifestDTO
  databaseIntegrity: BackupIntegrityDTO
  errors: string[]
  warnings: string[]
}

export interface BackupRestorePreflightDTO {
  canRestore: boolean
  checkedAt: string
  verification: BackupVerificationDTO
  targetDatabase: string
  targetDatabaseExists: boolean
  targetConfig?: string
  targetConfigExists: boolean
  requiredBytes: number
  availableBytes: number
  availableBytesKnown: boolean
  unsupportedMigrations: string[]
  errors: string[]
  warnings: string[]
}

export interface BackupCreateBody {
  destinationPath: string
}

export interface BackupPathBody {
  backupPath: string
}

export type LibraryHealthStatus = "healthy" | "attention" | "critical"
export type LibraryHealthSeverity = "critical" | "warning" | "info"

export interface LibraryHealthDatabaseDTO {
  quickCheckOk: boolean
  quickCheckMessages: string[]
  foreignKeyOk: boolean
  foreignKeyCount: number
}

export interface LibraryHealthFindingDTO {
  id: string
  category: string
  severity: LibraryHealthSeverity
  entityType: string
  entityId?: string
  label: string
  path?: string
  message: string
  details?: Record<string, unknown>
  repairActions?: string[]
}

export interface LibraryHealthSummaryDTO {
  totalFindings: number
  criticalFindings: number
  warningFindings: number
  infoFindings: number
  skippedOfflineFiles: number
  categoryCounts: Record<string, number>
}

export interface LibraryHealthReportDTO {
  scannedAt: string
  status: LibraryHealthStatus
  database: LibraryHealthDatabaseDTO
  storageStatuses: LibraryPathStorageStatusDTO[]
  summary: LibraryHealthSummaryDTO
  findings: LibraryHealthFindingDTO[]
  truncated: boolean
}

export interface StartLibraryHealthRepairBody {
  action: "rescrape_metadata"
  categories?: Array<"metadata_missing" | "metadata_failed">
  findingIds?: string[]
  limit?: number
  confirm: boolean
}

export interface StartLibraryHealthActionBody {
  action: "cleanup_orphan_state" | "cleanup_import_staging"
  findingIds: string[]
  confirm: boolean
}

export interface LibraryHealthRepairItemDTO {
  ordinal: number
  findingId: string
  category: string
  movieId: string
  label: string
  status: "pending" | "queued" | "succeeded" | "failed" | "cancelled"
  childTaskId?: string
  errorCode?: string
  errorMessage?: string
  startedAt?: string
  finishedAt?: string
}

export interface LibraryHealthRepairDTO {
  repairId: string
  taskId: string
  action: "rescrape_metadata"
  categories: string[]
  status: "pending" | "running" | "completed" | "partial_failed" | "failed" | "cancelled"
  totalItems: number
  completedItems: number
  succeededItems: number
  failedItems: number
  createdAt: string
  startedAt?: string
  finishedAt?: string
  items: LibraryHealthRepairItemDTO[]
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

export interface PatchComicSettingsBody {
  comicLibraryEnabled?: boolean
  autoComicLibraryWatch?: boolean
  defaultComicImportLibraryPathId?: string
  comicReader?: ComicReaderSettingsDTO
  comicCache?: ComicCacheSettingsDTO
}

export interface PatchPhotoSettingsBody {
  photoLibraryEnabled?: boolean
  autoPhotoLibraryWatch?: boolean
  defaultPhotoImportLibraryPathId?: string
  photoViewer?: PhotoViewerSettingsDTO
  photoCache?: PhotoCacheSettingsDTO
}

export interface PatchSettingsBody extends PatchComicSettingsBody, PatchPhotoSettingsBody {
  organizeLibrary?: boolean
  autoLibraryWatch?: boolean
  autoActorProfileScrape?: boolean
  autoDownloadUpdates?: boolean
  launchAtLogin?: boolean
  browserPluginEnabled?: boolean
  discoveryEnabled?: boolean
  lanEnabled?: boolean
  curatedFrameExportFormat?: CuratedFrameExportFormat
  curatedFrameExportMode?: CuratedFrameExportMode
  defaultImportLibraryPathId?: string
  /** Absolute backup destination directory; an empty string clears the remembered value. */
  backupDirectory?: string
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
  /** 实验性 Agent provider 配置；发送则合并更新 */
  aiProvider?: PatchAIProviderBody
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

export interface MovieImportUploadProgress {
  loaded: number
  total: number
  percent: number
}

export interface ComicImportUploadProgress {
  loaded: number
  total: number
  percent: number
}

export interface PhotoImportUploadProgress {
  loaded: number
  total: number
  percent: number
}

export interface MovieImportUploadFileManifest {
  relativePath: string
  size: number
  lastModified?: number
}

export interface CreateMovieImportUploadBody {
  files: MovieImportUploadFileManifest[]
}

export type ImportMovieCodeMatchKind = "exact" | "similar"

export interface CheckImportMovieCodesBody {
  names: string[]
}

export interface ImportMovieCodeMatchDTO {
  movieId: string
  code: string
  title: string
  matchKind: ImportMovieCodeMatchKind
}

export interface ImportMovieCodeCheckItemDTO {
  name: string
  extractedCode?: string
  matches: ImportMovieCodeMatchDTO[]
}

export interface ImportMovieCodeCheckDTO {
  items: ImportMovieCodeCheckItemDTO[]
  matchedCount: number
}

export interface MovieImportUploadChunkDTO {
  index: number
  offset: number
  size: number
}

export interface MovieImportUploadFileDTO {
  fileId: string
  relativePath: string
  size: number
  bytesReceived: number
  complete: boolean
  state?: string
  /** Persisted chunk ranges; lets a resumed client skip already-uploaded chunks. */
  chunks?: MovieImportUploadChunkDTO[]
}

export interface MovieImportUploadDTO {
  uploadId: string
  targetPath: string
  chunkSize: number
  bytesReceived: number
  totalBytes: number
  state: string
  expiresAt?: string
  recoveryStatus?: string
  recoveryError?: string
  files: MovieImportUploadFileDTO[]
  task: TaskDTO
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
  externalLinks?: string[]
  aliases?: string[]
}

export interface CreateMovieClipBody {
  startSec: number
  endSec: number
  format: "gif" | "mp4" | "webm"
  fps?: number
  width?: number
  curatedFrameId?: string
}

export interface PatchActorExternalLinksBody {
  externalLinks: string[]
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
  /** Exact tag match; comma-separated or repeated values require every tag (metadata or user tags). */
  tag?: string
  /** Exact actor names; comma-separated or repeated values require every actor (AND). */
  actor?: string
  /** Exact studio names; comma-separated or repeated values match any studio (OR). */
  studio?: string
  playState?: "all" | "unwatched" | "in-progress" | "completed"
  /** Minimum local user rating 0–5. Ignored when `unrated` is true. */
  userRating?: number
  unrated?: boolean
  /** `4k` 同时匹配 4K / 2160p / UHD */
  resolution?: string
  /** RFC3339 或 YYYY-MM-DD；后端转换为 UTC 后做 added_at 下界筛选 */
  addedAfter?: string
  /** Exact year or `unknown` */
  year?: string
  runtime?: SavedViewRuntime
  catalog?: SavedViewCatalog
  limit?: number
  offset?: number
}

export interface ActorMergeActorRefDTO {
  id: number
  name: string
  aliases: string[]
}

export interface ActorMergeAssociationSummaryDTO {
  sourceCount: number
  targetCount: number
  duplicateCount: number
  resultCount: number
}

export interface ActorMergeValuesSummaryDTO {
  source: string[]
  target: string[]
  result: string[]
}

export interface ActorMergeProfileFieldDTO {
  field: string
  sourceValue: string
  targetValue: string
  defaultSelection: "source" | "target"
  conflict: boolean
}

export interface ActorMergeBlockingReasonDTO {
  code: string
  message: string
}

export interface ActorMergePreviewRequest {
  sourceName: string
  targetName: string
}

export interface ActorMergePreviewDTO {
  previewToken: string
  source: ActorMergeActorRefDTO
  target: ActorMergeActorRefDTO
  movies: ActorMergeAssociationSummaryDTO
  userTags: ActorMergeValuesSummaryDTO
  externalLinks: ActorMergeValuesSummaryDTO
  recommendationFeedback: ActorMergeAssociationSummaryDTO
  curatedFramesAffected: number
  aliasesToMove: string[]
  profileFields: ActorMergeProfileFieldDTO[]
  canApply: boolean
  blockingReasons: ActorMergeBlockingReasonDTO[]
  requiredDecisions: string[]
}

export type ActorMergeProfileSelection = "source" | "target"

export interface ApplyActorMergeRequest {
  sourceName: string
  targetName: string
  previewToken: string
  confirm: boolean
  profileDecisions?: Record<string, ActorMergeProfileSelection>
}

export interface ActorMergeAuditSummaryDTO {
  movies: ActorMergeAssociationSummaryDTO
  userTags: string[]
  externalLinks: string[]
  aliases: string[]
  recommendationFeedback: ActorMergeAssociationSummaryDTO
  curatedFramesAffected: number
  profileDecisions: Record<string, ActorMergeProfileSelection>
}

export interface ActorMergeAuditDTO {
  id: string
  sourceActorId: number
  targetActorId: number
  sourceName: string
  targetName: string
  previewToken: string
  summary: ActorMergeAuditSummaryDTO
  appliedAt: string
}

export interface ActorMergeAuditListDTO {
  items: ActorMergeAuditDTO[]
  total: number
  limit: number
  offset: number
}

export type SavedViewMode = "library" | "favorites" | "recent" | "tags" | "trash"
export type SavedViewTab = "all" | "new" | "top-rated"
export type SavedViewPlayState = "all" | "unwatched" | "in-progress" | "completed"
export type SavedViewRuntime = "short" | "standard" | "long"
export type SavedViewCatalog = "unscraped" | "no-cover"
export type SavedViewSort =
  | "added"
  | "release"
  | "rating"
  | "code"
  | "actor"
  | "studio"
  | "year"

export interface SavedViewFiltersV1 {
  schemaVersion: 1
  mode?: SavedViewMode
  q?: string
  /** Comma-separated exact tags; movies must include every tag in metadata or user tags. */
  tag?: string
  /** Comma-separated exact actors; movies must include every selected actor (AND). */
  actor?: string
  /** Comma-separated exact studios; movies may match any selected studio (OR). */
  studio?: string
  tab?: SavedViewTab
  /** Grid sort; omitted means added-at (or the legacy tab mapping). */
  sort?: SavedViewSort
  playState?: SavedViewPlayState
  /** Minimum local user rating 0–5. Ignored when `unrated` is true. */
  userRating?: number
  /** Only movies with no local user rating. */
  unrated?: boolean
  resolution?: string
  addedWithinDays?: number
  /** Exact release year, or `unknown` when year is missing. */
  year?: string
  runtime?: SavedViewRuntime
  catalog?: SavedViewCatalog
}

export interface SavedViewDTO {
  id: string
  name: string
  filters: SavedViewFiltersV1
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface SavedViewsDTO {
  items: SavedViewDTO[]
}

export interface CreateSavedViewBody {
  name: string
  filters: SavedViewFiltersV1
}

export interface PatchSavedViewBody {
  name?: string
  filters?: SavedViewFiltersV1
}

export interface ReorderSavedViewsBody {
  ids: string[]
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

/** GET /playback/watch-time/daily */
export interface PlaybackWatchTimeDailyItemDTO {
  dayKey: string
  watchedSec: number
}

export interface PlaybackWatchTimeDailyListDTO {
  items: PlaybackWatchTimeDailyItemDTO[]
  totalWatchedSec: number
  activeDays: number
  maxDayWatchedSec: number
  longestStreakDays: number
}

/** POST /playback/watch-time/daily */
export interface AddPlaybackWatchTimeBody {
  movieId: string
  dayKey: string
  watchedSec: number
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

export interface PlaybackSessionStatusDTO {
  sessionId: string
  movieId: string
  sessionKind?: string
  transcodeProfile?: string
  startPositionSec?: number
  startedAt?: string
  lastAccessedAt?: string
  expiresAt?: string
  finishedAt?: string
  state?: string
  lastError?: string
  encoderSpeed?: string
  writtenDurationSec?: number
  lastSeekKind?: string
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
  motion?: CuratedFrameMotionDTO
}

export interface CuratedFrameMotionDTO {
  status: "processing" | "ready" | "error"
  contentType: string
  durationSec: number
  width: number
  height: number
  fps: number
  fileSize: number
  artifactUrl?: string
  errorMessage?: string
  createdAt?: string
  updatedAt?: string
}

export interface ListCuratedFramesParams {
  cursor?: string
  skipTotal?: boolean
  q?: string
  actor?: string
  movieId?: string
  /** Exact tag match; when multiple values are provided, frames must include every tag. */
  tag?: string
  tags?: string[]
  limit?: number
  offset?: number
}

export interface CuratedFramesListDTO {
  nextCursor?: string
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
  format?: CuratedFrameExportFormat
}

/** GET /library/played-movies */
export interface PlayedMoviesListDTO {
  movieIds: string[]
}

export interface HomepageDailyRecommendationsDTO {
  dateUtc: string
  generatedAt: string
  generationVersion?: string
  heroMovieIds: string[]
  recommendationMovieIds: string[]
  recommendations: HomepageRecommendationItemDTO[]
}

export type HomepageRecommendationReasonCode =
  | "high_user_rating"
  | "favorite"
  | "recently_added"
  | "well_rated"
  | "rediscovery"
  | "catalog_discovery"
  | "shared_actor"
  | "shared_studio"
  | "shared_tag"

export interface HomepageRecommendationReasonDTO {
  code: HomepageRecommendationReasonCode
  entityType?: "actor" | "studio" | "tag"
  entityValue?: string
}

export interface HomepageRecommendationFeedbackEffectDTO {
  feedbackId: string
  targetType: "actor" | "studio" | "tag"
  targetValue: string
  effect: "weight_reduced"
}

export interface HomepageRecommendationItemDTO {
  movieId: string
  reasons: HomepageRecommendationReasonDTO[]
  feedbackEffects: HomepageRecommendationFeedbackEffectDTO[]
}

export type RecommendationFeedbackAction = "not_interested" | "snooze" | "less"
export type RecommendationFeedbackTargetType = "movie" | "actor" | "studio" | "tag"

export interface RecommendationFeedbackDTO {
  id: string
  action: RecommendationFeedbackAction
  targetType: RecommendationFeedbackTargetType
  targetValue: string
  sourceMovieId: string
  expiresAt?: string
  createdAt: string
  updatedAt: string
}

export interface RecommendationFeedbackListDTO {
  items: RecommendationFeedbackDTO[]
}

export interface CreateRecommendationFeedbackBody {
  action: RecommendationFeedbackAction
  targetType: RecommendationFeedbackTargetType
  targetValue: string
  sourceMovieId: string
  durationDays?: number
}

export type PersonalInsightsRange = "30d" | "90d" | "365d" | "all"
export type PersonalInsightsDimension = "actor" | "studio" | "tag"

export interface PersonalInsightsOverviewDTO {
  range: PersonalInsightsRange
  from: string
  to: string
  timezone: string
  generatedAt: string
  dataSince: string | null
  watchedSeconds: number
  startedMovies: number
  completedMovies: number
  completionRate: number | null
  completionThreshold: number
  ratedMovies: number
  averageUserRating: number | null
}

export interface PersonalInsightsBreakdownItemDTO {
  name: string
  watchedSeconds: number
  movieCount: number
  shareOfTotal: number
}

export interface PersonalInsightsBreakdownDTO {
  range: PersonalInsightsRange
  dimension: PersonalInsightsDimension
  from: string
  to: string
  timezone: string
  generatedAt: string
  dataSince: string | null
  totalWatchedSeconds: number
  attribution: "full-per-entity"
  items: PersonalInsightsBreakdownItemDTO[]
  limit: number
}

export interface RefreshHomepageDailyRecommendationsBody {
  preserveHeroMovieIds?: string[]
  excludeRecommendationMovieIds?: string[]
}
