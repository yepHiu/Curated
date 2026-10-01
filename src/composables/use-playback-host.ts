import { computed, inject, onBeforeUnmount, provide, ref, shallowReactive, shallowRef, watch, type InjectionKey } from "vue"
import type { RouteLocationNormalizedLoaded } from "vue-router"
import type { Movie } from "@/domain/movie/types"
import { clearActivePlaybackSession } from "@/composables/use-active-playback-session"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"

export interface HostedPlaybackTarget {
  movie: Movie
  autoplay: boolean
}

export type PlaybackHost = ReturnType<typeof createPlaybackHost>
export const playbackHostKey: InjectionKey<PlaybackHost> = Symbol("curated-playback-host")

/** 由应用壳层拥有一个活动播放器；只在原生 PiP 存在时跨业务路由保留。 */
export function createPlaybackHost(route: RouteLocationNormalizedLoaded) {
  let stopMedia: (() => void) | null = null
  let toggleMedia: (() => Promise<void>) | null = null
  const target = shallowRef<HostedPlaybackTarget | null>(null)
  const pipActive = ref(false)
  const playing = ref(false)
  const playerRoute = shallowReactive({ ...route, query: { ...route.query } })
  const visible = computed(() => Boolean(target.value && route.name === "player" && route.params.id === target.value.movie.id))

  /** 当前影片返回主页面时复用实例，不重新请求 descriptor 或应用过时的续播时间。 */
  function hasMovie(movieId: string) {
    return target.value?.movie.id === movieId
  }

  /** 保存播放器来源路由，防止后台播放时被其它页面的 query 和 params 污染。 */
  function start(movie: Movie, autoplay: boolean, origin: RouteLocationNormalizedLoaded) {
    if (hasMovie(movie.id)) return
    Object.assign(playerRoute, origin, { query: { ...origin.query }, params: { ...origin.params } })
    pipActive.value = false
    playing.value = false
    target.value = { movie, autoplay }
  }

  /** 显式停止移除实例并清除续播入口；普通离页保留已有进度快照。 */
  function stop(clearSnapshot = true) {
    const movieId = target.value?.movie.id
    // 先保存并暂停，清除快照之后卸载不能再发布旧影片状态。
    stopMedia?.()
    target.value = null
    pipActive.value = false
    playing.value = false
    if (clearSnapshot && movieId) clearActivePlaybackSession(movieId)
  }

  /** 宿主绑定媒体停止及播放切换操作，作用域销毁时撤销注册。 */
  function registerMediaControls(controls: { stop: () => void; toggle: () => Promise<void> }) {
    stopMedia = controls.stop
    toggleMedia = controls.toggle
    return () => {
      // 只移除自己注册的操作，避免旧宿主影响后继宿主。
      if (stopMedia === controls.stop) stopMedia = null
      if (toggleMedia === controls.toggle) toggleMedia = null
    }
  }

  /** 侧栏与播放器通过同一宿主操作媒体，不需要跨组件持有 video 引用。 */
  async function togglePlayback() {
    await toggleMedia?.()
  }

  /** 只接收当前实例事件，迟到的旧播放器退出事件不能关闭新影片。 */
  function setPipActive(movieId: string, active: boolean) {
    if (!hasMovie(movieId)) return
    pipActive.value = active
    if (!active && !visible.value) stop(false)
  }

  /** 后台操作按钮与视频原生播放状态同步。 */
  function setPlaying(movieId: string, active: boolean) {
    if (hasMovie(movieId)) playing.value = active
  }

  watch(() => [route.name, route.params.id], () => {
    // 换片立即释放旧实例；普通导航仅保留已开启的小窗。
    if (!target.value || visible.value) return
    if (route.name === "player" || !pipActive.value) stop(false)
  }, { flush: "sync" })

  watch(() => route.fullPath, () => {
    // 仅前台同片路由更新快照，消费 t/autoplay 后不在下一次换流重新应用。
    if (visible.value) Object.assign(playerRoute, route, { query: { ...route.query }, params: { ...route.params } })
  }, { flush: "sync" })

  watch(authLockService.status, (status) => {
    // 认证失效时停止活动视频，即使业务路由尚未跳转到锁定页。
    if (isAuthLockEnabled() && status.pinEnabled && !status.unlocked) stop()
  }, { flush: "sync" })

  onBeforeUnmount(() => {
    // 锁定页替换壳层、服务器页面销毁和应用退出均结束实例。
    stop()
  })

  return { target, pipActive, playing, visible, playerRoute, hasMovie, start, stop, registerMediaControls, togglePlayback, setPipActive, setPlaying }
}

/** 在壳层提供作用域内的播放所有权，不跨服务器或重新挂载的应用共享实例。 */
export function providePlaybackHost(route: RouteLocationNormalizedLoaded) {
  const host = createPlaybackHost(route)
  provide(playbackHostKey, host)
  return host
}

/** 播放路由注册影片；独立组件测试或其它宿主可选择保留原有页内播放方式。 */
export function usePlaybackHost() {
  return inject(playbackHostKey, null)
}
