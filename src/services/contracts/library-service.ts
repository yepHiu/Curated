import type { ComputedRef } from "vue"
import type {
  ActorListItemDTO,
  ActorMergeAuditDTO,
  ActorMergeAuditListDTO,
  ActorMergePreviewDTO,
  ActorMergePreviewRequest,
  ApplyActorMergeRequest,
  ActorProfileDTO,
  ActorsListDTO,
  AIProviderSettingsDTO,
  AIProviderTestResponse,
  PatchAIProviderBody,
  BackendLogSettingsDTO,
  BackupManifestDTO,
  BackupRestorePreflightDTO,
  BackupVerificationDTO,
  ConnectedClientsDTO,
  CuratedFrameExportFormat,
  CuratedFrameExportMode,
  HealthDTO,
  HomepageDailyRecommendationsDTO,
  CreateRecommendationFeedbackBody,
  RecommendationFeedbackDTO,
  RecommendationFeedbackListDTO,
  RefreshHomepageDailyRecommendationsBody,
  LibraryPathStorageStatusDTO,
  LibraryHealthRepairDTO,
  LibraryHealthReportDTO,
  ListActorsParams,
  MetadataMovieScrapeMode,
  MetadataRefreshQueuedDTO,
  MovieImportUploadProgress,
  MovieImportUploadFileManifest,
  ImportMovieCodeCheckDTO,
  NativePlaybackLaunchDTO,
  MovieCommentDTO,
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
  PlaybackDescriptorDTO,
  PlaybackSessionStatusDTO,
  PatchBackendLogBody,
  PostCuratedFramesExportBody,
  PatchMovieBody,
  PatchPlayerSettingsBody,
  PingAllProvidersResponse,
  PlayerSettingsDTO,
  ProviderHealthDTO,
  ProxySettingsDTO,
  ProxyJavBusPingRequestBody,
  ProxyJavBusPingResponse,
  PutMovieCommentBody,
  TaskDTO,
  CreateMovieClipBody,
  StartLibraryHealthRepairBody,
  StartLibraryHealthActionBody,
  SavedViewDTO,
  SavedViewFiltersV1,
} from "@/api/types"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

/** 一个可继续的断点续传上传会话（本地账本 + 后端状态核对后的视图模型）。 */
export interface ResumableMovieImportSession {
  uploadId: string
  /** 会话创建时的文件指纹（relativePath + size + lastModified），用于匹配重新选择的文件。 */
  files: MovieImportUploadFileManifest[]
  totalBytes: number
  bytesReceived: number
  expiresAt?: string
}

export interface LibraryService {
  movies: ComputedRef<readonly Movie[]>
  /** 电影主列表首轮加载是否已完成；true 代表拿到过一次明确结果，不代表一定有数据。 */
  moviesLoaded: ComputedRef<boolean>
  /** 最近一次资料库列表/详情加载错误；用于视图层展示用户可见反馈。 */
  loadError: ComputedRef<string | null>
  /** 回收站列表（Web：mode=trash；Mock：带 trashedAt 的条目） */
  trashedMovies: ComputedRef<readonly Movie[]>
  libraryStats: ComputedRef<readonly LibraryStat[]>
  libraryPaths: ComputedRef<readonly LibrarySetting[]>
  libraryPathStorageStatuses: ComputedRef<readonly LibraryPathStorageStatusDTO[]>
  /** Ordered user-defined library filters (SQLite in Web API; localStorage in Mock). */
  savedViews: ComputedRef<readonly SavedViewDTO[]>
  defaultImportLibraryPathId: ComputedRef<string>
  backupDirectory: ComputedRef<string>
  refreshSettings(): Promise<void>
  checkLibraryPathStorageStatus(libraryPathIds?: string[]): Promise<void>
  rebindLibraryPathStorage(id: string): Promise<void>
  /** Web：自 API 重新拉取全库列表（扫描/监听入库后更新计数与海报）；Mock：空操作 */
  reloadMoviesFromApi(): Promise<void>
  /** Web/Mock: returns the full active movie list used by CSV export. */
  listMoviesForExport(): Promise<readonly Movie[]>
  /** Web：仅在需要展示回收站时再拉取 trashed 列表；Mock：空操作 */
  ensureTrashLoaded(): Promise<void>
  refreshSavedViews(): Promise<void>
  createSavedView(name: string, filters: SavedViewFiltersV1): Promise<SavedViewDTO>
  updateSavedView(id: string, patch: { name?: string; filters?: SavedViewFiltersV1 }): Promise<SavedViewDTO>
  deleteSavedView(id: string): Promise<void>
  reorderSavedViews(ids: string[]): Promise<void>
  /** 与后端 GET/PATCH /api/settings 同步；mock 为本地状态 */
  organizeLibrary: ComputedRef<boolean>
  setOrganizeLibrary(value: boolean): Promise<void>
  /** 库目录监听触发的自动扫描/刮削；mock 为本地状态 */
  autoLibraryWatch: ComputedRef<boolean>
  setAutoLibraryWatch(value: boolean): Promise<void>
  autoActorProfileScrape: ComputedRef<boolean>
  setAutoActorProfileScrape(value: boolean): Promise<void>
  autoDownloadUpdates: ComputedRef<boolean>
  setAutoDownloadUpdates(value: boolean): Promise<void>
  launchAtLogin: ComputedRef<boolean>
  launchAtLoginSupported: ComputedRef<boolean>
  setLaunchAtLogin(value: boolean): Promise<void>
  curatedFrameExportFormat: ComputedRef<CuratedFrameExportFormat>
  setCuratedFrameExportFormat(format: CuratedFrameExportFormat): Promise<void>
  curatedFrameExportMode: ComputedRef<CuratedFrameExportMode>
  setCuratedFrameExportMode(mode: CuratedFrameExportMode): Promise<void>
  /** 影片刮削源：空为自动；mock 下列表常为空，仅支持自动 */
  metadataMovieProvider: ComputedRef<string>
  metadataMovieProviders: ComputedRef<readonly string[]>
  /** 有序的 Provider 列表；空数组表示自动（全源） */
  metadataMovieProviderChain: ComputedRef<readonly string[]>
  /** 后端当前生效的刮削策略（切换模式时可保留链列表供再次启用） */
  metadataMovieScrapeMode: ComputedRef<MetadataMovieScrapeMode>
  setMetadataMovieProvider(name: string): Promise<void>
  /** 设置有序的 Provider 列表；空数组表示自动（全源） */
  setMetadataMovieProviderChain(chain: string[]): Promise<void>
  /** 仅切换 auto | specified | chain，不删除已保存的 provider / chain 配置 */
  setMetadataMovieScrapeMode(mode: MetadataMovieScrapeMode): Promise<void>
  /** HTTP 代理配置 */
  proxy: ComputedRef<ProxySettingsDTO>
  setProxy(config: ProxySettingsDTO): Promise<void>
  /** 实验性 Agent provider 配置（Web：library-config.cfg；Mock：localStorage） */
  aiProvider: ComputedRef<AIProviderSettingsDTO>
  setAIProvider(patch: PatchAIProviderBody): Promise<void>
  /** 实验：测试 provider 连通；可选传草稿配置（不先保存） */
  testAIProvider(provider?: AIProviderSettingsDTO): Promise<AIProviderTestResponse>
  /** 播放器 / HLS / 原生播放器偏好 */
  playerSettings: ComputedRef<PlayerSettingsDTO>
  patchPlayerSettings(patch: PatchPlayerSettingsBody): Promise<void>
  /** 后端日志目录与级别（Web：library-config.cfg；Mock：内存） */
  backendLog: ComputedRef<BackendLogSettingsDTO>
  patchBackendLog(patch: PatchBackendLogBody): Promise<void>
  listConnectedClients(): Promise<ConnectedClientsDTO>
  health(): Promise<HealthDTO>
  createBackup(destinationPath: string): Promise<BackupManifestDTO>
  setBackupDirectory(directory: string): Promise<void>
  verifyBackup(backupPath: string): Promise<BackupVerificationDTO>
  preflightBackupRestore(backupPath: string): Promise<BackupRestorePreflightDTO>
  scanLibraryHealth(): Promise<LibraryHealthReportDTO>
  startLibraryHealthRepair(body: StartLibraryHealthRepairBody): Promise<LibraryHealthRepairDTO>
  getLibraryHealthRepair(repairId: string): Promise<LibraryHealthRepairDTO>
  startLibraryHealthAction(body: StartLibraryHealthActionBody): Promise<TaskDTO>
  pingProxyJavbus(body?: ProxyJavBusPingRequestBody): Promise<ProxyJavBusPingResponse>
  pingProxyGoogle(body?: ProxyJavBusPingRequestBody): Promise<ProxyJavBusPingResponse>
  pingProvider(name: string): Promise<ProviderHealthDTO>
  pingAllProviders(): Promise<PingAllProvidersResponse>
  getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO>
  refreshHomepageDailyRecommendations(body?: RefreshHomepageDailyRecommendationsBody): Promise<HomepageDailyRecommendationsDTO>
  listHomepageRecommendationFeedback(): Promise<RecommendationFeedbackListDTO>
  createHomepageRecommendationFeedback(body: CreateRecommendationFeedbackBody): Promise<RecommendationFeedbackDTO>
  deleteHomepageRecommendationFeedback(id: string): Promise<void>
  /** Web：后端会尝试对该路径启动初次扫描，返回任务供上层轮询；Mock 恒为 null */
  addLibraryPath(path: string, title?: string): Promise<TaskDTO | null>
  updateLibraryPathTitle(id: string, title: string): Promise<void>
  removeLibraryPath(id: string): Promise<void>
  revealLibraryPathInFileManager(id: string): Promise<void>
  setDefaultImportLibraryPathId(id: string): Promise<void>
  /**
   * 导入影片。resumeUploadId 指向既有续传会话时只补传缺失分片再提交；
   * Web 下大文件自动走断点续传并把会话写入本地账本。
   */
  importMovies(
    files: File[],
    options?: {
      onUploadProgress?: (progress: MovieImportUploadProgress) => void
      resumeUploadId?: string
    },
  ): Promise<TaskDTO | null>
  /** 导入前按文件名解析番号，检查库中是否已有相同或类似条目。 */
  checkImportMovieCodes(names: string[]): Promise<ImportMovieCodeCheckDTO>
  /** 当前可继续的续传会话列表（Web：本地账本 + 后端状态核对；Mock：恒为空）。 */
  listResumableMovieImports(): Promise<ResumableMovieImportSession[]>
  /** 放弃一个未完成的续传会话：删除服务端暂存与本地账本条目。 */
  abandonMovieImportUpload(uploadId: string): Promise<void>
  /** Returns task when web scan started; mock returns null. */
  scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null>
  getTaskStatus(taskId: string): Promise<TaskDTO>
  createMovieClip(movieId: string, body: Omit<CreateMovieClipBody, "format"> & { format?: "gif" }): Promise<TaskDTO>
  /** 单部影片重新刮削；Web 返回任务供轮询；mock 返回 null。 */
  refreshMovieMetadata(movieId: string): Promise<TaskDTO | null>
  /** Web：请求后端在系统文件管理器中显示该片主视频；Mock 会拒绝。 */
  revealMovieInFileManager(movieId: string): Promise<void>
  /**
   * 按已配置的库根路径批量排队元数据刮削（不重新扫盘）。
   * Web：POST /library/metadata-scrape；Mock：返回零计数演示结果。
   */
  refreshMetadataForLibraryPaths(paths: string[]): Promise<MetadataRefreshQueuedDTO>
  getMovieById(movieId?: string): Movie | undefined
  /** Load one movie through the active adapter and merge it into that adapter's cache when possible. */
  loadMovieDetail(movieId: string): Promise<Movie | undefined>
  /**
   * Web：列表未包含该 id 时拉取单条并写入缓存（避免仅加载首页导致播放/详情找不到）。
   * Mock：空操作。
   */
  ensureMovieCached(movieId: string): Promise<void>
  /**
   * Web：返回后端给出的播放描述（当前为 direct-play，后续可扩展 remux / transcode）。
   * Mock：返回 null。
   */
  getMoviePlayback(movieId: string, options?: { startPositionSec?: number; signal?: AbortSignal }): Promise<PlaybackDescriptorDTO | null>
  /**
   * 尽力预取播放描述符（点击进入播放器时提前发起）。Web：短 TTL 一次性缓存，
   * 下一次 `getMoviePlayback` 直接消费；Mock：无操作。
   */
  prefetchMoviePlayback(movieId: string, startPositionSec?: number): (() => void) | void
  createPlaybackSession(
    movieId: string,
    mode: PlaybackDescriptorDTO["mode"],
    startPositionSec?: number,
    signal?: AbortSignal,
  ): Promise<PlaybackDescriptorDTO | null>
  getPlaybackSession(sessionId: string): Promise<PlaybackSessionStatusDTO | null>
  launchNativePlayback(movieId: string, startPositionSec?: number): Promise<NativePlaybackLaunchDTO | null>
  /**
   * 从当前库缓存中随机推荐若干部（排除自身），最多 `limit` 条（默认 6）。
   * 顺序与选集由 `movieId` 派生种子决定，同一影片在候选集合不变时可复现，避免界面无意义跳动。
   */
  getRelatedMovies(movieId: string, limit?: number): Movie[]
  /**
   * 更新收藏与/或用户评分（Web：PATCH /api/library/movies/{id}；Mock：内存）。
   * 失败时 Web 适配器会恢复列表快照并抛出错误。
   */
  patchMovie(movieId: string, body: PatchMovieBody): Promise<Movie | undefined>
  /** 仅更新收藏；等价于 patchMovie(id, { isFavorite }) */
  toggleFavorite(movieId: string, nextValue?: boolean): Promise<Movie | undefined>
  /** 移入回收站（Web：DELETE 无 permanent；Mock：标记 trashedAt） */
  deleteMovie(movieId: string): Promise<void>
  /** 从回收站恢复（Web：POST …/restore；Mock：清除 trashedAt） */
  restoreMovie(movieId: string): Promise<void>
  /** 永久删除（须已在回收站；Web：DELETE ?permanent=true） */
  deleteMoviePermanently(movieId: string): Promise<void>
  deletePlaybackSession(sessionId: string): Promise<void>
  /**
   * Web：把详情合并进列表缓存（已存在则覆盖同 id）。Mock：空操作。
   */
  mergeMovieIntoCache(movie: Movie): void
  /** Web：GET /library/actors；Mock：由内存影片聚合 */
  listActors(params?: ListActorsParams): Promise<ActorsListDTO>
  /** Web：GET /library/actors/profile；Mock：由内存演员聚合基础资料 */
  getActorProfile(name: string): Promise<ActorProfileDTO>
  /** Web：POST /library/actors/scrape；Mock：返回已完成任务 */
  scrapeActorProfile(name: string): Promise<TaskDTO>
  /** Web：PATCH /library/actors/tags；Mock：内存 Map */
  patchActorUserTags(name: string, userTags: string[]): Promise<ActorListItemDTO>
  /** Web：PATCH /library/actors/external-links；Mock：内存 Map */
  patchActorExternalLinks(name: string, externalLinks: string[]): Promise<ActorProfileDTO>
  /** Read-only merge preview with stale-preview token and every affected association. */
  previewActorMerge(body: ActorMergePreviewRequest): Promise<ActorMergePreviewDTO>
  /** Confirm and transactionally apply a previously previewed canonical merge. */
  applyActorMerge(body: ApplyActorMergeRequest): Promise<ActorMergeAuditDTO>
  /** Query persisted canonical merge audit history. */
  listActorMergeAudits(params?: { limit?: number; offset?: number }): Promise<ActorMergeAuditListDTO>
  /** Return bounded, server-side personal viewing metrics for one local calendar range. */
  getPersonalInsightsOverview(params: {
    range: PersonalInsightsRange
    timezone: string
  }): Promise<PersonalInsightsOverviewDTO>
  /** Return one bounded actor, studio, or tag attribution ranking for the same range. */
  getPersonalInsightsBreakdown(params: {
    range: PersonalInsightsRange
    timezone: string
    dimension: PersonalInsightsDimension
    limit?: number
  }): Promise<PersonalInsightsBreakdownDTO>
  /** Web：GET /library/movies/{id}/comment；Mock：localStorage */
  getMovieComment(movieId: string): Promise<MovieCommentDTO>
  /** Web：PUT /library/movies/{id}/comment；Mock：localStorage */
  putMovieComment(movieId: string, body: PutMovieCommentBody): Promise<MovieCommentDTO>
  /** Web：POST /curated-frames/export；Mock：不支持后端打包导出 */
  exportCuratedFrames(body: PostCuratedFramesExportBody): Promise<{ blob: Blob; filename: string }>
}
