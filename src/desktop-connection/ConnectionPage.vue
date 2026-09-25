<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowRight, CircleAlert, History, LoaderCircle, Moon, Network, RefreshCw, Server, Sun, Trash2 } from "lucide-vue-next"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useTheme } from "@/composables/use-theme"

interface Connection { url: string; name: string; serverId: string }
interface Discovered { serverId: string; name: string; version: string; urls: string[]; expiresAt: number }
interface ConnectionAPI {
  checkUpdate(): Promise<string>
  discover(): Promise<Discovered[]>
  list(): Promise<{ connections: Connection[]; lastUrl?: string; activeUrl?: string; suggestedUrl?: string; desktopVersion: string }>
  connect(url: string): Promise<{ ok: boolean; error?: string }>
  cancel(): Promise<void>
  forget(url: string): Promise<void>
}
const api = (window as unknown as { curatedConnection?: ConnectionAPI }).curatedConnection
const { t } = useI18n()
const { resolvedMode, setThemePreference } = useTheme()
const address = ref("")
const activeUrl = ref("")
const connections = ref<Connection[]>([])
const busy = ref(false)
const error = ref("")
const version = ref("")
const updateMessage = ref("")
const updating = ref(false)
const found = ref<Discovered[]>([])
const scanning = ref(false)
const discoveryError = ref("")
let expiry: ReturnType<typeof setTimeout> | undefined
let hintTimer: ReturnType<typeof setTimeout> | undefined
let hintAttempts = 0
let disposed = false

/** 更新结果留在客户端页脚，不干扰服务器连接表单。 */
async function checkUpdate() {
  if (!api || updating.value) return
  updating.value = true
  updateMessage.value = ""
  try { updateMessage.value = await api.checkUpdate() }
  catch (reason) { updateMessage.value = String(reason) }
  finally { updating.value = false }
}

/** 首次启动短暂等待本机安装提示，不覆盖用户输入。 */
async function waitForLocalHint() {
  if (!api || disposed || address.value || hintAttempts++ >= 15) return
  try {
    const state = await api.list()
    if (state.suggestedUrl && !address.value) address.value = state.suggestedUrl
  } catch { /* 主表单承载连接错误，本机候选提示可缺省。 */ }
  if (!disposed && !address.value) hintTimer = setTimeout(waitForLocalHint, 1000)
}

/** 移除过期发现结果，保留最近连接记录。 */
function expireDiscovered() {
  found.value = found.value.filter(item => item.expiresAt > Date.now())
  if (found.value.length) {
    expiry = setTimeout(expireDiscovered, Math.max(1, Math.min(...found.value.map(item => item.expiresAt)) - Date.now()))
  }
}

/** 搜索失败在发现区展示，不把 HTTP 手动连接标为失败。 */
async function discover() {
  if (!api || scanning.value) return
  scanning.value = true
  discoveryError.value = ""
  found.value = []
  clearTimeout(expiry)
  try { found.value = await api.discover(); expireDiscovered() }
  catch (reason) { discoveryError.value = String(reason) }
  finally { scanning.value = false }
}

/** 同步主进程中已成功连接的档案与当前目标。 */
async function refresh() {
  if (!api) throw new Error(t("unavailable"))
  const state = await api.list()
  connections.value = state.connections
  activeUrl.value = state.activeUrl ?? ""
  version.value = state.desktopVersion
  return state
}

/** 连接中保留地址和取消入口，失败不清空表单。 */
async function connect(url = address.value) {
  if (!api || busy.value) return
  busy.value = true
  error.value = ""
  address.value = url
  try {
    const result = await api.connect(url)
    if (!result.ok) error.value = result.error ?? t("failed")
    else await refresh()
  } catch (reason) { error.value = String(reason) }
  finally { busy.value = false }
}

/** 取消请求的错误也回到当前表单，避免未处理 Promise。 */
async function cancel() {
  try { await api?.cancel() } catch (reason) { error.value = String(reason) }
}

/** 当前连接不可删除；忘记其它档案后刷新列表。 */
async function forget(url: string) {
  if (!api || url === activeUrl.value) return
  try { await api.forget(url); await refresh() } catch (reason) { error.value = String(reason) }
}

/** 只修改客户端本地主题，不修改远端 Server 偏好。 */
function toggleTheme() {
  setThemePreference(resolvedMode.value === "dark" ? "light" : "dark")
}

// 卸载时停止本地提示轮询与发现过期计时。
onUnmounted(() => { disposed = true; clearTimeout(expiry); clearTimeout(hintTimer) })
// 首屏加载记录与发现；已存目标保持自动重连行为。
onMounted(async () => {
  try {
    const state = await refresh()
    address.value = state.lastUrl ?? state.suggestedUrl ?? ""
    void discover()
    if (!address.value) void waitForLocalHint()
    if (state.lastUrl && !state.activeUrl) await connect(state.lastUrl)
  } catch (reason) { error.value = String(reason) }
})
</script>

<template>
  <main class="h-dvh overflow-y-auto overscroll-contain bg-background text-foreground">
    <div class="mx-auto flex w-full max-w-xl flex-col gap-4 px-5 py-6 sm:px-6">
      <div class="flex items-center justify-between gap-3 pb-2">
        <span class="desktop-wordmark text-xl font-semibold tracking-wide text-primary">Curated Desktop</span>
        <Button variant="ghost" size="icon" class="size-11 sm:size-8" :aria-label="resolvedMode === 'dark' ? t('light') : t('dark')" @click="toggleTheme">
          <Sun v-if="resolvedMode === 'dark'" aria-hidden="true" />
          <Moon v-else aria-hidden="true" />
        </Button>
      </div>
      <Card class="gap-4 border-0 py-5 shadow-none">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] grid-rows-1 items-center gap-x-2.5 px-5 pb-0">
          <span class="flex size-9 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary" aria-hidden="true"><Server class="size-4" /></span>
          <h1 class="min-w-0 text-lg font-semibold tracking-tight">{{ t('title') }}</h1>
        </CardHeader>
        <CardContent class="px-5 pt-0">
          <form class="flex flex-col gap-4" :aria-busy="busy" @submit.prevent="connect()">
            <FieldGroup>
              <Field class="gap-2">
                <FieldLabel for="server-address">{{ t('address') }}</FieldLabel>
                <Input id="server-address" v-model="address" class="min-h-11 sm:min-h-10" placeholder="http://192.168.1.20:8081" autocomplete="url" autocapitalize="off" :spellcheck="false" :disabled="busy || !api" :aria-describedby="error ? 'connection-error' : undefined" required />
              </Field>
            </FieldGroup>
            <Alert v-if="error" id="connection-error" variant="destructive" class="border-destructive/30 bg-destructive/10">
              <CircleAlert aria-hidden="true" /><AlertTitle>{{ t('failed') }}</AlertTitle>
              <AlertDescription class="break-words">{{ error }}</AlertDescription>
            </Alert>
            <div class="flex flex-wrap items-center justify-end gap-2">
              <Button v-if="busy" type="button" variant="outline" class="min-h-11 rounded-full sm:min-h-9" @click="cancel">{{ t('cancel') }}</Button>
              <Button type="submit" class="min-h-11 rounded-full px-5 sm:min-h-9" :disabled="busy || !api || !address.trim()">
                <LoaderCircle v-if="busy" class="motion-safe:animate-spin" aria-hidden="true" />
                {{ busy ? t('connecting') : t('connect') }}
                <ArrowRight v-if="!busy" aria-hidden="true" />
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <section v-if="connections.length" aria-labelledby="recent-title">
        <Card class="gap-3 border-border py-4 shadow-sm">
          <CardHeader class="flex flex-row items-center gap-2 px-5 pb-0">
            <History class="size-4 text-muted-foreground" aria-hidden="true" />
            <h2 id="recent-title" class="text-sm font-semibold">{{ t('recent') }}</h2>
          </CardHeader>
          <CardContent class="px-5 pt-0">
            <ul class="divide-y divide-border/60">
              <li v-for="item in connections" :key="item.url" class="flex min-w-0 items-center gap-2 py-3 first:pt-0 last:pb-0">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2"><p class="truncate text-sm font-medium">{{ item.name }}</p><Badge v-if="item.url === activeUrl" variant="secondary">{{ t('current') }}</Badge></div>
                  <p class="mt-1 truncate text-xs text-muted-foreground" :title="item.url">{{ item.url }}</p>
                </div>
                <Button variant="outline" size="sm" class="min-h-11 rounded-full sm:min-h-8" :disabled="busy || item.url === activeUrl" @click="connect(item.url)">{{ t('connect') }}</Button>
                <Button variant="ghost" size="icon-sm" class="size-11 text-muted-foreground sm:size-8" :disabled="busy || item.url === activeUrl" :aria-label="t('forget', { name: item.name })" @click="forget(item.url)"><Trash2 aria-hidden="true" /></Button>
              </li>
            </ul>
          </CardContent>
        </Card>
      </section>

      <section aria-labelledby="discovery-title">
        <Card class="gap-3 border-0 py-4 shadow-none">
          <CardHeader class="flex flex-row items-center justify-between gap-3 px-5 pb-0">
            <div class="flex min-w-0 items-center gap-2"><Network class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" /><h2 id="discovery-title" class="text-sm font-semibold">{{ t('discovery') }}</h2></div>
            <Button variant="ghost" size="sm" class="min-h-11 rounded-full text-muted-foreground sm:min-h-8" :disabled="scanning || !api" @click="discover"><RefreshCw :class="{ 'motion-safe:animate-spin': scanning }" aria-hidden="true" />{{ t('refresh') }}</Button>
          </CardHeader>
          <CardContent class="px-5 pt-0">
            <div v-if="scanning" role="status" class="flex min-h-28 items-center justify-center gap-2 text-sm text-muted-foreground"><LoaderCircle class="size-4 motion-safe:animate-spin" aria-hidden="true" />{{ t('searching') }}</div>
            <Alert v-else-if="discoveryError" variant="destructive" class="border-destructive/30 bg-destructive/10"><CircleAlert aria-hidden="true" /><AlertTitle>{{ t('discoveryFailed') }}</AlertTitle><AlertDescription class="break-words">{{ discoveryError }}</AlertDescription></Alert>
            <Empty v-else-if="!found.length" role="status" class="gap-3 rounded-lg bg-muted/20 px-4 py-5 md:p-5">
              <EmptyHeader><EmptyMedia variant="icon" class="mb-0 size-9"><Network class="size-4" aria-hidden="true" /></EmptyMedia><EmptyTitle class="text-sm">{{ t('empty') }}</EmptyTitle><EmptyDescription class="text-xs">{{ t('emptyHint') }}</EmptyDescription></EmptyHeader>
            </Empty>
            <ul v-else class="divide-y divide-border/60">
              <li v-for="item in found" :key="item.serverId" class="space-y-2 py-3 first:pt-0 last:pb-0">
                <div class="flex items-center gap-2"><p class="min-w-0 truncate text-sm font-medium">{{ item.name }}</p><span class="shrink-0 text-xs text-muted-foreground">{{ item.version }}</span></div>
                <div v-for="url in item.urls" :key="url" class="flex items-center gap-3">
                  <span class="min-w-0 flex-1 truncate text-xs text-muted-foreground" :title="url">{{ url }}</span>
                  <Badge v-if="url === activeUrl" variant="secondary">{{ t('current') }}</Badge>
                  <Button v-else variant="outline" size="sm" class="min-h-11 rounded-full sm:min-h-8" :disabled="busy || item.expiresAt <= Date.now()" @click="connect(url)">{{ t('connect') }}</Button>
                </div>
              </li>
            </ul>
          </CardContent>
        </Card>
      </section>
      <div class="flex flex-col gap-2">
        <p v-if="updateMessage" role="status" class="max-h-16 overflow-y-auto break-words text-xs text-muted-foreground">{{ updateMessage }}</p>
        <div class="flex items-center justify-between gap-3">
          <span class="truncate text-xs text-muted-foreground">Desktop {{ version }}</span>
          <Button variant="ghost" size="sm" class="min-h-11 rounded-full text-xs text-muted-foreground sm:min-h-8" :disabled="updating || !api" @click="checkUpdate"><RefreshCw v-if="updating" class="motion-safe:animate-spin" aria-hidden="true" />{{ updating ? t('updating') : t('update') }}</Button>
        </div>
      </div>
    </div>
  </main>
</template>
