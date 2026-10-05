<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Info, Maximize2, Minimize2, Monitor, Repeat2, SkipBack, SkipForward, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { Alert, AlertTitle } from "@/components/ui/alert"
import PlayerTransportControls from "@/components/jav-library/PlayerTransportControls.vue"
import PlayerPlaybackSettingsMenu from "@/components/jav-library/PlayerPlaybackSettingsMenu.vue"
import MoviePartSelect from "@/components/jav-library/MoviePartSelect.vue"
import { usePlayerImmersiveChrome } from "@/lib/player-immersive-chrome"
import { shouldIgnoreGlobalPlaybackHotkeysForTarget } from "@/lib/player-shortcuts"
import type { DesktopPlaybackCommand, DesktopPlaybackSnapshot } from "../../electron/playback-contract"
import { nativeMessages, type NativeMessage } from "../native-player-prototype/messages"

const bridge = window.curatedPlayer
const snapshot = ref<DesktopPlaybackSnapshot>()
const busy = ref(false)
const error = ref("")
const diagnostics = ref(false)
const settings = ref(false)
const focusedControl = ref(false)
const draft = ref<number>()
const { locale } = useI18n()
const lang = computed(() => snapshot.value?.locale === "en-US" ? "en" : snapshot.value?.locale === "ja-JP" ? "ja" : "zh-CN")
const extra = computed(() => lang.value === "en" ? { web: "Use Web player", auto: "Auto-advance", previous: "Previous movie", next: "Next movie" }
  : lang.value === "ja" ? { web: "Web プレイヤーに切り替え", auto: "自動連続再生", previous: "前の作品", next: "次の作品" }
  : { web: "切换到 Web 播放器", auto: "自动连播", previous: "上一部影片", next: "下一部影片" })
function t(key: NativeMessage) { return nativeMessages[lang.value][key] }
const state = computed(() => snapshot.value?.state)
const playing = computed(() => state.value?.status === "playing")
const active = computed(() => state.value?.status === "playing" || state.value?.status === "paused")
const immersive = usePlayerImmersiveChrome({ hasPlayback: active, isPlaying: playing })
const chromeShown = computed(() => immersive.chromeVisible.value || diagnostics.value || settings.value || focusedControl.value || busy.value || Boolean(error.value))
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
  if (previous?.sessionId !== next.sessionId) { error.value = ""; immersive.revealChrome() }
}
async function command(input: DesktopPlaybackCommand) {
  if (!bridge || !snapshot.value || busy.value) return
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
  volumeTimer = setTimeout(() => { void command({ action: "volume", value }) }, 120)
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
  if (shouldIgnoreGlobalPlaybackHotkeysForTarget(event.target)) return
  const key = event.key.toLowerCase()
  const action = key === " " || key === "k" ? toggle : key === "arrowleft" || key === "j" ? () => seek(-10)
    : key === "arrowright" || key === "l" ? () => seek(10) : key === "m" ? mute
    : key === "f" ? () => command({ action: "fullscreen" }) : key === "escape" && snapshot.value?.fullscreen ? () => command({ action: "fullscreen" })
    : key === "arrowup" ? () => command({ action: "volume", value: Math.min(100, (state.value?.volume ?? 0) + 5) })
    : key === "arrowdown" ? () => command({ action: "volume", value: Math.max(0, (state.value?.volume ?? 0) - 5) })
    : key === "d" ? () => { diagnostics.value = !diagnostics.value } : undefined
  if (action) { event.preventDefault(); void action(); immersive.revealChrome() }
}
onMounted(async () => {
  if (!bridge) { error.value = t("failure"); return }
  unsubscribe = bridge.subscribe(receive)
  receive(await bridge.snapshot())
  window.addEventListener("keydown", keydown)
})
onBeforeUnmount(() => {
  unsubscribe?.(); clearTimeout(clickTimer); clearTimeout(volumeTimer)
  immersive.dispose()
  window.removeEventListener("keydown", keydown)
})
</script>

<template>
  <main class="native-player-surface dark relative h-screen overflow-hidden text-white" @pointermove="immersive.revealChrome" @focusin="focusIn" @focusout="focusOut">
    <button type="button" tabindex="-1" class="absolute inset-0 outline-none" :aria-label="playing ? t('pause') : t('resume')" @click="click" @dblclick="doubleClick" />
    <header class="pointer-events-none absolute inset-x-0 top-0 flex items-center justify-between gap-4 bg-gradient-to-b from-black/80 to-transparent p-5 pb-12 transition-opacity" :class="chromeShown ? 'opacity-100' : 'opacity-0'">
      <div class="min-w-0"><p class="truncate text-lg font-medium">{{ snapshot?.movie?.title || snapshot?.movie?.code || 'Curated' }}</p><p class="text-sm text-white/60">{{ snapshot?.movie?.code }}</p></div>
      <div class="pointer-events-auto flex items-center gap-2">
        <MoviePartSelect v-if="files.length > 1" :files="files" :model-value="state?.fileId" :disabled="busy" @update:model-value="command({ action: 'part', fileId: $event })" />
        <Button variant="ghost" size="icon" :aria-label="t('close')" @click="command({ action: 'close' })"><X /></Button>
      </div>
    </header>
    <div v-if="state?.status === 'starting'" class="pointer-events-none absolute inset-0 flex items-center justify-center">{{ t('starting') }}</div>
    <div v-if="state && ['ended', 'stopped', 'error'].includes(state.status)" class="absolute inset-0 flex items-center justify-center">
      <Button variant="secondary" @click="toggle">{{ t('replay') }}</Button>
    </div>
    <aside v-if="diagnostics" class="absolute left-5 top-24 rounded-2xl border border-white/15 bg-black/85 p-4 text-sm">
      <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
        <dt>{{ t('codec') }}</dt><dd>{{ state?.codec || '—' }}</dd><dt>{{ t('hwdec') }}</dt><dd>{{ state?.hwdec || '—' }}</dd>
        <dt>{{ t('dropped') }}</dt><dd>{{ state?.droppedFrames }}</dd><dt>{{ t('decoderDropped') }}</dt><dd>{{ state?.decoderDroppedFrames }}</dd>
      </dl>
    </aside>
    <footer class="absolute inset-x-0 bottom-0 flex flex-col gap-3 bg-gradient-to-t from-black/90 via-black/55 to-transparent p-4 pt-12 transition-opacity sm:p-5 sm:pt-12" :class="chromeShown ? 'opacity-100' : 'pointer-events-none opacity-0 focus-within:pointer-events-auto focus-within:opacity-100'">
      <Alert v-if="error || state?.error || state?.progressError" variant="destructive"><AlertTitle>{{ error || state?.error || t('saveFailed') }}</AlertTitle></Alert>
      <div class="flex items-center justify-between gap-3 text-sm text-white/80 tabular-nums"><span>{{ time(draft ?? state?.positionSec ?? 0) }} / {{ time(state?.durationSec ?? 0) }}</span><span class="max-w-[55%] truncate">{{ currentFile?.fileName }}</span></div>
      <Slider :model-value="[draft ?? state?.positionSec ?? 0]" :max="Math.max(1, state?.durationSec ?? 0)" :step="0.1" :disabled="!active || busy" :aria-label="t('time')" @update:model-value="draft = $event?.[0]" @value-commit="command({ action: 'seek', value: $event?.[0] }); draft = undefined" />
      <PlayerTransportControls :playing="playing" :disabled="!active || busy" :volume-values="[state?.volume ?? 100]" :volume-percent="state?.volume ?? 100" :muted="state?.volume === 0" :labels="labels" @toggle="toggle" @seek-back="seek(-10)" @seek-forward="seek(10)" @mute="mute" @volume="volume">
        <Button variant="ghost" size="icon" :disabled="queueIndex <= 0 || busy" :aria-label="extra.previous" @click="command({ action: 'movie', movieId: snapshot!.queue[queueIndex - 1]! })"><SkipBack /></Button>
        <Button variant="ghost" size="icon" :disabled="queueIndex < 0 || queueIndex >= (snapshot?.queue.length ?? 0) - 1 || busy" :aria-label="extra.next" @click="command({ action: 'movie', movieId: snapshot!.queue[queueIndex + 1]! })"><SkipForward /></Button>
        <Button variant="ghost" size="icon" :aria-label="extra.auto" :aria-pressed="snapshot?.autoAdvance" :class="snapshot?.autoAdvance ? 'text-primary' : ''" @click="command({ action: 'autoAdvance', enabled: !snapshot?.autoAdvance })"><Repeat2 /></Button>
        <PlayerPlaybackSettingsMenu native :disabled="!active || busy" :playback-rate="state?.speed ?? 1" @update:playback-rate="command({ action: 'speed', value: $event })" @update:open="settings = $event" />
        <Button variant="ghost" size="icon" :aria-label="extra.web" :title="extra.web" @click="command({ action: 'web' })"><Monitor /></Button>
        <Button variant="ghost" size="icon" :aria-label="t('diagnostics')" :aria-pressed="diagnostics" @click="diagnostics = !diagnostics"><Info /></Button>
        <Button variant="ghost" size="icon" :aria-label="snapshot?.fullscreen ? t('exitFullscreen') : t('fullscreen')" @click="command({ action: 'fullscreen' })"><Minimize2 v-if="snapshot?.fullscreen" /><Maximize2 v-else /></Button>
      </PlayerTransportControls>
    </footer>
  </main>
</template>
