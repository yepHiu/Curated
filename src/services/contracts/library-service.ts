import type { ComputedRef } from "vue"
import type {
  ActorListItemDTO,
  ActorsListDTO,
  BackendLogSettingsDTO,
  ListActorsParams,
  MetadataMovieScrapeMode,
  MetadataRefreshQueuedDTO,
  NativePlaybackLaunchDTO,
  PlaybackDescriptorDTO,
  PatchBackendLogBody,
  PatchMovieBody,
  PatchPlayerSettingsBody,
  PlayerSettingsDTO,
  ProxySettingsDTO,
  TaskDTO,
} from "@/api/types"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

export interface LibraryService {
  movies: ComputedRef<readonly Movie[]>
  /** 回收站列表（Web：mode=trash；Mock：带 trashedAt 的条目） */
  trashedMovies: ComputedRef<readonly Movie[]>
  libraryStats: ComputedRef<readonly LibraryStat[]>
  libraryPaths: ComputedRef<readonly LibrarySetting[]>
  refreshSettings(): Promise<void>
  /** Web：自 API 重新拉取全库列表（扫描/监听入库后更新计数与海报）；Mock：空操作 */
  reloadMoviesFromApi(): Promise<void>
  /** Web：仅在需要展示回收站时再拉取 trashed 列表；Mock：空操作 */
  ensureTrashLoaded(): Promise<void>
  /** 与后端 GET/PATCH /api/settings 同步；mock 为本地状态 */
  organizeLibrary: ComputedRef<boolean>
  setOrganizeLibrary(value: boolean): Promise<void>
  /** 新库根首次扫描时的扩展导入识别（Curated / 外部整理）；默认关，与 organizeLibrary 独立 */
  extendedLibraryImport: ComputedRef<boolean>
  setExtendedLibraryImport(value: boolean): Promise<void>
  /** 库目录监听触发的自动扫描/刮削；mock 为本地状态 */
  autoLibraryWatch: ComputedRef<boolean>
  setAutoLibraryWatch(value: boolean): Promise<void>
  autoActorProfileScrape: ComputedRef<boolean>
  setAutoActorProfileScrape(value: boolean): Promise<void>
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
  /** 播放器 / HLS / 原生播放器偏好 */
  playerSettings: ComputedRef<PlayerSettingsDTO>
  patchPlayerSettings(patch: PatchPlayerSettingsBody): Promise<void>
  /** 后端日志目录与级别（Web：library-config.cfg；Mock：内存） */
  backendLog: ComputedRef<BackendLogSettingsDTO>
  patchBackendLog(patch: PatchBackendLogBody): Promise<void>
  /** Web：后端会尝试对该路径启动初次扫描，返回任务供上层轮询；Mock 恒为 null */
  addLibraryPath(path: string, title?: string): Promise<TaskDTO | null>
  updateLibraryPathTitle(id: string, title: string): Promise<void>
  removeLibraryPath(id: string): Promise<void>
  /** Returns task when web scan started; mock returns null. */
  scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null>
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
  /**
   * Web：列表未包含该 id 时拉取单条并写入缓存（避免仅加载首页导致播放/详情找不到）。
   * Mock：空操作。
   */
  ensureMovieCached(movieId: string): Promise<void>
  /**
   * Web：返回后端给出的播放描述（当前为 direct-play，后续可扩展 remux / transcode）。
   * Mock：返回 null。
   */
  getMoviePlayback(movieId: string): Promise<PlaybackDescriptorDTO | null>
  createPlaybackSession(
    movieId: string,
    mode: PlaybackDescriptorDTO["mode"],
    startPositionSec?: number,
  ): Promise<PlaybackDescriptorDTO | null>
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
  /**
   * Web：把详情合并进列表缓存（已存在则覆盖同 id）。Mock：空操作。
   */
  mergeMovieIntoCache(movie: Movie): void
  /** Web：GET /library/actors；Mock：由内存影片聚合 */
  listActors(params?: ListActorsParams): Promise<ActorsListDTO>
  /** Web：PATCH /library/actors/tags；Mock：内存 Map */
  patchActorUserTags(name: string, userTags: string[]): Promise<ActorListItemDTO>
}
