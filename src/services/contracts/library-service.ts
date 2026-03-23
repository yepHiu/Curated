import type { ComputedRef } from "vue"
import type {
  ActorListItemDTO,
  ActorsListDTO,
  ListActorsParams,
  MetadataRefreshQueuedDTO,
  PatchMovieBody,
  TaskDTO,
} from "@/api/types"
import type { LibrarySetting, LibraryStat } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"

export interface LibraryService {
  movies: ComputedRef<readonly Movie[]>
  libraryStats: ComputedRef<readonly LibraryStat[]>
  libraryPaths: ComputedRef<readonly LibrarySetting[]>
  refreshSettings(): Promise<void>
  /** Web：自 API 重新拉取全库列表（扫描/监听入库后更新计数与海报）；Mock：空操作 */
  reloadMoviesFromApi(): Promise<void>
  /** 与后端 GET/PATCH /api/settings 同步；mock 为本地状态 */
  organizeLibrary: ComputedRef<boolean>
  setOrganizeLibrary(value: boolean): Promise<void>
  /** 库目录监听触发的自动扫描/刮削；mock 为本地状态 */
  autoLibraryWatch: ComputedRef<boolean>
  setAutoLibraryWatch(value: boolean): Promise<void>
  /** 影片刮削源：空为自动；mock 下列表常为空，仅支持自动 */
  metadataMovieProvider: ComputedRef<string>
  metadataMovieProviders: ComputedRef<readonly string[]>
  setMetadataMovieProvider(name: string): Promise<void>
  /** Web：后端会尝试对该路径启动初次扫描，返回任务供上层轮询；Mock 恒为 null */
  addLibraryPath(path: string, title?: string): Promise<TaskDTO | null>
  updateLibraryPathTitle(id: string, title: string): Promise<void>
  removeLibraryPath(id: string): Promise<void>
  /** Returns task when web scan started; mock returns null. */
  scanLibraryPaths(paths?: string[]): Promise<TaskDTO | null>
  /** 单部影片重新刮削；Web 返回任务供轮询；mock 返回 null。 */
  refreshMovieMetadata(movieId: string): Promise<TaskDTO | null>
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
   * Web：返回可赋给 video.src 的流地址；无后端或未启用 API 时返回 null。
   * Mock：返回 null（或可选固定演示 URL）。
   */
  getMoviePlaybackUrl(movieId: string): string | null
  getRelatedMovies(movieId: string, limit?: number): Movie[]
  /**
   * 更新收藏与/或用户评分（Web：PATCH /api/library/movies/{id}；Mock：内存）。
   * 失败时 Web 适配器会恢复列表快照并抛出错误。
   */
  patchMovie(movieId: string, body: PatchMovieBody): Promise<Movie | undefined>
  /** 仅更新收藏；等价于 patchMovie(id, { isFavorite }) */
  toggleFavorite(movieId: string, nextValue?: boolean): Promise<Movie | undefined>
  /** 删除影片（Web：请求后端并从本地列表移除；Mock：仅从内存列表移除） */
  deleteMovie(movieId: string): Promise<void>
  /**
   * Web：把详情合并进列表缓存（已存在则覆盖同 id）。Mock：空操作。
   */
  mergeMovieIntoCache(movie: Movie): void
  /** Web：GET /library/actors；Mock：由内存影片聚合 */
  listActors(params?: ListActorsParams): Promise<ActorsListDTO>
  /** Web：PATCH /library/actors/tags；Mock：内存 Map */
  patchActorUserTags(name: string, userTags: string[]): Promise<ActorListItemDTO>
}
