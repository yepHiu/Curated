<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { Play, MonitorPlay, FolderOpen, Info, Maximize2, Minimize2, Loader2, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectGroup, SelectItem } from "@/components/ui/select"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { Slider } from "@/components/ui/slider"
import PlayerTransportControls from "@/components/jav-library/PlayerTransportControls.vue"
import PlayerPlaybackSettingsMenu from "@/components/jav-library/PlayerPlaybackSettingsMenu.vue"
import MoviePartSelect from "@/components/jav-library/MoviePartSelect.vue"
import { usePlayerImmersiveChrome } from "@/lib/player-immersive-chrome"
import { shouldIgnoreGlobalPlaybackHotkeysForTarget } from "@/lib/player-shortcuts"
import { getPlayerAudioPrefs, savePlayerAudioPrefs } from "@/lib/player-volume-storage"
import { resolveInitialLocale } from "@/lib/locale-storage"
import { nativeMessages, type NativeMessage } from "./messages"
import { emptyNativePlayerState, type NativeLabControl, type NativeLabMovie, type NativeLabStatus } from "../../electron/native-player-contract"

const bridge = window.nativePlayerLab
const locale = resolveInitialLocale()
/** 原型文案与现有客户端语言一致，诊断码保留技术标识。 */
function t(key: NativeMessage): string { return nativeMessages[locale][key] }
const status = ref<NativeLabStatus>({ engineReady: false, state: emptyNativePlayerState() })
const origin = ref("http://127.0.0.1:8080")
const pin = ref("")
const query = ref("")
const movies = ref<NativeLabMovie[]>([])
const movie = ref<NativeLabMovie>()
const fileId = ref("")
const startSec = ref(0)
const audioPrefs = getPlayerAudioPrefs()
const volume = ref(audioPrefs.volumePercent)
const muted = ref(audioPrefs.muted)
const preferredSpeed = ref(1)
const seekDraft = ref<number>()
const prepareOpen = ref(true)
const diagnosticsOpen = ref(false)
const settingsOpen = ref(false)
const playingMovie = ref<NativeLabMovie>()
const busy = ref(false)
const error = ref("")
const searched = ref(false)
let interval: ReturnType<typeof setInterval> | undefined
let polling = false
let volumeTimer: ReturnType<typeof setTimeout> | undefined
let clickTimer: ReturnType<typeof setTimeout> | undefined
const unlocked = computed(() => { /* Server 的解锁状态决定业务控件准入。 */ return status.value.connection?.unlocked === true })
const active = computed(() => { /* 控制按钮不对已结束或失败的进程继续发命令。 */ return ["playing", "paused"].includes(status.value.state.status) })
const playing = computed(() => { /* 自动隐藏只在实际播放时生效。 */ return status.value.state.status === "playing" })
const immersive = usePlayerImmersiveChrome({ hasPlayback: active, isPlaying: playing })
const chromeShown = computed(() => { /* 弹窗、菜单与键盘焦点保留交互控件。 */ return immersive.chromeVisible.value || prepareOpen.value || diagnosticsOpen.value || settingsOpen.value })
const currentFile = computed(() => { /* 标题跟随实际播放文件，而非准备区的待播选择。 */ return playingMovie.value?.files.find((file) => { return file.id === status.value.state.fileId }) })
const partFiles = computed(() => { /* 共享分段控件只需要显示身份和名称。 */ return playingMovie.value?.files.map((file, index) => { return { ...file, partIndex: index + 1 } }) ?? [] })
const transportLabels = computed(() => { /* 共享控件接受显示文案，不依赖特定播放引擎。 */
  return { play: t("resume"), pause: t("pause"), seekBack: t("seekBack"), seekForward: t("seekForward"), volume: t("volume"), mute: t("mute"), unmute: t("unmute") }
})

/** 时间显示与现有播放器一致，长片包含小时。 */
function timeLabel(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds))
  const minutes = Math.floor(total / 60)
  return total >= 3600 ? `${Math.floor(total / 3600)}:${String(minutes % 60).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`
    : `${minutes}:${String(total % 60).padStart(2, "0")}`
}

/** 轮询只读状态，每轮完成前不发下一轮；窗口销毁会移除计时器。 */
async function refresh(): Promise<void> {
  if (!bridge || polling) return
  polling = true
  try { status.value = await bridge.getStatus() } finally { polling = false }
}

/** 将失败压成诊断码，不把 IPC 堆栈或敏感请求参数显示在页面。 */
async function perform(action: () => Promise<unknown>): Promise<void> {
  if (!bridge || busy.value) return
  busy.value = true
  error.value = ""
  try { await action(); await refresh() }
  catch (failure) {
    const code = failure instanceof Error ? failure.message.match(/\b[A-Z][A-Z_0-9]{3,}\b/g)?.at(-1) : undefined
    error.value = `${t("failure")}${code ? ` · ${code}` : ""}`
  } finally { busy.value = false }
}

/** 切换 Server 后清除来自旧来源的影片选择。 */
async function connect(): Promise<void> {
  await perform(async () => {
    // 连接成功前保留当前选择，失败不会伪造新来源状态。
    await bridge!.connect(origin.value)
    movies.value = []; movie.value = undefined; fileId.value = ""; searched.value = false
  })
}

/** 引擎路径通过原生文件选择器获取，页面只接收就绪结果。 */
async function selectExecutable(): Promise<void> {
  await perform(() => { /* 不接受页面输入的任意可执行路径。 */ return bridge!.selectExecutable() })
}

/** 解锁完成或失败后都清除输入的 PIN。 */
async function unlock(): Promise<void> {
  const value = pin.value
  pin.value = ""
  await perform(() => { /* PIN 只用于当前主进程 session 的一次请求。 */ return bridge!.unlock(value) })
}

/** 检索返回有限的影片列表；空结果显示明确反馈。 */
async function search(): Promise<void> {
  await perform(async () => {
    // 搜索不选择或启动任何影片。
    movies.value = await bridge!.search(query.value)
    searched.value = true
  })
}

/** 选择作品后加载文件，并读取第一片的独立续播位置。 */
async function chooseMovie(value: unknown): Promise<void> {
  if (typeof value !== "string") return
  await perform(async () => {
    // 完整详情不包含 Server 磁盘路径。
    const detail = await bridge!.detail(value)
    movie.value = detail
    fileId.value = detail.files[0]?.id ?? ""
    startSec.value = await bridge!.resume({ movieId: detail.id, fileId: fileId.value || undefined })
  })
}

/** 第二片等文件切换只更新待播放选择，不改动当前播放器身份。 */
async function chooseFile(value: unknown): Promise<void> {
  if (typeof value !== "string" || !movie.value) return
  await perform(async () => {
    // 每个文件都从 Server 单独读取进度。
    fileId.value = value
    startSec.value = await bridge!.resume({ movieId: movie.value!.id, fileId: value })
  })
}

/** 明确提交作品、文件及绝对起点到原生会话。 */
async function play(): Promise<void> {
  if (!movie.value) return
  await perform(async () => {
    // renderer 不发送原始媒体地址；成功后才隐藏准备区并更新标题。
    await bridge!.start({ movieId: movie.value!.id, fileId: fileId.value || undefined, startSec: Number(startSec.value) })
    playingMovie.value = movie.value
    await bridge!.control({ action: "volume", value: muted.value ? 0 : volume.value })
    await bridge!.control({ action: "speed", value: preferredSpeed.value })
    prepareOpen.value = false
    immersive.revealChrome()
  })
}

/** 播放中的分段切换读取该文件的独立续播位置。 */
async function openPart(value: string): Promise<void> {
  if (!playingMovie.value) return
  await perform(async () => {
    // 保留当前影片身份，不使用准备区另一部作品的选择。
    const start = await bridge!.resume({ movieId: playingMovie.value!.id, fileId: value })
    await bridge!.start({ movieId: playingMovie.value!.id, fileId: value, startSec: start })
    await bridge!.control({ action: "volume", value: muted.value ? 0 : volume.value })
    await bridge!.control({ action: "speed", value: preferredSpeed.value })
    immersive.revealChrome()
  })
}

/** 停止或结束后重新播放实际打开的文件。 */
async function replay(): Promise<void> {
  if (!playingMovie.value) return
  movie.value = playingMovie.value
  fileId.value = status.value.state.fileId ?? ""
  startSec.value = 0
  await play()
}

/** 控制当前正在播放的会话，与待播放文件选择分开。 */
async function control(action: NativeLabControl["action"], value?: number): Promise<void> {
  await perform(() => { /* 所有控制使用枚举及数值，无通用命令接口。 */ return bridge!.control({ action, value }) })
}

/** 成功应用倍速后跨分段保留；状态菜单继续显示引擎实际速率。 */
async function setPlaybackRate(value: number): Promise<void> {
  await perform(async () => {
    // 不在失败或被拒绝的命令上更新偏好。
    await bridge!.control({ action: "speed", value })
    preferredSpeed.value = value
  })
}

/** 拖动过程中保留预览，只有提交时才发出一次实际寻址。 */
function previewSeek(values?: number[]): void { seekDraft.value = values?.[0] }
/** 进度条提交后恢复引擎的实际位置。 */
async function commitSeek(values?: number[]): Promise<void> {
  const target = values?.[0]
  if (target === undefined) return
  await control("seek", target)
  seekDraft.value = undefined
  if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
  immersive.revealChrome()
}
/** 快进后退共用已有 10 秒默认步长，原生引擎校验绝对位置。 */
async function seekDelta(delta: number): Promise<void> {
  await control("seek", Math.max(0, Math.min(status.value.state.durationSec, status.value.state.positionSec + delta)))
  immersive.showSeekFeedback(delta)
}
/** 播放和暂停使用同一控件及快捷键入口。 */
async function togglePlayback(): Promise<void> {
  if (!active.value) return
  await control(playing.value ? "pause" : "resume")
  immersive.revealChrome()
}
/** 音量合并为尾端命令，不丢弃快速拖动的最终值。 */
function updateVolume(values?: number[]): void {
  if (!values?.length) return
  volume.value = Math.round(Math.max(0, Math.min(100, values[0]!)))
  muted.value = false
  clearTimeout(volumeTimer)
  volumeTimer = setTimeout(() => {
    // 音量命令通过主进程队列串行，与其它播放动作共用实例。
    void bridge!.control({ action: "volume", value: volume.value }).then(() => {
      savePlayerAudioPrefs({ volumePercent: volume.value, muted: false })
      return refresh()
    }).catch(() => { error.value = t("failure") })
  }, 120)
}
/** 静音保留滑条原值，取消静音恢复之前的音量。 */
async function toggleMute(): Promise<void> {
  clearTimeout(volumeTimer)
  const next = !muted.value && volume.value > 0
  await perform(async () => {
    await bridge!.control({ action: "volume", value: next ? 0 : volume.value || 100 })
    muted.value = next
    if (!next && volume.value === 0) volume.value = 100
    savePlayerAudioPrefs({ volumePercent: volume.value, muted: muted.value })
  })
}
/** 只请求受限窗口动作，不把 HWND 暴露给页面。 */
async function windowAction(action: "fullscreen" | "minimize" | "close"): Promise<void> {
  await perform(() => { /* 窗口动作独立于引擎业务队列。 */ return bridge!.windowAction(action) })
}
/** 单击延迟避免双击全屏时额外暂停。 */
function surfaceClick(): void { clearTimeout(clickTimer); clickTimer = setTimeout(() => { void togglePlayback() }, 220) }
/** 双击直接切换宿主全屏。 */
function surfaceDoubleClick(): void { clearTimeout(clickTimer); void windowAction("fullscreen") }
/** 空格、方向键、M/F/D 与现有播放器的快捷键保持一致。 */
function onKeydown(event: KeyboardEvent): void {
  if (prepareOpen.value || event.altKey || event.ctrlKey || event.metaKey || shouldIgnoreGlobalPlaybackHotkeysForTarget(event.target)) return
  if (event.target instanceof HTMLElement && event.target.closest('button:not([data-native-video-surface]), [role="menu"], [role="menuitemradio"]')) return
  if (event.code === "Escape" && status.value.window?.fullscreen) { event.preventDefault(); void windowAction("fullscreen"); return }
  if (!active.value || busy.value) return
  const actions: Record<string, () => void> = {
    Space: () => { void togglePlayback() }, KeyK: () => { void togglePlayback() },
    ArrowLeft: () => { void seekDelta(-10) }, KeyJ: () => { void seekDelta(-10) },
    ArrowRight: () => { void seekDelta(10) }, KeyL: () => { void seekDelta(10) },
    ArrowUp: () => { updateVolume([volume.value + 5]) }, ArrowDown: () => { updateVolume([volume.value - 5]) },
    KeyM: () => { void toggleMute() }, KeyF: () => { void windowAction("fullscreen") }, KeyD: () => { diagnosticsOpen.value = !diagnosticsOpen.value },
  }
  if (actions[event.code]) { event.preventDefault(); actions[event.code]!() }
}

onMounted(() => {
  // 页面一进入先读取当前能力，轮询失败使用操作回执显示。
  void perform(async () => {
    // 重新加载控制页时显示实际连接来源，不把默认输入伪装成当前 Server。
    await refresh()
    if (status.value.connection) origin.value = status.value.connection.origin
  })
  interval = setInterval(() => { void refresh().catch(() => { /* 页面销毁竞争时不追加错误提示。 */ }) }, 500)
  document.addEventListener("keydown", onKeydown)
})
onBeforeUnmount(() => {
  // 播放资源由主进程回收，页面只释放计时器和本地事件。
  clearInterval(interval); clearTimeout(volumeTimer); clearTimeout(clickTimer)
  document.removeEventListener("keydown", onKeydown); immersive.dispose()
})
</script>

<template>
  <main class="native-player-surface relative h-full overflow-hidden text-white" @mousemove="immersive.onPageMouseMove" @contextmenu.prevent>
    <!-- 画面层只提供点击命中，不占用重复的 Tab 停靠点；键盘焦点提示保留在播放控件上。 -->
    <button v-if="active" data-native-video-surface type="button" tabindex="-1" class="absolute inset-0 size-full outline-none" :class="chromeShown ? '' : 'cursor-none'" :aria-label="playing ? t('pause') : t('resume')" @click="surfaceClick" @dblclick="surfaceDoubleClick" />
    <div v-if="!active" class="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-background px-6 text-center text-foreground">
      <MonitorPlay class="size-12 text-primary" aria-hidden="true" />
      <h1 class="text-xl font-semibold">{{ status.state.status === 'idle' ? t('title') : t(status.state.status) }}</h1>
      <p class="text-sm text-muted-foreground">{{ status.state.status === 'idle' ? t('idleHint') : playingMovie?.title }}</p>
      <Button v-if="playingMovie && ['ended', 'stopped'].includes(status.state.status)" :disabled="busy" @click="replay"><Play data-icon="inline-start" />{{ t('replay') }}</Button>
      <Button v-else :disabled="busy" @click="prepareOpen = true"><FolderOpen data-icon="inline-start" />{{ t('chooseMedia') }}</Button>
    </div>
    <header class="absolute inset-x-0 top-0 flex flex-wrap items-center gap-2 bg-gradient-to-b from-black/85 via-black/40 to-transparent p-4 transition-opacity sm:flex-nowrap sm:gap-3 sm:p-5" :class="chromeShown ? 'opacity-100' : 'pointer-events-none opacity-0 focus-within:pointer-events-auto focus-within:opacity-100'">
      <div class="min-w-0 flex-1">
        <p class="flex min-w-0 items-baseline text-lg font-semibold leading-tight sm:text-xl">
          <span v-if="playingMovie?.code" class="mr-2 max-w-[35%] shrink-0 truncate font-medium text-white/70">{{ playingMovie.code }}</span>
          <span class="min-w-0 truncate" :title="playingMovie?.title">{{ playingMovie?.title || 'Curated' }}</span>
        </p>
      </div>
      <MoviePartSelect v-if="partFiles.length > 1" :files="partFiles" :model-value="status.state.fileId" :disabled="busy" @update:model-value="openPart" />
      <Button variant="secondary" size="icon" class="shrink-0 rounded-full bg-white/10 text-white hover:bg-white/20" :disabled="busy" :aria-label="t('chooseMedia')" @click="prepareOpen = true"><FolderOpen /></Button>
      <Button v-if="status.window?.fullscreen" variant="secondary" size="icon" class="shrink-0 rounded-full bg-white/10 text-white hover:bg-white/20" :aria-label="t('close')" @click="windowAction('close')"><X /></Button>
    </header>
    <div v-if="busy" role="status" class="pointer-events-none absolute inset-0 flex items-center justify-center gap-3"><Loader2 class="size-8 animate-spin motion-reduce:animate-none" /><span>{{ t('busy') }}</span></div>
    <div v-if="immersive.feedback.value && !chromeShown" class="pointer-events-none absolute inset-0 flex items-center justify-center"><span class="rounded-2xl bg-black/60 px-6 py-4 text-xl">{{ immersive.feedback.value.label }}</span></div>
    <aside v-if="diagnosticsOpen" class="absolute left-5 top-24 max-w-[calc(100%-2.5rem)] rounded-2xl border border-white/15 bg-black/85 p-4 text-sm" :aria-label="t('diagnostics')">
      <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
        <dt>{{ t('codec') }}</dt><dd class="truncate">{{ status.state.codec || '—' }}</dd>
        <dt>{{ t('hwdec') }}</dt><dd>{{ status.state.hwdec || '—' }}</dd>
        <dt>{{ t('dropped') }}</dt><dd>{{ status.state.droppedFrames }}</dd>
        <dt>{{ t('decoderDropped') }}</dt><dd>{{ status.state.decoderDroppedFrames }}</dd>
      </dl>
    </aside>
    <footer v-if="playingMovie" class="absolute inset-x-0 bottom-0 flex flex-col gap-3 bg-gradient-to-t from-black/90 via-black/55 to-transparent p-4 pt-12 transition-opacity sm:p-5 sm:pt-12" :class="chromeShown ? 'opacity-100' : 'pointer-events-none opacity-0 focus-within:pointer-events-auto focus-within:opacity-100'">
      <Alert v-if="!prepareOpen && (error || status.state.error || status.state.progressError)" variant="destructive"><AlertTitle>{{ error || status.state.error || t('saveFailed') }}</AlertTitle></Alert>
      <div class="flex items-center justify-between gap-3 text-sm text-white/80 tabular-nums">
        <span>{{ timeLabel(seekDraft ?? status.state.positionSec) }} / {{ timeLabel(status.state.durationSec) }}</span>
        <span class="max-w-[55%] truncate" :title="currentFile?.fileName">{{ currentFile?.fileName }}</span>
      </div>
      <Slider :model-value="[seekDraft ?? status.state.positionSec]" :max="Math.max(1, status.state.durationSec)" :step="0.1" :disabled="!active || busy" :aria-label="t('time')" @update:model-value="previewSeek" @value-commit="commitSeek" />
      <PlayerTransportControls :playing="playing" :disabled="!active || busy" :volume-values="[muted ? 0 : volume]" :volume-percent="muted ? 0 : volume" :muted="muted || volume === 0" :labels="transportLabels" @toggle="togglePlayback" @seek-back="seekDelta(-10)" @seek-forward="seekDelta(10)" @mute="toggleMute" @volume="updateVolume">
        <PlayerPlaybackSettingsMenu native :disabled="!active || busy" :playback-rate="status.state.speed" @update:playback-rate="setPlaybackRate" @update:open="settingsOpen = $event" />
        <Button variant="secondary" size="icon" class="size-9 shrink-0 rounded-full bg-white/10 text-white hover:bg-white/20" :aria-label="t('diagnostics')" :aria-pressed="diagnosticsOpen" @click="diagnosticsOpen = !diagnosticsOpen"><Info /></Button>
        <Button variant="secondary" size="icon" class="size-9 shrink-0 rounded-full bg-white/10 text-white hover:bg-white/20" :disabled="busy" :aria-label="status.window?.fullscreen ? t('exitFullscreen') : t('fullscreen')" :aria-pressed="status.window?.fullscreen" @click="windowAction('fullscreen')"><Minimize2 v-if="status.window?.fullscreen" /><Maximize2 v-else /></Button>
      </PlayerTransportControls>
    </footer>
    <Dialog v-model:open="prepareOpen">
      <DialogContent class="flex max-h-[calc(100vh-3rem)] flex-col overflow-y-auto sm:max-w-2xl">
        <DialogHeader><DialogTitle>{{ t('title') }}</DialogTitle><DialogDescription>{{ t('subtitle') }}</DialogDescription></DialogHeader>
        <Alert v-if="!bridge" variant="destructive"><AlertTitle>{{ t('unavailable') }}</AlertTitle></Alert>
        <Alert v-if="error || status.state.error" variant="destructive"><AlertTitle>{{ error || t('error') }}</AlertTitle><AlertDescription v-if="status.state.error">{{ status.state.error }}</AlertDescription></Alert>
    <Card>
      <CardHeader><CardTitle>{{ t('prepare') }}</CardTitle><CardDescription>{{ status.engineReady ? t('ready') : t('missing') }}</CardDescription></CardHeader>
      <CardContent>
        <FieldGroup>
          <Field><FieldLabel for="server">{{ t('server') }}</FieldLabel>
            <div class="flex flex-wrap gap-2"><Input id="server" v-model="origin" class="min-w-0 flex-1" :disabled="busy" @keydown.enter="connect" /><Button variant="outline" :disabled="!bridge || busy" @click="connect">{{ t('connect') }}</Button></div>
          </Field>
          <Field v-if="status.connection && !unlocked"><FieldLabel for="pin">{{ t('pin') }}</FieldLabel>
            <div class="flex gap-2"><Input id="pin" v-model="pin" type="password" inputmode="numeric" autocomplete="off" :disabled="busy" @keydown.enter="unlock" /><Button :disabled="busy || !pin" @click="unlock">{{ t('unlock') }}</Button></div>
          </Field>
        </FieldGroup>
      </CardContent>
      <CardFooter class="flex flex-wrap items-center justify-between gap-2">
        <Button variant="outline" :disabled="!bridge || busy" @click="selectExecutable">{{ t('engine') }}</Button>
        <span v-if="status.connection" class="text-sm text-muted-foreground">{{ unlocked ? t('connected') : t('locked') }}</span>
      </CardFooter>
    </Card>
    <Card>
      <CardHeader><CardTitle>{{ t('media') }}</CardTitle><CardDescription v-if="!unlocked">{{ t('lockedHint') }}</CardDescription></CardHeader>
      <CardContent>
        <FieldGroup>
          <Field><FieldLabel for="query">{{ t('query') }}</FieldLabel><div class="flex gap-2"><Input id="query" v-model="query" :disabled="!unlocked || busy" @keydown.enter="search" /><Button variant="outline" :disabled="!unlocked || busy" @click="search">{{ t('search') }}</Button></div></Field>
          <Field v-if="movies.length"><FieldLabel for="movie">{{ t('movie') }}</FieldLabel>
            <Select :model-value="movie?.id" :disabled="busy" @update:model-value="chooseMovie"><SelectTrigger id="movie"><SelectValue :placeholder="t('selectMovie')" /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="item in movies" :key="item.id" :value="item.id">{{ item.code }} · {{ item.title }}</SelectItem></SelectGroup></SelectContent></Select>
          </Field>
          <p v-else-if="searched" class="text-sm text-muted-foreground">{{ t('noMovies') }}</p>
          <Field v-if="movie?.files.length"><FieldLabel for="file">{{ t('file') }}</FieldLabel>
            <Select :model-value="fileId" :disabled="busy" @update:model-value="chooseFile"><SelectTrigger id="file"><SelectValue :placeholder="t('selectFile')" /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="file in movie.files" :key="file.id" :value="file.id">{{ file.fileName }}</SelectItem></SelectGroup></SelectContent></Select>
          </Field>
          <Field><FieldLabel for="start">{{ t('start') }}</FieldLabel><Input id="start" v-model.number="startSec" type="number" min="0" :disabled="!movie || busy" /></Field>
        </FieldGroup>
      </CardContent>
      <CardFooter class="flex flex-wrap items-center justify-between gap-2"><span class="text-sm text-muted-foreground">{{ status.engineReady ? t('nativeWindow') : t('engineRequired') }}</span><Button :disabled="!status.engineReady || !unlocked || !movie || busy" @click="play"><Play data-icon="inline-start" />{{ t('play') }}</Button></CardFooter>
    </Card>
      </DialogContent>
    </Dialog>
  </main>
</template>
