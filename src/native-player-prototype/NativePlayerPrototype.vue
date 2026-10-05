<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { Play, Pause, Square, MonitorPlay } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Select, SelectTrigger, SelectValue, SelectContent, SelectGroup, SelectItem } from "@/components/ui/select"
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
const seekSec = ref(0)
const speed = ref(1)
const volume = ref(100)
const busy = ref(false)
const error = ref("")
const searched = ref(false)
let interval: ReturnType<typeof setInterval> | undefined
let polling = false
const unlocked = computed(() => { /* Server 的解锁状态决定业务控件准入。 */ return status.value.connection?.unlocked === true })
const active = computed(() => { /* 控制按钮不对已结束或失败的进程继续发命令。 */ return ["playing", "paused"].includes(status.value.state.status) })

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
  await perform(() => { /* renderer 不发送原始媒体地址。 */
    return bridge!.start({ movieId: movie.value!.id, fileId: fileId.value || undefined, startSec: Number(startSec.value) })
  })
}

/** 控制当前正在播放的会话，与待播放文件选择分开。 */
async function control(action: NativeLabControl["action"], value?: number): Promise<void> {
  await perform(() => { /* 所有控制使用枚举及数值，无通用命令接口。 */ return bridge!.control({ action, value }) })
}

onMounted(() => {
  // 页面一进入先读取当前能力，轮询失败使用操作回执显示。
  void perform(async () => {
    // 重新加载控制页时显示实际连接来源，不把默认输入伪装成当前 Server。
    await refresh()
    if (status.value.connection) origin.value = status.value.connection.origin
  })
  interval = setInterval(() => { void refresh().catch(() => { /* 页面销毁竞争时不追加错误提示。 */ }) }, 500)
})
onBeforeUnmount(() => { /* 播放资源由主进程关闭窗口事件回收。 */ clearInterval(interval) })
</script>

<template>
  <main class="mx-auto flex h-full max-w-4xl flex-col gap-5 overflow-y-auto p-5 sm:p-8">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <MonitorPlay class="size-7 text-primary" aria-hidden="true" />
        <div><h1 class="text-xl font-semibold">{{ t('title') }}</h1><p class="text-sm text-muted-foreground">{{ t('subtitle') }}</p></div>
      </div>
      <Badge variant="secondary" aria-live="polite">{{ busy ? t('busy') : t(status.state.status) }}</Badge>
    </header>
    <Alert v-if="!bridge" variant="destructive"><AlertTitle>{{ t('unavailable') }}</AlertTitle></Alert>
    <Alert v-if="error || status.state.error" variant="destructive" role="alert">
      <AlertTitle>{{ error || t('error') }}</AlertTitle><AlertDescription v-if="status.state.error">{{ status.state.error }}</AlertDescription>
    </Alert>
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
    <Card>
      <CardHeader><CardTitle>{{ t('playback') }}</CardTitle><CardDescription>{{ t('time') }} · {{ status.state.positionSec.toFixed(1) }} / {{ status.state.durationSec.toFixed(1) }} s</CardDescription></CardHeader>
      <CardContent class="flex flex-col gap-4">
        <div class="flex flex-wrap gap-2"><Button variant="outline" :disabled="!active || busy" @click="control(status.state.status === 'paused' ? 'resume' : 'pause')"><Pause data-icon="inline-start" />{{ status.state.status === 'paused' ? t('resume') : t('pause') }}</Button><Button variant="outline" :disabled="!active || busy" @click="control('stop')"><Square data-icon="inline-start" />{{ t('stop') }}</Button></div>
        <FieldGroup class="grid gap-4 sm:grid-cols-3">
          <Field><FieldLabel for="seek">{{ t('seek') }}</FieldLabel><Input id="seek" v-model.number="seekSec" type="number" min="0" :disabled="!active || busy" /><Button variant="outline" :disabled="!active || busy" @click="control('seek', Number(seekSec))">{{ t('jump') }}</Button></Field>
          <Field><FieldLabel for="speed">{{ t('speed') }}</FieldLabel><Input id="speed" v-model.number="speed" type="number" min="0.25" max="4" step="0.25" :disabled="!active || busy" /><Button variant="outline" :disabled="!active || busy" @click="control('speed', Number(speed))">{{ t('apply') }}</Button></Field>
          <Field><FieldLabel for="volume">{{ t('volume') }}</FieldLabel><Input id="volume" v-model.number="volume" type="number" min="0" max="100" :disabled="!active || busy" /><Button variant="outline" :disabled="!active || busy" @click="control('volume', Number(volume))">{{ t('apply') }}</Button></Field>
        </FieldGroup>
      </CardContent>
      <CardFooter class="flex flex-wrap gap-4 text-sm text-muted-foreground" :aria-label="t('diagnostics')">
        <span>{{ t('codec') }}: {{ status.state.codec || '—' }}</span><span>{{ t('hwdec') }}: {{ status.state.hwdec || '—' }}</span><span>{{ t('dropped') }}: {{ status.state.droppedFrames }}</span><span>{{ t('decoderDropped') }}: {{ status.state.decoderDroppedFrames }}</span>
      </CardFooter>
    </Card>
    <Alert v-if="status.state.progressError"><AlertTitle>{{ t('saveFailed') }}</AlertTitle></Alert>
  </main>
</template>
