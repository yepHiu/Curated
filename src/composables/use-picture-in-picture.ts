import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from "vue"

export type PictureInPictureFailure = "notReady" | "gesture" | "blocked" | "unsupported" | "failed"

/** 将浏览器异常归为可本地化的操作反馈，不直接向用户暴露底层异常。 */
export function classifyPictureInPictureFailure(error: unknown): PictureInPictureFailure {
  const name = error instanceof Error || error instanceof DOMException ? error.name : ""
  if (name === "InvalidStateError") return "notReady"
  if (name === "NotAllowedError") return "gesture"
  if (name === "SecurityError") return "blocked"
  if (name === "NotSupportedError") return "unsupported"
  return "failed"
}

/** 管理原生视频 PiP：媒体准入、请求互斥、系统事件及迟到请求的清理。 */
export function usePictureInPicture(
  video: Ref<HTMLVideoElement | null>,
  source: Ref<string | null>,
  reportFailure: (failure: PictureInPictureFailure) => void,
) {
  const supported = ref(false)
  const active = ref(false)
  const pending = ref(false)
  const mediaRevision = ref(0)
  let generation = 0
  let disposed = false
  let exitPendingFor: HTMLVideoElement | null = null

  /** 始终以浏览器实际所属视频同步状态，包含用户从系统窗口关闭的情况。 */
  function sync() {
    active.value = Boolean(video.value && document.pictureInPictureElement === video.value)
    mediaRevision.value++
  }

  /** 接口存在与媒体就绪是独立条件；暂停的视频也允许进入。 */
  const canEnter = computed(() => {
    void mediaRevision.value
    const element = video.value
    return Boolean(supported.value && source.value && element && !element.disablePictureInPicture &&
      element.readyState >= HTMLMediaElement.HAVE_METADATA && element.videoWidth > 0)
  })
  const canToggle = computed(() => !pending.value && (active.value || canEnter.value))

  /** 仅退出指定视频的 PiP，清理失败不会打扰用户或影响其它视频。 */
  function exitFor(element: HTMLVideoElement) {
    if (document.pictureInPictureElement !== element || exitPendingFor === element) return
    exitPendingFor = element
    try {
      void document.exitPictureInPicture()
        .catch(() => { /* 浏览器关闭与卸载可能竞争。 */ })
        .finally(() => { /* 清理完成后允许后续退出。 */ exitPendingFor = null })
    } catch { exitPendingFor = null /* 接口在页面销毁期间可能不可用。 */ }
  }

  /** 直接在用户手势中调用原生接口；异步完成后不触碰已换源或卸载的实例。 */
  async function toggle() {
    const element = video.value
    if (!element || disposed || pending.value || !supported.value) return
    const exiting = document.pictureInPictureElement === element
    if (!exiting && !canEnter.value) {
      reportFailure("notReady")
      return
    }
    const requestGeneration = generation
    pending.value = true
    try {
      if (exiting) {
        exitPendingFor = element
        await document.exitPictureInPicture()
      }
      else await element.requestPictureInPicture()
      if (disposed || requestGeneration !== generation || video.value !== element) {
        exitFor(element)
        return
      }
    } catch (error) {
      if (!disposed && requestGeneration === generation && video.value === element) {
        reportFailure(classifyPictureInPictureFailure(error))
      }
    } finally {
      if (exiting) exitPendingFor = null
      if (!disposed && requestGeneration === generation) {
        pending.value = false
        sync()
      }
    }
  }

  watch([video, source], (_values, _previous, onCleanup) => {
    // 换源令旧请求失效，并在当前元素的媒体及系统 PiP 事件上刷新准入条件。
    generation++
    pending.value = false
    const element = video.value
    const events = ["loadedmetadata", "loadeddata", "emptied", "error", "enterpictureinpicture", "leavepictureinpicture"]
    if (element) for (const event of events) element.addEventListener(event, sync)
    sync()
    onCleanup(() => {
      // 移除旧元素的监听，避免换片后旧事件污染状态。
      if (element) for (const event of events) element.removeEventListener(event, sync)
    })
  }, { immediate: true, flush: "post" })

  onMounted(() => {
    // 采用能力检测，渲染器不假定 Electron 或某个浏览器存在。
    supported.value = document.pictureInPictureEnabled === true &&
      typeof HTMLVideoElement.prototype.requestPictureInPicture === "function"
    sync()
  })
  onBeforeUnmount(() => {
    // 阻止迟到完成发布状态；进入请求若随后成功，会由 toggle 清理归属视频。
    disposed = true
    generation++
    if (video.value) exitFor(video.value)
  })

  return { supported, active, pending, canEnter, canToggle, sync, toggle, exitFor }
}
