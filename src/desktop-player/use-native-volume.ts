import { computed, onScopeDispose, ref, watch, type Ref } from "vue"
import type { DesktopPlayerBridge, DesktopPlaybackSnapshot } from "../../electron/playback-contract"

/** Keep the drag value visible while the native engine acknowledges volume commands. */
export function useNativeVolume(
  bridge: Pick<DesktopPlayerBridge, "command" | "snapshot"> | undefined,
  snapshot: Ref<DesktopPlaybackSnapshot | undefined>,
  receive: (value: DesktopPlaybackSnapshot) => void,
  onError: () => void,
) {
  const draft = ref<number>()
  const percent = computed(() => draft.value ?? snapshot.value?.state.volume ?? 100)
  let timer: ReturnType<typeof setTimeout> | undefined
  let revision = 0

  function cancel() {
    clearTimeout(timer)
    revision++
    draft.value = undefined
  }

  function update(values?: number[]) {
    const raw = values?.[0]
    const sessionId = snapshot.value?.sessionId
    if (!bridge || !sessionId || raw === undefined || !Number.isFinite(raw)) return
    const value = Math.max(0, Math.min(100, Math.round(raw)))
    const currentRevision = ++revision
    draft.value = value
    clearTimeout(timer)
    timer = setTimeout(async () => {
      try {
        await bridge.command(sessionId, { action: "volume", value })
        const next = await bridge.snapshot()
        if (sessionId !== snapshot.value?.sessionId) return
        receive(next)
      } catch {
        if (currentRevision === revision) onError()
      } finally {
        if (currentRevision === revision) draft.value = undefined
      }
    }, 120)
  }

  watch(() => snapshot.value?.sessionId, cancel, { flush: "sync" })
  onScopeDispose(cancel)
  return { percent, update, cancel }
}
