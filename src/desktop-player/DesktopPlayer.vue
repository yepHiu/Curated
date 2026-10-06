<script setup lang="ts">
import { computed, nextTick, onMounted, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useElementSize } from "@vueuse/core"
import { Camera, Film, Info, Loader2, Maximize2, Minimize2, Monitor, Repeat2, SkipBack, SkipForward, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { Alert, AlertTitle } from "@/components/ui/alert"
import { Dialog, DialogClose, DialogContent, DialogTitle } from "@/components/ui/dialog"
import CaptureReceipt from "@/components/jav-library/CaptureReceipt.vue"
import FrameImageViewer from "@/components/jav-library/FrameImageViewer.vue"
import PlayerTransportControls from "@/components/jav-library/PlayerTransportControls.vue"
import PlayerPlaybackSettingsMenu from "@/components/jav-library/PlayerPlaybackSettingsMenu.vue"
import MoviePartSelect from "@/components/jav-library/MoviePartSelect.vue"
import NativePlaybackInfo from "@/components/jav-library/NativePlaybackInfo.vue"
import NativeClipCaptureFeedback from "@/components/jav-library/NativeClipCaptureFeedback.vue"
import { nativePlaybackInfoMessages } from "@/lib/native-playback-info"
import { usePlayerImmersiveChrome } from "@/lib/player-immersive-chrome"
import { formatCuratedCaptureKeyLabel, shouldIgnoreGlobalPlaybackHotkeysForTarget } from "@/lib/player-shortcuts"
import { useNativeFrameCapture } from "./use-native-frame-capture"
import type { DesktopPlaybackCommand, DesktopPlaybackSnapshot } from "../../electron/playback-contract"
import { nativeMessages, type NativeMessage } from "../native-player-prototype/messages"

const bridge = window.curatedPlayer
const snapshot = ref<DesktopPlaybackSnapshot>()
const busy = ref(false)
const error = ref("")
const diagnostics = ref(false)
const infoBusy = ref(false)
const infoFeedback = ref("")
const settings = ref(false)
const focusedControl = ref(false)
const draft = ref<number>()
const surface = ref<HTMLElement>()
const footer = ref<HTMLElement>()
const playbackControls = ref<HTMLElement>()
const { height: footerHeight } = useElementSize(footer, { width: 0, height: 0 }, { box: "border-box" })
const { height: controlsHeight } = useElementSize(playbackControls, { width: 0, height: 0 }, { box: "border-box" })
const { width: surfaceWidth } = useElementSize(surface)
// 对齐 Web 的回执位置，只避让实际进度条/按钮，不把渐变留白与时间行算作控件。
const feedbackClearance = computed(() => controlsHeight.value + (surfaceWidth.value >= 640 ? 20 : 16) + 12)
const receiptBottom = computed(() => Math.max(surfaceWidth.value >= 640 ? 96 : 176, feedbackClearance.value))
const clipBottom = computed(() => Math.max(128, feedbackClearance.value))
const { locale, t: uiT } = useI18n()
const lang = computed(() => snapshot.value?.locale === "en-US" ? "en" : snapshot.value?.locale === "ja-JP" ? "ja" : "zh-CN")
const extra = computed(() => lang.value === "en" ? { web: "Use Web player", auto: "Auto-advance", previous: "Previous movie", next: "Next movie" }
  : lang.value === "ja" ? { web: "Web プレイヤーに切り替え", auto: "自動連続再生", previous: "前の作品", next: "次の作品" }
  : { web: "切换到 Web 播放器", auto: "自动连播", previous: "上一部影片", next: "下一部影片" })
function t(key: NativeMessage) { return nativeMessages[lang.value][key] }
const capture = useNativeFrameCapture(bridge, snapshot, code => t(code === "CAPTURE_SAVE_FAILED" ? "captureSaveFailed"
  : code === "CAPTURE_TOO_LARGE" ? "captureTooLarge" : code === "CAPTURE_NOT_READY" ? "captureNotReady" : "captureFailed"))
const captureLabel = computed(() => `${t("capture")} (${formatCuratedCaptureKeyLabel(capture.keyCode.value)})`)
const clipFeedback = capture.clipFeedback
const state = computed(() => snapshot.value?.state)
const playing = computed(() => state.value?.status === "playing")
const active = computed(() => state.value?.status === "playing" || state.value?.status === "paused")
const immersive = usePlayerImmersiveChrome({ hasPlayback: active, isPlaying: playing })
const chromeShown = computed(() => immersive.chromeVisible.value || diagnostics.value || settings.value || focusedControl.value || busy.value || capture.busy.value || Boolean(clipFeedback.value) || Boolean(capture.receipt.value) || capture.previewOpen.value || Boolean(error.value))
const files = computed(() => snapshot.value?.movie?.files.map((file, index) => ({ ...file, partIndex: index + 1 })) ?? [])
const currentFile = computed(() => files.value.find(file => file.id === state.value?.fileId))
const queueIndex = computed(() => snapshot.value?.queue.indexOf(snapshot.value?.movie?.id ?? "") ?? -1)
const labels = computed(() => ({ play: t("resume"), pause: t("pause"), seekBack: t("seekBack"), seekForward: t("seekForward"),
  volume: t("volume"), mute: t("mute"), unmute: t("unmute") }))
let unsubscribe: (() => void) | undefined
let clickTimer: ReturnType<typeof setTimeout> | undefined
let volumeTimer: ReturnType<typeof setTimeout> | undefined
let unmutedVolume = 100
function receive(next: DesktopPlaybackSnapshot) {
  if (snapshot.value && snapshot.value.revision > next.revision) return
  const previous = snapshot.value
  snapshot.value = next
  locale.value = lang.value
  document.documentElement.lang = lang.value
  if (previous?.sessionId !== next.sessionId) { error.value = ""; infoFeedback.value = ""; immersive.revealChrome() }
}
function closeDiagnostics() {
  diagnostics.value = false
  void nextTick(() => surface.value?.querySelector<HTMLButtonElement>("[data-native-info-trigger]")?.focus())
}
async function exportDiagnostics(action: "copy" | "save") {
  const sessionId = snapshot.value?.sessionId
  if (!bridge || !sessionId || infoBusy.value) return
  infoBusy.value = true; infoFeedback.value = ""
  const labels = nativePlaybackInfoMessages[lang.value]
  try {
    const result = action === "copy" ? await bridge.copyDiagnostics(sessionId) : await bridge.saveDiagnostics(sessionId)
    if (snapshot.value?.sessionId === sessionId && result !== "cancelled") infoFeedback.value = action === "copy" ? labels.copied : labels.saved
  } catch {
    if (snapshot.value?.sessionId === sessionId) infoFeedback.value = labels.failed
  } finally { infoBusy.value = false }
}
async function command(input: DesktopPlaybackCommand) {
  if (!bridge || !snapshot.value || busy.value) return
  if (["seek", "part", "movie", "web", "stop", "close", "replay"].includes(input.action)) capture.cancelPress()
  busy.value = true; error.value = ""
  try { await bridge.command(snapshot.value.sessionId, input); receive(await bridge.snapshot()) }
  catch (failure) {
    const code = failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) : undefined
    error.value = `${t("failure")}${code ? ` · ${code}` : ""}`
  } finally { busy.value = false }
}
function time(value: number) {
  const total = Math.max(0, Math.floor(value))
  return `${total >= 3600 ? `${Math.floor(total / 3600)}:` : ""}${String(Math.floor(total / 60) % 60).padStart(total >= 3600 ? 2 : 1, "0")}:${String(total % 60).padStart(2, "0")}`
}
async function toggle() {
  await command(active.value ? { action: playing.value ? "pause" : "resume" } : { action: "replay" })
  immersive.revealChrome()
}
async function seek(delta: number) {
  await command({ action: "seek", value: Math.max(0, Math.min(state.value?.durationSec ?? 0, (state.value?.positionSec ?? 0) + delta)) })
  immersive.showSeekFeedback(delta)
}
function volume(values?: number[]) {
  const value = values?.[0]
  if (value === undefined) return
  clearTimeout(volumeTimer)
  volumeTimer = setTimeout(() => {
    const current = snapshot.value
    if (!bridge || !current) return
    // 尾端值必须入 main 的串行队列，不能因另一个 UI 动作 busy 而丢弃。
    void bridge.command(current.sessionId, { action: "volume", value }).then(async () => receive(await bridge.snapshot()))
      .catch(() => { error.value = t("failure") })
  }, 120)
}
async function mute() {
  const value = state.value?.volume ?? 100
  if (value > 0) unmutedVolume = value
  await command({ action: "volume", value: value > 0 ? 0 : unmutedVolume })
}
function click() {
  clearTimeout(clickTimer)
  clickTimer = setTimeout(() => { void toggle() }, 220)
}
function doubleClick() { clearTimeout(clickTimer); void command({ action: "fullscreen" }) }
function focusIn(event: FocusEvent) { focusedControl.value = event.target instanceof HTMLElement && event.target.tabIndex >= 0; immersive.revealChrome() }
function focusOut(event: FocusEvent) { focusedControl.value = event.relatedTarget instanceof HTMLElement && event.relatedTarget.tabIndex >= 0 }
function keydown(event: KeyboardEvent) {
  if (event.key === "Escape" && diagnostics.value && !capture.previewOpen.value) { event.preventDefault(); closeDiagnostics(); return }
  if (event.ctrlKey || event.altKey || event.metaKey || event.shiftKey || capture.previewOpen.value || shouldIgnoreGlobalPlaybackHotkeysForTarget(event.target)) return
  if (event.code === capture.keyCode.value) {
    event.preventDefault()
    if (!event.repeat && !busy.value) capture.startPress()
    immersive.revealChrome()
    return
  }
  const key = event.key.toLowerCase()
  const action = key === " " || key === "k" ? toggle : key === "arrowleft" || key === "j" ? () => seek(-10)
    : key === "arrowright" || key === "l" ? () => seek(10) : key === "m" ? mute
    : key === "f" ? () => command({ action: "fullscreen" }) : key === "escape" && snapshot.value?.fullscreen ? () => command({ action: "fullscreen" })
    : key === "arrowup" ? () => command({ action: "volume", value: Math.min(100, (state.value?.volume ?? 0) + 5) })
    : key === "arrowdown" ? () => command({ action: "volume", value: Math.max(0, (state.value?.volume ?? 0) - 5) })
    : key === "d" ? () => { diagnostics.value = !diagnostics.value } : undefined
  if (action) { event.preventDefault(); void action(); immersive.revealChrome() }
}
function keyup(event: KeyboardEvent) {
  if (event.code !== capture.keyCode.value) return
  event.preventDefault(); capture.finishPress()
}
function startCapture(event: PointerEvent) {
  if (event.button !== 0 || busy.value) return
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  capture.startPress()
}
function captureClick(event: MouseEvent) {
  if (event.detail === 0) void capture.capture()
}
function toggleGif() {
  if (capture.recording.value || capture.phase.value === "armed") capture.finishPress()
  else capture.startPress()
}
function cancelGif() {
  if (capture.recording.value) capture.cancelPress()
  else void capture.clipAction("cancel")
}
onMounted(async () => {
  if (!bridge) { error.value = t("failure"); return }
  unsubscribe = bridge.subscribe(receive)
  receive(await bridge.snapshot())
  await capture.preferences()
  window.addEventListener("keydown", keydown)
  window.addEventListener("keyup", keyup)
  window.addEventListener("blur", capture.cancelPress)
  window.addEventListener("focus", capture.preferences)
})
onBeforeUnmount(() => {
  unsubscribe?.(); clearTimeout(clickTimer); clearTimeout(volumeTimer)
  immersive.dispose()
  window.removeEventListener("keydown", keydown)
  window.removeEventListener("keyup", keyup)
  window.removeEventListener("blur", capture.cancelPress)
  window.removeEventListener("focus", capture.preferences)
})
</script>

<template>
  <main ref="surface" class="native-player-surface dark relative h-screen overflow-hidden text-white" @pointermove="immersive.revealChrome" @focusin="focusIn" @focusout="focusOut">
    <button type="button" tabindex="-1" class="absolute inset-0 outline-none" :aria-label="playing ? t('pause') : t('resume')" @click="click" @dblclick="doubleClick" />
    <header class="pointer-events-none absolute inset-x-0 top-0 flex items-center justify-between gap-4 bg-gradient-to-b from-black/80 to-transparent p-4 pb-12 transition-opacity sm:p-5 sm:pb-12" :class="chromeShown ? 'opacity-100' : 'opacity-0 focus-within:opacity-100'">
      <div class="min-w-0"><p class="truncate text-lg font-medium">{{ snapshot?.movie?.title || snapshot?.movie?.code || 'Curated' }}</p><p class="truncate text-sm text-white/60">{{ snapshot?.movie?.code }}</p></div>
      <div class="pointer-events-auto flex shrink-0 items-center">
        <MoviePartSelect v-if="files.length > 1" :files="files" :model-value="state?.fileId" :disabled="busy" @update:model-value="command({ action: 'part', fileId: $event })" />
      </div>
    </header>
    <div v-if="state?.status === 'starting'" class="pointer-events-none absolute inset-0 flex items-center justify-center">{{ t('starting') }}</div>
    <div v-if="immersive.feedback.value && !chromeShown" class="pointer-events-none absolute inset-0 flex items-center justify-center"><span class="rounded-2xl bg-black/60 px-6 py-4 text-xl">{{ immersive.feedback.value.label }}</span></div>
    <div v-if="state && ['ended', 'stopped', 'error'].includes(state.status)" class="absolute inset-0 flex items-center justify-center">
      <Button variant="secondary" @click="toggle">{{ t('replay') }}</Button>
    </div>
    <NativePlaybackInfo v-if="diagnostics" :state="state" :lang="lang" :busy="infoBusy" :feedback="infoFeedback"
      :style="{ bottom: `${footerHeight + 12}px` }" @close="closeDiagnostics" @copy="exportDiagnostics('copy')" @save="exportDiagnostics('save')" />
    <NativeClipCaptureFeedback v-if="clipFeedback" :recording="capture.recording.value" :elapsed-sec="capture.elapsedSec.value" :progress="capture.progress.value" :clip="capture.clipStatus.value" :busy="capture.busy.value"
      :style="{ bottom: `${clipBottom}px` }" @cancel="cancelGif" @retry="capture.clipAction('retry')" @dismiss="capture.dismiss" />
    <CaptureReceipt v-if="capture.receipt.value && !clipFeedback" :job="capture.receipt.value" :pending="capture.busy.value ? 1 : 0" :retryable="capture.retryable.value"
      :style="{ bottom: `${receiptBottom}px` }"
      @retry="capture.capture(true)" @view="capture.previewOpen.value = true" @dismiss="capture.dismiss" />
    <p class="sr-only" role="status" aria-live="polite">{{ capture.receipt.value ? uiT(capture.busy.value ? 'curated.captureSaving' : capture.receipt.value.committed ? 'curated.captureSaved' : 'curated.captureFailed') : '' }} {{ capture.receipt.value?.error }}</p>
    <Dialog v-model:open="capture.previewOpen.value">
      <DialogContent :portal-to="surface" :show-close-button="false" minimal-motion class="flex max-h-[90vh] flex-col sm:max-w-[90vw]" :aria-describedby="undefined">
        <DialogTitle>{{ uiT('curated.captureView') }}</DialogTitle>
        <DialogClose as-child><Button variant="ghost" size="icon" class="absolute right-3 top-3 rounded-full" :aria-label="uiT('common.close')"><X /></Button></DialogClose>
        <div class="h-[75vh]"><FrameImageViewer :src="capture.receipt.value?.preview ?? ''" :alt="capture.receipt.value?.movie.code ?? ''" /></div>
      </DialogContent>
    </Dialog>
    <footer ref="footer" class="absolute inset-x-0 bottom-0 flex flex-col gap-3 bg-gradient-to-t from-black/90 via-black/55 to-transparent p-4 pt-12 transition-opacity sm:p-5 sm:pt-12" :class="chromeShown ? 'opacity-100' : 'pointer-events-none opacity-0 focus-within:pointer-events-auto focus-within:opacity-100'">
      <Alert v-if="error || state?.error || state?.progressError" variant="destructive"><AlertTitle>{{ error || state?.error || t('saveFailed') }}</AlertTitle></Alert>
      <div class="flex items-center justify-between gap-3 text-sm text-white/80 tabular-nums"><span>{{ time(draft ?? state?.positionSec ?? 0) }} / {{ time(state?.durationSec ?? 0) }}</span><span class="max-w-[55%] truncate">{{ currentFile?.fileName }}</span></div>
      <div ref="playbackControls" class="flex flex-col gap-3">
        <Slider :model-value="[draft ?? state?.positionSec ?? 0]" :max="Math.max(1, state?.durationSec ?? 0)" :step="0.1" :disabled="!active || busy" :aria-label="t('time')" @update:model-value="draft = $event?.[0]" @value-commit="command({ action: 'seek', value: $event?.[0] }); draft = undefined" />
        <PlayerTransportControls :playing="playing" :disabled="!active || busy" :volume-values="[state?.volume ?? 100]" :volume-percent="state?.volume ?? 100" :muted="state?.volume === 0" :labels="labels" @toggle="toggle" @seek-back="seek(-10)" @seek-forward="seek(10)" @mute="mute" @volume="volume">
        <Button variant="ghost" size="icon" class="rounded-full" :disabled="!capture.available.value || busy" :aria-label="captureLabel" :title="captureLabel" @pointerdown="startCapture" @pointerup="capture.finishPress" @pointercancel="capture.cancelPress" @click="captureClick"><Loader2 v-if="capture.busy.value" class="animate-spin motion-reduce:animate-none" /><Camera v-else /></Button>
        <Button variant="ghost" size="icon" class="rounded-full" :disabled="((!playing || !capture.available.value) && !capture.recording.value) || busy" :aria-label="capture.recording.value ? uiT('curated.stopClip') : 'GIF'" :title="capture.recording.value ? uiT('curated.stopClip') : 'GIF'" :aria-pressed="capture.recording.value" @click="toggleGif"><Film /></Button>
        <Button variant="ghost" size="icon" class="rounded-full" :disabled="queueIndex <= 0 || busy" :aria-label="extra.previous" @click="command({ action: 'movie', movieId: snapshot!.queue[queueIndex - 1]! })"><SkipBack /></Button>
        <Button variant="ghost" size="icon" class="rounded-full" :disabled="queueIndex < 0 || queueIndex >= (snapshot?.queue.length ?? 0) - 1 || busy" :aria-label="extra.next" @click="command({ action: 'movie', movieId: snapshot!.queue[queueIndex + 1]! })"><SkipForward /></Button>
        <Button variant="ghost" size="icon" class="rounded-full" :aria-label="extra.auto" :aria-pressed="snapshot?.autoAdvance" @click="command({ action: 'autoAdvance', enabled: !snapshot?.autoAdvance })"><Repeat2 /></Button>
        <PlayerPlaybackSettingsMenu native :disabled="!active || busy" :playback-rate="state?.speed ?? 1" @update:playback-rate="command({ action: 'speed', value: $event })" @update:open="settings = $event" />
        <Button variant="ghost" size="icon" class="rounded-full" :aria-label="extra.web" :title="extra.web" @click="command({ action: 'web' })"><Monitor /></Button>
        <Button data-native-info-trigger variant="ghost" size="icon" class="rounded-full" :aria-label="nativePlaybackInfoMessages[lang].title" :aria-pressed="diagnostics" @click="diagnostics = !diagnostics"><Info /></Button>
        <Button variant="ghost" size="icon" class="rounded-full" :aria-label="snapshot?.fullscreen ? t('exitFullscreen') : t('fullscreen')" :aria-pressed="snapshot?.fullscreen" @click="command({ action: 'fullscreen' })"><Minimize2 v-if="snapshot?.fullscreen" /><Maximize2 v-else /></Button>
        </PlayerTransportControls>
      </div>
    </footer>
  </main>
</template>

<style scoped>
/* 原生 HUD 局部表面：沿用应用语义色，不改变共享 Web 控件。 */
:deep([data-movie-part-select]) {
  background-color: var(--surface-muted);
  color: var(--foreground);
  border-color: var(--border);
}
:deep([data-movie-part-select]:hover) { background-color: var(--accent); }
footer :deep(button[aria-pressed="true"]),
footer :deep(button[data-state="open"]) {
  background-color: color-mix(in srgb, var(--primary) 18%, transparent);
  color: var(--primary);
}
</style>
