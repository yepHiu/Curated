<script setup lang="ts">
import {
  computed,
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  type Ref,
} from "vue"
import { watchDebounced } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"
import { HttpClientError } from "@/api/http-client"
import type {
  CuratedFrameExportFormat,
  HealthDTO,
  ProviderHealthDTO,
  ProxySettingsDTO,
} from "@/api/types"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { pushAppToast } from "@/composables/use-app-toast"
import {
  SETTINGS_SCROLL_EL_KEY,
  SETTINGS_SCROLL_ROOT_ID,
  useSettingsScrollPreserve,
} from "@/composables/use-settings-scroll-preserve"
import { useTheme } from "@/composables/use-theme"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import {
  CheckSquare,
  Database,
  FolderOpen,
  FolderPlus,
  ListChecks,
  RefreshCw,
  Sparkles,
  X,
} from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  getStoredDirectoryHandle,
  setStoredDirectoryHandle,
  supportsFileSystemAccess,
} from "@/lib/curated-frames/db"
import {
  formatAboutBackendVersion,
} from "@/lib/about-version"
import {
  getCuratedCaptureKeyCode,
  getCuratedFrameSaveMode,
  setCuratedFrameSaveMode,
} from "@/lib/curated-frames/settings-storage"
import { formatCuratedCaptureKeyLabel } from "@/lib/player-shortcuts"
import SettingsAboutSection from "@/components/jav-library/settings/SettingsAboutSection.vue"
import SettingsCuratedSection from "@/components/jav-library/settings/SettingsCuratedSection.vue"
import SettingsGeneralSection from "@/components/jav-library/settings/SettingsGeneralSection.vue"
import SettingsLibraryPathActions from "@/components/jav-library/settings/SettingsLibraryPathActions.vue"
import SettingsMaintenanceSection from "@/components/jav-library/settings/SettingsMaintenanceSection.vue"
import SettingsMetadataAutomationSection from "@/components/jav-library/settings/SettingsMetadataAutomationSection.vue"
import SettingsMetadataModeSection from "@/components/jav-library/settings/SettingsMetadataModeSection.vue"
import SettingsMetadataProviderChainSection from "@/components/jav-library/settings/SettingsMetadataProviderChainSection.vue"
import SettingsMetadataProviderSelectSection from "@/components/jav-library/settings/SettingsMetadataProviderSelectSection.vue"
import SettingsNetworkSection from "@/components/jav-library/settings/SettingsNetworkSection.vue"
import SettingsOrganizeSection from "@/components/jav-library/settings/SettingsOrganizeSection.vue"
import SettingsOverviewSection from "@/components/jav-library/settings/SettingsOverviewSection.vue"
import SettingsPlaybackSection from "@/components/jav-library/settings/SettingsPlaybackSection.vue"
import { useLibraryService } from "@/services/library-service"
import {
  SETTINGS_NAV_ITEMS,
  type SettingsSectionSlug,
  isSettingsSectionSlug,
} from "@/lib/settings-nav"
import {
  statusTextClass,
} from "@/lib/ui/status-tone"
import { cn } from "@/lib/utils"

/** 设置页内按钮、选择器触发器、侧栏 Tab 统一高度 32px（h-8） */
const SETTINGS_CONTROL_H32_CLASS =
  "[&_[data-slot=button]]:!h-8 [&_[data-slot=button]]:!min-h-8 [&_[data-slot=button]]:!max-h-8 [&_[data-slot=button][data-size=icon]]:!size-8 [&_[data-slot=button][data-size=icon-sm]]:!size-8 [&_[data-slot=button][data-size=icon-lg]]:!size-8 [&_[data-slot=select-trigger]]:!h-8 [&_[data-slot=select-trigger]]:!min-h-8 [&_[data-slot=select-trigger]]:!max-h-8 [&_[data-slot=select-trigger]]:!py-0 [&_[data-slot=tabs-trigger]]:!h-8 [&_[data-slot=tabs-trigger]]:!min-h-8 [&_[data-slot=tabs-trigger]]:!max-h-8 [&_[data-slot=tabs-trigger]]:!py-0"
/** 资料库路径行：铅笔 / 扫描 / 移除 三枚图标按钮，统一 36×36 圆角矩形（覆盖上方全局 icon size-8） */
/** 资料库路径行：除「更多 ⋮」外，其它图标按钮 36×36（更多菜单为 ghost 竖三点，不强制放大，见 `data-overflow-menu-trigger`） */
const SETTINGS_LIBRARY_PATH_ACTION_ICONS_CLASS =
  "[&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!size-9 [&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!min-h-9 [&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!min-w-9 [&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!max-h-9 [&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!max-w-9 [&_.library-path-toolbar_[data-slot=button][data-size=icon]:not([data-overflow-menu-trigger])]:!rounded-lg"

const { t, locale } = useI18n()
const { themePreference, setThemePreference } = useTheme()
type ProxyScheme = "http" | "socks5"
const DEFAULT_PROXY_HOST = "127.0.0.1"

function setThemeFromSelect(v: unknown) {
  if (v === "light" || v === "dark" || v === "system") {
    setThemePreference(v)
  }
}
const route = useRoute()
const router = useRouter()

const settingsScrollElRef = inject<Ref<HTMLElement | null>>(SETTINGS_SCROLL_EL_KEY, ref(null))
const settingsNavItems = SETTINGS_NAV_ITEMS
const activeSlug = ref<SettingsSectionSlug>("overview")
const renderedSettingsSlugs = ref<SettingsSectionSlug[]>(["overview"])

function shouldRenderSettingsSection(slug: SettingsSectionSlug): boolean {
  return renderedSettingsSlugs.value.includes(slug)
}

function bindSettingsScrollRoot(el: unknown) {
  if (!settingsScrollElRef) return
  if (el == null) {
    settingsScrollElRef.value = null
    return
  }
  settingsScrollElRef.value = el instanceof HTMLElement ? el : null
}

onBeforeUnmount(() => {
  if (settingsScrollElRef) settingsScrollElRef.value = null
})

function scrollSettingsRootToTop() {
  const el =
    settingsScrollElRef.value ??
    (document.getElementById(SETTINGS_SCROLL_ROOT_ID) as HTMLElement | null)
  if (el) el.scrollTop = 0
}

function resolveSettingsSlugFromRoute(): SettingsSectionSlug {
  const raw = route.query.section
  const s = typeof raw === "string" ? raw : Array.isArray(raw) ? raw[0] : undefined
  if (s === "libraryBehavior") {
    router.replace({ query: { ...route.query, section: "library" } }).catch(() => {})
    return "library"
  }
  if (s === "logging") {
    router.replace({ query: { ...route.query, section: "general" } }).catch(() => {})
    return "general"
  }
  if (s && isSettingsSectionSlug(s)) return s
  return "overview"
}

watch(activeSlug, (slug) => {
  if (!renderedSettingsSlugs.value.includes(slug)) {
    renderedSettingsSlugs.value = [...renderedSettingsSlugs.value, slug]
  }
  if (slug === "about") {
    void ensureAboutHealthLoaded()
  }
  void nextTick(() => scrollSettingsRootToTop())
  if (route.query.section !== slug) {
    router.replace({ query: { ...route.query, section: slug } }).catch(() => {})
  }
})

watch(
  () => route.query.section,
  (raw) => {
    const s = typeof raw === "string" ? raw : Array.isArray(raw) ? raw[0] : undefined
    if (!s) {
      if (activeSlug.value !== "overview") {
        activeSlug.value = "overview"
      }
      return
    }
    if (s === "libraryBehavior") {
      router.replace({ query: { ...route.query, section: "library" } }).catch(() => {})
      if (activeSlug.value !== "library") activeSlug.value = "library"
      return
    }
    if (s === "logging") {
      router.replace({ query: { ...route.query, section: "general" } }).catch(() => {})
      if (activeSlug.value !== "general") activeSlug.value = "general"
      return
    }
    if (!isSettingsSectionSlug(s)) return
    if (s !== activeSlug.value) {
      activeSlug.value = s
    }
  },
)

const libraryService = useLibraryService()
const scanTaskTracker = useScanTaskTracker()
const { withPreservedScroll, withSyncPreservedScroll } = useSettingsScrollPreserve()
/** Plain object services don't unwrap nested ComputedRefs in templates */
const libraryPathsList = computed(() => libraryService.libraryPaths.value)

const addPathDialogOpen = ref(false)
const removePathDialogOpen = ref(false)
const removePathPending = ref<{ id: string; title: string; path: string } | null>(null)
const removePathBusy = ref(false)
const curatedCaptureShortcutLabel = computed(() =>
  formatCuratedCaptureKeyLabel(getCuratedCaptureKeyCode()),
)
const newPath = ref("")
const newPathTitle = ref("")
const addBusy = ref(false)
const scanPathBusy = ref<string | null>(null)
const revealPathBusy = ref<string | null>(null)
const fullScanBusy = ref(false)
const pathAddError = ref("")
const directoryHint = ref("")
const pickDirectoryBusy = ref(false)
const editingLibraryPathId = ref<string | null>(null)
const editLibraryTitleDraft = ref("")
const editTitleBusy = ref(false)
const editTitleError = ref("")
const scanFeedbackError = ref("")
/** 按目录批量元数据刷新：成功摘要 */
const metadataRefreshSuccess = ref("")
/** 按目录批量元数据刷新：错误文案 */
const metadataRefreshError = ref("")
const metadataRefreshBusy = ref(false)
/** 选中的库根路径（与后端配置的 path 字符串一致，用于 POST metadata-scrape） */
const selectedMetadataRefreshPaths = ref<string[]>([])
/** 资料库路径：与资料库页一致的「批量管理」模式（显示勾选与批量操作条） */
const libraryPathsBatchMode = ref(false)
/** 后台保存中：仅作轻提示，不禁用开关以免打断动画、体感卡顿 */
const organizeLibrarySaving = ref(false)
const organizeLibraryError = ref("")
const autoLibraryWatchSaving = ref(false)
const autoLibraryWatchError = ref("")
const autoActorProfileScrapeSaving = ref(false)
const autoActorProfileScrapeError = ref("")
const launchAtLoginSaving = ref(false)
const launchAtLoginError = ref("")
const curatedExportFormatSaving = ref(false)
const curatedExportFormatError = ref("")
const metadataMovieSaving = ref(false)
const metadataMovieError = ref("")

const proxyEnabledDraft = ref(false)
const proxySchemeDraft = ref<ProxyScheme>("http")
const proxyHostDraft = ref(DEFAULT_PROXY_HOST)
const proxyPortDraft = ref("")
const proxyUsernameDraft = ref("")
const proxyPasswordDraft = ref("")
const proxySaving = ref(false)
const proxyError = ref("")
const proxyJavbusBusy = ref(false)
const proxyJavbusResult = ref("")
const proxyJavbusResultOk = ref<boolean | null>(null)
const proxyGoogleBusy = ref(false)
const proxyGoogleResult = ref("")
const proxyGoogleResultOk = ref<boolean | null>(null)
const proxyAutoVerifyBusy = ref(false)
let proxyDraftSyncDepth = 0
let proxyAutoVerifySeq = 0

const proxyOutboundPingBusy = computed(
  () => proxyJavbusBusy.value || proxyGoogleBusy.value,
)

/** 代理用户名/密码折叠；有已保存认证信息时默认展开 */
const proxyAuthExpanded = ref(false)

function runWithoutProxyDraftSideEffects<T>(fn: () => T): T {
  proxyDraftSyncDepth += 1
  try {
    return fn()
  } finally {
    proxyDraftSyncDepth = Math.max(0, proxyDraftSyncDepth - 1)
  }
}

function syncProxyDraftFromService() {
  runWithoutProxyDraftSideEffects(() => {
    const p = libraryService.proxy.value
    const parsed = parseProxyUrlDraftParts(p.url)
    proxyEnabledDraft.value = Boolean(p.enabled)
    proxySchemeDraft.value = parsed.scheme
    proxyHostDraft.value = parsed.host
    proxyPortDraft.value = parsed.port
    proxyUsernameDraft.value = (p.username ?? parsed.username ?? "").trim()
    proxyPasswordDraft.value = p.password ?? parsed.password ?? ""
    const hasAuth =
      !!(p.username?.trim() || (p.password ?? "").length > 0)
    if (hasAuth) {
      proxyAuthExpanded.value = true
    }
  })
}

function parseProxyUrlDraftParts(rawUrl?: string | null): {
  scheme: ProxyScheme
  host: string
  port: string
  username?: string
  password?: string
} {
  const fallback = {
    scheme: "http" as ProxyScheme,
    host: DEFAULT_PROXY_HOST,
    port: "",
  }
  const trimmed = rawUrl?.trim()
  if (!trimmed) return fallback

  try {
    const parsed = new URL(trimmed)
    const protocol = parsed.protocol.replace(/:$/, "").toLowerCase()
    const scheme: ProxyScheme = protocol.startsWith("socks5") ? "socks5" : "http"
    return {
      scheme,
      host: parsed.hostname || DEFAULT_PROXY_HOST,
      port: parsed.port || "",
      username: parsed.username ? decodeURIComponent(parsed.username) : undefined,
      password: parsed.password ? decodeURIComponent(parsed.password) : undefined,
    }
  } catch {
    const match = /^([a-z0-9+.-]+):\/\/([^:/?#]+)(?::(\d+))?/i.exec(trimmed)
    if (!match) return fallback
    return {
      scheme: match[1].toLowerCase().startsWith("socks5") ? "socks5" : "http",
      host: match[2]?.trim() || DEFAULT_PROXY_HOST,
      port: match[3]?.trim() || "",
    }
  }
}

function normalizeProxyHostDraft(): string {
  return proxyHostDraft.value.trim() || DEFAULT_PROXY_HOST
}

function buildProxySettingsFromDraft(): ProxySettingsDTO | null {
  proxyError.value = ""

  if (!proxyEnabledDraft.value) {
    return { enabled: false }
  }

  const host = normalizeProxyHostDraft()
  if (!host) {
    proxyError.value = t("settings.proxyHostRequired")
    return null
  }

  const portRaw = proxyPortDraft.value.trim()
  if (!portRaw) {
    proxyError.value = t("settings.proxyPortRequired")
    return null
  }
  if (!/^\d+$/.test(portRaw)) {
    proxyError.value = t("settings.proxyPortInvalid")
    return null
  }

  const port = Number.parseInt(portRaw, 10)
  if (!Number.isFinite(port) || port <= 0 || port > 65535) {
    proxyError.value = t("settings.proxyPortInvalid")
    return null
  }

  return {
    enabled: true,
    url: `${proxySchemeDraft.value}://${host}:${port}`,
    username: proxyUsernameDraft.value.trim() || undefined,
    password: proxyPasswordDraft.value || undefined,
  }
}

function clearProxyTestResults() {
  proxyJavbusResult.value = ""
  proxyJavbusResultOk.value = null
  proxyGoogleResult.value = ""
  proxyGoogleResultOk.value = null
}

watch(
  () => [
    proxyEnabledDraft.value,
    proxySchemeDraft.value,
    proxyHostDraft.value,
    proxyPortDraft.value,
    proxyUsernameDraft.value,
    proxyPasswordDraft.value,
  ] as const,
  () => {
    if (proxyDraftSyncDepth > 0) return
    clearProxyTestResults()
  },
)

const proxyStatusMessage = computed(() => {
  if (proxyJavbusResult.value && proxyJavbusResultOk.value !== null) {
    return {
      text: proxyJavbusResult.value,
      className: proxyJavbusResultOk.value ? statusTextClass("success") : statusTextClass("danger"),
    }
  }
  if (proxyGoogleResult.value && proxyGoogleResultOk.value !== null) {
    return {
      text: proxyGoogleResult.value,
      className: proxyGoogleResultOk.value ? statusTextClass("success") : statusTextClass("danger"),
    }
  }
  if (useWebApi && proxyAutoVerifyBusy.value) {
    return {
      text: t("settings.proxyPingGoogleTesting"),
      className: "text-xs text-muted-foreground motion-safe:animate-pulse",
    }
  }
  if (useWebApi && proxySaving.value) {
    return {
      text: t("settings.proxySyncing"),
      className: "text-xs text-muted-foreground motion-safe:animate-pulse",
    }
  }
  if (proxyError.value) {
    return {
      text: proxyError.value,
      className: cn("text-sm", statusTextClass("danger")),
    }
  }
  return null
})

function applySavedProxyGoogleResult(
  seq: number,
  result: { ok: boolean; latencyMs?: number; httpStatus?: number; message?: string },
) {
  if (seq !== proxyAutoVerifySeq) return
  if (result.ok) {
    proxyGoogleResultOk.value = true
    proxyGoogleResult.value = t("settings.proxyPingGoogleOk", {
      ms: result.latencyMs,
      status: result.httpStatus ?? "—",
    })
    pushAppToast(
      t("settings.proxySaveToastVerified", {
        ms: result.latencyMs,
        status: result.httpStatus ?? "—",
      }),
      {
        variant: "success",
        durationMs: 4200,
      },
    )
    return
  }

  const detail = result.message?.trim() || t("settings.proxyPingJavbusFailUnknown")
  proxyGoogleResultOk.value = false
  proxyGoogleResult.value = t("settings.proxyPingJavbusFail", { message: detail })
  pushAppToast(t("settings.proxySaveToastUnverified", { message: detail }), {
    variant: "warning",
    durationMs: 5200,
  })
}

async function verifySavedProxyInBackground(seq: number) {
  try {
    const verifyResult = await libraryService.pingProxyGoogle()
    applySavedProxyGoogleResult(seq, verifyResult)
  } catch (err) {
    const detail =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message
          ? err.message
          : t("settings.proxyPingJavbusFailUnknown")
    applySavedProxyGoogleResult(seq, { ok: false, message: detail })
  } finally {
    if (seq === proxyAutoVerifySeq) {
      proxyAutoVerifyBusy.value = false
    }
  }
}

/** 日志保留天数：0=模块默认约 7 天；另含 1/5/10/30 预设；其它已保存值会多出一项「当前 n 天」 */
async function saveProxySettings() {
  const body = buildProxySettingsFromDraft()
  if (!body) return
  try {
    await withPreservedScroll(async () => {
      proxySaving.value = true
      try {
        await libraryService.setProxy(body)
        syncProxyDraftFromService()
      } finally {
        proxySaving.value = false
      }
    })
    if (!useWebApi) {
      pushAppToast(t("settings.proxySaveToastMock"), {
        variant: "success",
        durationMs: 3200,
      })
      return
    }

    if (!body.enabled) {
      pushAppToast(t("settings.proxySaveToastDisabled"), {
        variant: "success",
        durationMs: 3200,
      })
      return
    }

    proxyAutoVerifySeq += 1
    const verifySeq = proxyAutoVerifySeq
    proxyAutoVerifyBusy.value = true
    proxyGoogleResult.value = ""
    proxyGoogleResultOk.value = null
    pushAppToast(t("settings.proxySaveToastSaved"), {
      variant: "success",
      durationMs: 2400,
    })
    void verifySavedProxyInBackground(verifySeq)
  } catch (err) {
    console.error("[settings] save proxy failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      proxyError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      proxyError.value = err.message
    } else {
      proxyError.value = t("settings.errSaveTitle")
    }
    pushAppToast(proxyError.value, {
      variant: "destructive",
      durationMs: 4200,
    })
  }
}

const settingsAutoSaveReady = ref(false)
/*
async function performSavePlaybackSettings() {
  playbackError.value = ""
  const shouldPatchServer = !playbackDraftMatchesServer()
  const shouldPersistBrowserTemplate = !playbackBrowserTemplateMatchesPersisted()
  let patch: PatchPlayerSettingsBody | null = null
  if (shouldPatchServer) {
    patch = buildPlaybackPatchFromDraft()
    if (!patch) {
      return
    }
  }
  try {
    // 播放项自动保存极高频；不再包 withPreservedScroll，避免 finally 里连续滚动回写与 Vue patch 冲突导致白屏。
    playbackSaving.value = true
    try {
      if (shouldPatchServer && patch) {
        await libraryService.patchPlayerSettings(patch)
        await nextTick()
        syncPlaybackDraftFromService()
      }
      if (shouldPersistBrowserTemplate) {
        playbackNativePlayerProtocolTemplateDraft.value = persistNativePlayerBrowserTemplate(
          playbackNativePlayerPresetDraft.value,
          playbackNativePlayerProtocolTemplateDraft.value,
        )
      }
      flashPlaybackSaved()
    } finally {
      playbackSaving.value = false
    }
  } catch (err) {
    console.error("[settings] save playback settings failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      playbackError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      playbackError.value = err.message
    } else {
      playbackError.value = t("settings.errSaveTitle")
    }
  }
}

async function savePlaybackSettings() {
  if (playbackSavePromise) {
    playbackSaveQueued = true
    return playbackSavePromise
  }

  playbackSavePromise = (async () => {
    do {
      playbackSaveQueued = false
      await performSavePlaybackSettings()
    } while (playbackSaveQueued)
  })()

  try {
    await playbackSavePromise
  } finally {
    playbackSavePromise = null
  }
}

function backendLogDraftMatchesServer(): boolean {
  const b = libraryService.backendLog.value
  if (backendLogDirDraft.value.trim() !== (b.logDir ?? "").trim()) {
    return false
  }
  const lvl = (backendLogLevelDraft.value || "info").trim() || "info"
  const serverLvl = ((b.logLevel ?? "info").trim() || "info") as string
  if (lvl !== serverLvl) {
    return false
  }
  const cur = Number.parseInt(backendLogMaxAgeDaysChoice.value, 10)
  const s = b.logMaxAgeDays
  if (s === undefined || s <= 0) {
    return backendLogMaxAgeDaysChoice.value === "0"
  }
  return Number.isFinite(cur) && cur === s
}

watchDebounced(
  () =>
    [
      backendLogDirDraft.value,
      backendLogMaxAgeDaysChoice.value,
      backendLogLevelDraft.value,
    ] as const,
  async () => {
    if (!settingsAutoSaveReady.value || !useWebApi) {
      return
    }
    if (backendLogDraftMatchesServer()) {
      return
    }
    await saveBackendLogSettings()
  },
  { debounce: 550, maxWait: 5000 },
)

watchDebounced(
  () =>
    [
      playbackHardwareDecodeDraft.value,
      playbackHardwareEncoderDraft.value,
      playbackNativePlayerPresetDraft.value,
      playbackNativePlayerEnabledDraft.value,
      playbackNativePlayerProtocolTemplateDraft.value,
      playbackStreamPushEnabledDraft.value,
      playbackForceStreamPushDraft.value,
      playbackFfmpegCommandDraft.value,
      playbackPreferNativePlayerDraft.value,
      playbackSeekForwardStepDraft.value,
      playbackSeekBackwardStepDraft.value,
    ] as const,
  async () => {
    try {
      if (!settingsAutoSaveReady.value) {
        return
      }
      if (playbackDraftMatchesServer() && playbackBrowserTemplateMatchesPersisted()) {
        return
      }
      await savePlaybackSettings()
    } catch (err) {
      console.error("[settings] playback autosave failed", err)
    }
  },
  { debounce: 550, maxWait: 5000 },
)
*/

async function testProxyJavbus() {
  const draft = buildProxySettingsFromDraft()
  if (!draft) return
  proxyJavbusResult.value = ""
  proxyJavbusResultOk.value = null
  if (!useWebApi) {
    return
  }
  await withPreservedScroll(async () => {
    proxyJavbusBusy.value = true
      try {
        const body = {
          proxy: draft,
        }
        const res = await libraryService.pingProxyJavbus(body)
        if (res.ok) {
          proxyJavbusResultOk.value = true
          proxyJavbusResult.value = t("settings.proxyPingJavbusOk", {
            ms: res.latencyMs,
            status: res.httpStatus ?? "—",
          })
        } else {
          proxyJavbusResultOk.value = false
          const detail = res.message?.trim() || t("settings.proxyPingJavbusFailUnknown")
          proxyJavbusResult.value = t("settings.proxyPingJavbusFail", { message: detail })
        }
      } catch (err) {
        proxyJavbusResultOk.value = false
        if (err instanceof HttpClientError && err.apiError?.message) {
          proxyJavbusResult.value = t("settings.proxyPingJavbusFail", {
            message: err.apiError.message,
          })
        } else if (err instanceof Error) {
          proxyJavbusResult.value = t("settings.proxyPingJavbusFail", { message: err.message })
        } else {
          proxyJavbusResult.value = t("settings.proxyPingJavbusFail", {
            message: t("settings.proxyPingJavbusFailUnknown"),
          })
        }
      } finally {
        proxyJavbusBusy.value = false
      }
    })
}

async function testProxyGoogle() {
  const draft = buildProxySettingsFromDraft()
  if (!draft) return
  proxyGoogleResult.value = ""
  proxyGoogleResultOk.value = null
  if (!useWebApi) {
    return
  }
  await withPreservedScroll(async () => {
    proxyGoogleBusy.value = true
      try {
        const body = {
          proxy: draft,
        }
        const res = await libraryService.pingProxyGoogle(body)
        if (res.ok) {
          proxyGoogleResultOk.value = true
          proxyGoogleResult.value = t("settings.proxyPingGoogleOk", {
            ms: res.latencyMs,
            status: res.httpStatus ?? "—",
          })
        } else {
          proxyGoogleResultOk.value = false
          const detail = res.message?.trim() || t("settings.proxyPingJavbusFailUnknown")
          proxyGoogleResult.value = t("settings.proxyPingJavbusFail", { message: detail })
        }
      } catch (err) {
        proxyGoogleResultOk.value = false
        if (err instanceof HttpClientError && err.apiError?.message) {
          proxyGoogleResult.value = t("settings.proxyPingJavbusFail", {
            message: err.apiError.message,
          })
        } else if (err instanceof Error) {
          proxyGoogleResult.value = t("settings.proxyPingJavbusFail", { message: err.message })
        } else {
          proxyGoogleResult.value = t("settings.proxyPingJavbusFail", {
            message: t("settings.proxyPingJavbusFailUnknown"),
          })
        }
      } finally {
        proxyGoogleBusy.value = false
      }
    })
}

const organizeLibrary = computed(() => libraryService.organizeLibrary.value)
const autoLibraryWatch = computed(() => libraryService.autoLibraryWatch.value)
const autoActorProfileScrape = computed(() => libraryService.autoActorProfileScrape.value)
const launchAtLogin = computed(() => libraryService.launchAtLogin.value)
const launchAtLoginSupported = computed(() => libraryService.launchAtLoginSupported.value)
const curatedFrameExportFormat = computed(() => libraryService.curatedFrameExportFormat.value)
const curatedExportFormatOptions = computed(
  (): { value: CuratedFrameExportFormat; label: string }[] => [
    { value: "jpg", label: "JPG" },
    { value: "webp", label: "WebP" },
    { value: "png", label: "PNG" },
  ],
)
const curatedDirectorySupported = computed(() => supportsFileSystemAccess())

const metadataMovieProvider = computed(() => libraryService.metadataMovieProvider.value.trim())
const metadataMovieProviders = computed(() => [...libraryService.metadataMovieProviders.value])
const metadataMovieOrphan = computed(() => {
  const cur = metadataMovieProvider.value
  if (!cur) return ""
  return metadataMovieProviders.value.some((p) => p.toLowerCase() === cur.toLowerCase()) ? "" : cur
})
const metadataMovieSelectOptions = computed(() => {
  const list = metadataMovieProviders.value
  const o = metadataMovieOrphan.value
  if (o) {
    const rest = list.filter((p) => p.toLowerCase() !== o.toLowerCase())
    return [o, ...rest]
  }
  return list
})
const canPickSpecifiedMetadata = computed(() => metadataMovieSelectOptions.value.length > 0)
/** 有站点列表，或已有保存的链时，允许进入「多源链式」并展示链管理（避免仅有链配置却无列表时整块 UI 被隐藏） */
const canUseMetadataChainMode = computed(
  () => canPickSpecifiedMetadata.value || libraryService.metadataMovieProviderChain.value.length > 0,
)
const metadataMovieMode = computed(() => libraryService.metadataMovieScrapeMode.value)

/**
 * 刮削策略在界面上的选中态：须与「用户当前点的单选项」一致。
 * 仅用服务端推导的 metadataMovieMode 时，在「已指定站点 + 链尚未保存」场景下会一直是 specified，
 * 导致多源链面板永远不出现。
 */
const metadataMovieModeUi = ref<"auto" | "specified" | "chain">(metadataMovieMode.value)

function syncMetadataMovieModeUiFromServer() {
  metadataMovieModeUi.value = metadataMovieMode.value
}

// Provider Chain management
const providerChainDraft = ref<string[]>([])
const availableProvidersForChain = computed(() => {
  const all = metadataMovieProviders.value
  const selected = new Set(providerChainDraft.value.map((p) => p.toLowerCase()))
  return all.filter((p) => !selected.has(p.toLowerCase()))
})
const selectedProviderToAdd = ref<string>("")
const metadataMovieChainSaving = ref(false)
const metadataMovieChainError = ref("")

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const viteMode = import.meta.env.MODE
const isViteDev = import.meta.env.DEV
const launchAtLoginDisabled = computed(
  () => !useWebApi || (!launchAtLoginSupported.value && !launchAtLogin.value),
)
const launchAtLoginUnavailableHint = computed(() => {
  if (!useWebApi) return t("settings.launchAtLoginMockHint")
  if (!launchAtLoginSupported.value) return t("settings.launchAtLoginUnsupportedHint")
  return ""
})
const aboutHealth = ref<HealthDTO | null>(null)
const aboutHealthLoading = ref(false)
const aboutHealthError = ref("")
const aboutHealthLoaded = ref(false)

const aboutBackendVersionDisplay = computed(() => {
  if (!useWebApi) {
    return t("settings.aboutVersionMock")
  }
  if (aboutHealthLoading.value) {
    return t("settings.aboutVersionLoading")
  }
  if (aboutHealthError.value) {
    return aboutHealthError.value
  }
  if (aboutHealth.value) {
    return formatAboutBackendVersion(aboutHealth.value)
  }
  return "—"
})

const aboutBackendVersionStatus = computed<"default" | "loading" | "error">(() => {
  if (aboutHealthError.value) {
    return "error"
  }
  if (aboutHealthLoading.value) {
    return "loading"
  }
  return "default"
})

async function loadAboutHealth() {
  if (!useWebApi) return
  aboutHealthLoading.value = true
  aboutHealthError.value = ""
  try {
    aboutHealth.value = await libraryService.health()
  } catch (err) {
    aboutHealth.value = null
    if (err instanceof HttpClientError && err.apiError?.message) {
      aboutHealthError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      aboutHealthError.value = err.message
    } else {
      aboutHealthError.value = t("settings.aboutVersionFetchFailed")
    }
  } finally {
    aboutHealthLoading.value = false
  }
}

async function ensureAboutHealthLoaded() {
  if (!useWebApi || aboutHealthLoaded.value || aboutHealthLoading.value) {
    return
  }
  aboutHealthLoaded.value = true
  await loadAboutHealth()
}

const providerHealthByName = ref<Record<string, ProviderHealthDTO>>({})
const providerPingAllBusy = ref(false)
const providerPingOneName = ref<string | null>(null)
const providerHealthPingAllSummary = ref("")
const providerHealthPingError = ref("")

function healthForProvider(name: string): ProviderHealthDTO | undefined {
  const map = providerHealthByName.value
  if (map[name]) return map[name]
  const lower = name.toLowerCase()
  for (const [k, v] of Object.entries(map)) {
    if (k.toLowerCase() === lower) return v
  }
  return undefined
}

async function pingAllMetadataProviders() {
  if (!useWebApi) return
  providerHealthPingError.value = ""
  providerHealthPingAllSummary.value = ""
  try {
    await withPreservedScroll(async () => {
      providerPingAllBusy.value = true
      try {
        const res = await libraryService.pingAllProviders()
        const next: Record<string, ProviderHealthDTO> = { ...providerHealthByName.value }
        for (const p of res.providers) {
          next[p.name] = p
        }
        providerHealthByName.value = next
        providerHealthPingAllSummary.value = t("settings.providerHealthPingAllSummary", {
          total: res.total,
          ok: res.ok,
          fail: res.fail,
        })
      } finally {
        providerPingAllBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] ping all providers failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      providerHealthPingError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      providerHealthPingError.value = err.message
    } else {
      providerHealthPingError.value = t("settings.providerHealthPingError")
    }
  }
}

async function pingOneMetadataProvider(name: string) {
  if (!useWebApi || !name.trim()) return
  providerHealthPingError.value = ""
  try {
    await withPreservedScroll(async () => {
      providerPingOneName.value = name
      try {
        const dto = await libraryService.pingProvider(name.trim())
        providerHealthByName.value = { ...providerHealthByName.value, [dto.name]: dto }
      } finally {
        providerPingOneName.value = null
      }
    })
  } catch (err) {
    console.error("[settings] ping provider failed", name, err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      providerHealthPingError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      providerHealthPingError.value = err.message
    } else {
      providerHealthPingError.value = t("settings.providerHealthPingError")
    }
  }
}

function initProviderChainDraft() {
  providerChainDraft.value = [...libraryService.metadataMovieProviderChain.value]
}

const chainDragFromIndex = ref<number | null>(null)

function reorderProviderChainDrag(from: number, to: number) {
  if (from === to || from < 0 || to < 0) return
  const draft = [...providerChainDraft.value]
  if (from >= draft.length || to >= draft.length) return
  withSyncPreservedScroll(() => {
    const next = [...draft]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    providerChainDraft.value = next
  })
}

function onChainDragStart(e: DragEvent, index: number) {
  chainDragFromIndex.value = index
  e.dataTransfer?.setData("text/plain", String(index))
  e.dataTransfer!.effectAllowed = "move"
}

function onChainDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = "move"
}

function onChainDrop(toIndex: number) {
  const from = chainDragFromIndex.value
  if (from == null) return
  reorderProviderChainDrag(from, toIndex)
  chainDragFromIndex.value = null
}

function onChainDragEnd() {
  chainDragFromIndex.value = null
}

const triggerScrapeCardBusy = ref(false)
const triggerScrapeCardSuccess = ref("")
const triggerScrapeCardError = ref("")

async function runTriggerScrapeAllLibraryRoots() {
  triggerScrapeCardSuccess.value = ""
  triggerScrapeCardError.value = ""
  if (!useWebApi) {
    triggerScrapeCardError.value = t("settings.triggerScrapeMockHint")
    return
  }
  const paths = libraryPathsList.value.map((p) => p.path.trim()).filter(Boolean)
  if (paths.length === 0) {
    triggerScrapeCardError.value = t("settings.triggerScrapeNoPaths")
    return
  }
  try {
    await withPreservedScroll(async () => {
      triggerScrapeCardBusy.value = true
      try {
        const dto = await libraryService.refreshMetadataForLibraryPaths(paths)
        const parts: string[] = [t("settings.metadataQueued", { n: dto.queued })]
        if (dto.skipped > 0) {
          parts.push(t("settings.metadataSkipped", { n: dto.skipped }))
        }
        if (dto.invalidPaths.length > 0) {
          parts.push(t("settings.metadataInvalid", { paths: dto.invalidPaths.join("；") }))
        }
        triggerScrapeCardSuccess.value = parts.join(" ")
      } finally {
        triggerScrapeCardBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] trigger scrape all roots failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      triggerScrapeCardError.value = err.apiError.message
    } else {
      triggerScrapeCardError.value = t("settings.errMetadataBatch")
    }
  }
}

function removeProviderFromChain(index: number) {
  withSyncPreservedScroll(() => {
    providerChainDraft.value = providerChainDraft.value.filter((_, i) => i !== index)
  })
}

function addProviderToChain() {
  const name = selectedProviderToAdd.value.trim()
  if (!name) return
  if (providerChainDraft.value.some((p) => p.toLowerCase() === name.toLowerCase())) {
    return
  }
  withSyncPreservedScroll(() => {
    providerChainDraft.value = [...providerChainDraft.value, name]
    selectedProviderToAdd.value = ""
  })
}

function providerChainsEqual(a: readonly string[], b: readonly string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

async function saveProviderChain() {
  if (providerChainsEqual(providerChainDraft.value, libraryService.metadataMovieProviderChain.value)) {
    metadataMovieChainError.value = ""
    return
  }
  metadataMovieChainError.value = ""
  try {
    await withPreservedScroll(async () => {
      metadataMovieChainSaving.value = true
      try {
        await libraryService.setMetadataMovieProviderChain(providerChainDraft.value)
        syncMetadataMovieModeUiFromServer()
      } finally {
        metadataMovieChainSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] save provider chain failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataMovieChainError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      metadataMovieChainError.value = err.message
    } else {
      metadataMovieChainError.value = t("settings.errSaveTitle")
    }
  }
}

watchDebounced(
  providerChainDraft,
  async () => {
    if (metadataMovieModeUi.value !== "chain") return
    if (providerChainsEqual(providerChainDraft.value, libraryService.metadataMovieProviderChain.value)) {
      return
    }
    await saveProviderChain()
  },
  { deep: true, debounce: 450 },
)

async function onMetadataMovieModeChain() {
  if (metadataMovieModeUi.value === "chain") return
  metadataMovieChainError.value = ""
  metadataMovieError.value = ""
  try {
    await withPreservedScroll(async () => {
      await libraryService.setMetadataMovieScrapeMode("chain")
      metadataMovieModeUi.value = "chain"
      initProviderChainDraft()
      syncMetadataMovieModeUiFromServer()
    })
  } catch (err) {
    console.error("[settings] metadata scrape mode (chain) failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataMovieChainError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      metadataMovieChainError.value = err.message
    } else {
      metadataMovieChainError.value = t("settings.errSaveTitle")
    }
  }
}

const dashboardStats = computed(() => libraryService.libraryStats.value)

/** 萃取帧：保存策略 */
const curatedSaveMode = ref<CuratedFrameSaveMode>("app")
const curatedExportDirLabel = ref("")
const curatedExportPickBusy = ref(false)
const curatedExportError = ref("")

async function refreshCuratedExportDirLabel() {
  try {
    const h = await getStoredDirectoryHandle()
    curatedExportDirLabel.value = h?.name ?? ""
  } catch {
    curatedExportDirLabel.value = ""
  }
}

async function pickCuratedExportDirectory() {
  curatedExportError.value = ""
  if (!supportsFileSystemAccess()) {
    curatedExportError.value = t("settings.curatedExportNoApi")
    return
  }
  curatedExportPickBusy.value = true
  try {
    const handle = await window.showDirectoryPicker!({ mode: "readwrite" })
    await setStoredDirectoryHandle(handle)
    curatedExportDirLabel.value = handle.name
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      return
    }
    console.error("[settings] curated export directory pick failed", err)
    curatedExportError.value =
      err instanceof DOMException && err.name === "NotAllowedError"
        ? t("settings.curatedExportDenied")
        : t("settings.curatedExportFail")
  } finally {
    curatedExportPickBusy.value = false
  }
}

async function clearCuratedExportDirectory() {
  curatedExportError.value = ""
  try {
    await setStoredDirectoryHandle(null)
    curatedExportDirLabel.value = ""
  } catch (err) {
    console.error("[settings] clear curated export directory failed", err)
    curatedExportError.value = t("settings.curatedExportClearFail")
  }
}


const hasMetadataPathSelection = computed(() => selectedMetadataRefreshPaths.value.length > 0)

function isMetadataPathChecked(path: string) {
  return selectedMetadataRefreshPaths.value.includes(path)
}

function toggleMetadataPathSelection(path: string) {
  const cur = selectedMetadataRefreshPaths.value
  if (cur.includes(path)) {
    selectedMetadataRefreshPaths.value = cur.filter((p) => p !== path)
  } else {
    selectedMetadataRefreshPaths.value = [...cur, path]
  }
}

function selectAllMetadataPaths() {
  selectedMetadataRefreshPaths.value = libraryPathsList.value.map((p) => p.path)
}

function clearMetadataPathSelection() {
  selectedMetadataRefreshPaths.value = []
}

function enterLibraryPathsBatchMode() {
  libraryPathsBatchMode.value = true
}

function exitLibraryPathsBatchMode() {
  libraryPathsBatchMode.value = false
  clearMetadataPathSelection()
  metadataRefreshSuccess.value = ""
  metadataRefreshError.value = ""
}

const canSaveNewPath = computed(() => {
  const t = newPath.value.trim()
  return t.length > 0 && isAbsoluteLibraryPath(t)
})

/** 选文件夹后的说明与「保存」状态合并为一条，避免重复段落 */
const directoryHintDisplay = computed(() => {
  const h = directoryHint.value.trim()
  if (!h) return ""
  if (!canSaveNewPath.value) {
    return `${h}\n\n${t("settings.pickFolderHintSaveSuffix")}`
  }
  return h
})

onMounted(async () => {
  activeSlug.value = resolveSettingsSlugFromRoute()
  const rawInit = route.query.section
  const sInit =
    typeof rawInit === "string" ? rawInit : Array.isArray(rawInit) ? rawInit[0] : undefined
  if (
    sInit &&
    sInit !== "libraryBehavior" &&
    sInit !== "logging" &&
    !isSettingsSectionSlug(sInit)
  ) {
    router.replace({ query: { ...route.query, section: "overview" } }).catch(() => {})
  }
  await withPreservedScroll(() => libraryService.refreshSettings())
  syncMetadataMovieModeUiFromServer()
  initProviderChainDraft()
  syncProxyDraftFromService()
  let mode = getCuratedFrameSaveMode()
  if (mode === "directory" && !supportsFileSystemAccess()) {
    mode = "app"
    setCuratedFrameSaveMode(mode)
  }
  curatedSaveMode.value = mode
  void refreshCuratedExportDirLabel()
  await nextTick()
  settingsAutoSaveReady.value = true
})

watch(curatedSaveMode, (mode) => {
  setCuratedFrameSaveMode(mode)
})

watch(addPathDialogOpen, (open) => {
  if (!open) {
    newPath.value = ""
    newPathTitle.value = ""
    pathAddError.value = ""
    directoryHint.value = ""
  }
})

function clearPathAddError() {
  pathAddError.value = ""
}

function startEditLibraryTitle(path: { id: string; title: string }) {
  editingLibraryPathId.value = path.id
  editLibraryTitleDraft.value = path.title
  editTitleError.value = ""
}

function cancelEditLibraryTitle() {
  editingLibraryPathId.value = null
  editLibraryTitleDraft.value = ""
  editTitleError.value = ""
}

async function saveLibraryPathTitle(id: string) {
  editTitleError.value = ""
  try {
    await withPreservedScroll(async () => {
      editTitleBusy.value = true
      try {
        await libraryService.updateLibraryPathTitle(id, editLibraryTitleDraft.value)
        cancelEditLibraryTitle()
      } finally {
        editTitleBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] update library title failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      editTitleError.value = err.apiError.message
    } else {
      editTitleError.value = t("settings.errSaveTitle")
    }
  }
}

async function browseForDirectory() {
  directoryHint.value = ""
  pickDirectoryBusy.value = true
  try {
    const outcome = await pickLibraryDirectory()
    if (outcome.status === "ok") {
      newPath.value = outcome.path
      clearPathAddError()
      return
    }
    if (outcome.status === "hint") {
      directoryHint.value = outcome.message
      if (outcome.suggestedTitle && !newPathTitle.value.trim()) {
        newPathTitle.value = outcome.suggestedTitle
      }
      await nextTick()
      document.getElementById("new-lib-path")?.focus()
      return
    }
    if (outcome.status === "unsupported") {
      directoryHint.value = t("settings.errPickUnsupported")
    }
  } finally {
    pickDirectoryBusy.value = false
  }
}

async function submitAddPath() {
  pathAddError.value = ""
  const trimmed = newPath.value.trim()
  if (!isAbsoluteLibraryPath(trimmed)) {
    pathAddError.value = t("settings.errAbsoluteRequired")
    return
  }
  try {
    await withPreservedScroll(async () => {
      addBusy.value = true
      try {
        const scanTask = await libraryService.addLibraryPath(
          newPath.value,
          newPathTitle.value || undefined,
        )
        if (scanTask?.taskId) {
          scanTaskTracker.start(scanTask.taskId)
        }
        newPath.value = ""
        newPathTitle.value = ""
        addPathDialogOpen.value = false
      } finally {
        addBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] add library path failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      pathAddError.value = err.apiError.message
    }
  }
}

function openRemovePathConfirm(entry: { id: string; title: string; path: string }) {
  removePathPending.value = entry
  removePathDialogOpen.value = true
}

watch(removePathDialogOpen, (open) => {
  if (!open) {
    removePathPending.value = null
    removePathBusy.value = false
  }
})

async function confirmRemoveLibraryPath() {
  const pending = removePathPending.value
  if (!pending) {
    return
  }
  removePathBusy.value = true
  try {
    await withPreservedScroll(() => libraryService.removeLibraryPath(pending.id))
    if (editingLibraryPathId.value === pending.id) {
      editingLibraryPathId.value = null
    }
    removePathDialogOpen.value = false
  } catch (err) {
    console.error("[settings] remove library path failed", err)
    const message =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message.trim()
          ? err.message
          : t("settings.removePathFailed")
    pushAppToast(message, { variant: "destructive" })
  } finally {
    removePathBusy.value = false
  }
}

async function rescanPath(path: string) {
  scanFeedbackError.value = ""
  try {
    await withPreservedScroll(async () => {
      scanPathBusy.value = path
      try {
        const task = await libraryService.scanLibraryPaths([path])
        if (task?.taskId) {
          scanTaskTracker.start(task.taskId)
        }
      } finally {
        scanPathBusy.value = null
      }
    })
  } catch (err) {
    console.error("[settings] rescan path failed", err)
    if (err instanceof HttpClientError && err.status === 409) {
      scanFeedbackError.value = t("settings.errScanConflict")
    } else if (err instanceof HttpClientError && err.apiError?.message) {
      scanFeedbackError.value = err.apiError.message
    } else {
      scanFeedbackError.value = t("settings.errScanStart")
    }
  }
}

async function revealLibraryPath(path: { id: string; title: string }) {
  scanFeedbackError.value = ""
  revealPathBusy.value = path.id
  try {
    await libraryService.revealLibraryPathInFileManager(path.id)
    pushAppToast(t("detail.revealSuccess"), {
      variant: "success",
      durationMs: 2400,
    })
  } catch (err) {
    console.error("[settings] reveal library path failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      scanFeedbackError.value = err.apiError.message
    } else if (err instanceof Error && err.message === "MOCK_REVEAL_NOT_SUPPORTED") {
      scanFeedbackError.value = t("detail.revealMockMode")
    } else {
      scanFeedbackError.value = t("detail.revealFailGeneric")
    }
  } finally {
    revealPathBusy.value = null
  }
}

async function onOrganizeLibraryChange(next: boolean) {
  organizeLibraryError.value = ""
  try {
    await withPreservedScroll(async () => {
      organizeLibrarySaving.value = true
      try {
        await libraryService.setOrganizeLibrary(next)
      } finally {
        organizeLibrarySaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] organize library toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      organizeLibraryError.value = err.apiError.message
    } else {
      organizeLibraryError.value = t("settings.errSaveTitle")
    }
  }
}

async function onAutoLibraryWatchChange(next: boolean) {
  autoLibraryWatchError.value = ""
  try {
    await withPreservedScroll(async () => {
      autoLibraryWatchSaving.value = true
      try {
        await libraryService.setAutoLibraryWatch(next)
      } finally {
        autoLibraryWatchSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] auto library watch toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      autoLibraryWatchError.value = err.apiError.message
    } else {
      autoLibraryWatchError.value = t("settings.errSaveTitle")
    }
  }
}

async function onAutoActorProfileScrapeChange(next: boolean) {
  autoActorProfileScrapeError.value = ""
  try {
    await withPreservedScroll(async () => {
      autoActorProfileScrapeSaving.value = true
      try {
        await libraryService.setAutoActorProfileScrape(next)
      } finally {
        autoActorProfileScrapeSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] auto actor profile scrape toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      autoActorProfileScrapeError.value = err.apiError.message
    } else {
      autoActorProfileScrapeError.value = t("settings.errSaveTitle")
    }
  }
}

async function onLaunchAtLoginChange(next: boolean) {
  launchAtLoginError.value = ""
  try {
    await withPreservedScroll(async () => {
      launchAtLoginSaving.value = true
      try {
        await libraryService.setLaunchAtLogin(next)
      } finally {
        launchAtLoginSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] launch-at-login toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      launchAtLoginError.value = err.apiError.message
    } else {
      launchAtLoginError.value = t("settings.errSaveTitle")
    }
  }
}

async function onCuratedExportFormatChange(next: CuratedFrameExportFormat) {
  if (next === curatedFrameExportFormat.value) return
  curatedExportFormatError.value = ""
  try {
    await withPreservedScroll(async () => {
      curatedExportFormatSaving.value = true
      try {
        await libraryService.setCuratedFrameExportFormat(next)
      } finally {
        curatedExportFormatSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] curated export format change failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      curatedExportFormatError.value = err.apiError.message
    } else {
      curatedExportFormatError.value = t("settings.errSaveTitle")
    }
  }
}

function onCuratedExportFormatSelect(next: unknown) {
  if (next !== "jpg" && next !== "webp" && next !== "png") return
  void onCuratedExportFormatChange(next)
}

async function onMetadataMovieModeAuto() {
  if (metadataMovieModeUi.value === "auto") return
  metadataMovieError.value = ""
  try {
    await withPreservedScroll(async () => {
      metadataMovieSaving.value = true
      try {
        await libraryService.setMetadataMovieProvider("")
        syncMetadataMovieModeUiFromServer()
      } finally {
        metadataMovieSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] metadata movie provider (auto) failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataMovieError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      metadataMovieError.value = err.message
    } else {
      metadataMovieError.value = t("settings.errSaveTitle")
    }
  }
}

async function onMetadataMovieModeSpecified() {
  if (!canPickSpecifiedMetadata.value) return
  if (metadataMovieModeUi.value === "specified") return
  metadataMovieError.value = ""
  const first = metadataMovieSelectOptions.value[0]
  try {
    await withPreservedScroll(async () => {
      metadataMovieSaving.value = true
      try {
        if (first) {
          await libraryService.setMetadataMovieProvider(first)
        }
        syncMetadataMovieModeUiFromServer()
      } finally {
        metadataMovieSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] metadata movie provider (specified) failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataMovieError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      metadataMovieError.value = err.message
    } else {
      metadataMovieError.value = t("settings.errSaveTitle")
    }
  }
}

async function onMetadataMovieSelect(next: unknown) {
  if (typeof next !== "string" || next === metadataMovieProvider.value) return
  metadataMovieError.value = ""
  try {
    await withPreservedScroll(async () => {
      metadataMovieSaving.value = true
      try {
        await libraryService.setMetadataMovieProvider(next)
        syncMetadataMovieModeUiFromServer()
      } finally {
        metadataMovieSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] metadata movie provider select failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataMovieError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      metadataMovieError.value = err.message
    } else {
      metadataMovieError.value = t("settings.errSaveTitle")
    }
  }
}

async function runFullScan() {
  scanFeedbackError.value = ""
  try {
    await withPreservedScroll(async () => {
      fullScanBusy.value = true
      try {
        const task = await libraryService.scanLibraryPaths()
        if (task?.taskId) {
          scanTaskTracker.start(task.taskId)
        }
      } finally {
        fullScanBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] full scan failed", err)
    if (err instanceof HttpClientError && err.status === 409) {
      scanFeedbackError.value = t("settings.errScanConflict")
    } else if (err instanceof HttpClientError && err.apiError?.message) {
      scanFeedbackError.value = err.apiError.message
    } else {
      scanFeedbackError.value = t("settings.errScanStart")
    }
  }
}

async function runMetadataRefreshForSelected() {
  metadataRefreshSuccess.value = ""
  metadataRefreshError.value = ""
  const paths = selectedMetadataRefreshPaths.value
  if (paths.length === 0) {
    metadataRefreshError.value = t("settings.errMetadataSelect")
    return
  }
  try {
    await withPreservedScroll(async () => {
      metadataRefreshBusy.value = true
      try {
        const dto = await libraryService.refreshMetadataForLibraryPaths(paths)
        const parts: string[] = [t("settings.metadataQueued", { n: dto.queued })]
        if (dto.skipped > 0) {
          parts.push(t("settings.metadataSkipped", { n: dto.skipped }))
        }
        if (dto.invalidPaths.length > 0) {
          parts.push(t("settings.metadataInvalid", { paths: dto.invalidPaths.join("；") }))
        }
        metadataRefreshSuccess.value = parts.join(" ")
      } finally {
        metadataRefreshBusy.value = false
      }
    })
  } catch (err) {
    console.error("[settings] metadata refresh by paths failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataRefreshError.value = err.apiError.message
    } else {
      metadataRefreshError.value = t("settings.errMetadataBatch")
    }
  }
}
</script>

<template>
  <Tabs
    v-model="activeSlug"
    :class="
      cn(
        'mx-auto flex h-full min-h-0 w-full max-w-7xl flex-col gap-6 pb-2 lg:flex-row lg:items-stretch lg:gap-6',
        SETTINGS_CONTROL_H32_CLASS,
        SETTINGS_LIBRARY_PATH_ACTION_ICONS_CLASS,
      )
    "
  >
    <nav
      class="w-full shrink-0 lg:max-h-[calc(100dvh-10.5rem)] lg:w-56 lg:overflow-y-auto lg:overscroll-contain lg:self-start lg:rounded-xl lg:border lg:border-border lg:bg-card lg:p-2"
      :aria-label="t('settings.navAriaLabel')"
    >
      <p class="sr-only">{{ t("settings.navJumpTo") }}</p>
      <div class="-mx-1 overflow-x-auto pb-2 lg:hidden">
        <div class="flex w-max min-w-full gap-2 px-1">
          <button
            v-for="item in settingsNavItems"
            :key="`mob-nav-${item.slug}`"
            type="button"
            :class="
              cn(
                'shrink-0 rounded-full px-3.5 py-2 text-sm font-medium transition-colors',
                activeSlug === item.slug
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground',
              )
            "
            @click="activeSlug = item.slug"
          >
            {{ t(item.labelKey) }}
          </button>
        </div>
      </div>

      <TabsList
        class="hidden h-auto min-h-0 w-full flex-col gap-1 border-0 bg-transparent p-0 text-muted-foreground lg:flex"
      >
        <TabsTrigger
          v-for="item in settingsNavItems"
          :key="`desk-${item.slug}`"
          :value="item.slug"
          class="h-auto w-full cursor-pointer flex-initial justify-start rounded-lg border border-transparent px-3 py-2 text-left text-sm shadow-none transition-colors duration-150 data-[state=inactive]:hover:bg-muted data-[state=inactive]:hover:text-foreground data-[state=active]:border-transparent data-[state=active]:bg-primary/10 data-[state=active]:font-medium data-[state=active]:text-primary data-[state=active]:shadow-none"
        >
          {{ t(item.labelKey) }}
        </TabsTrigger>
      </TabsList>
    </nav>

    <div
      :id="SETTINGS_SCROLL_ROOT_ID"
      :ref="bindSettingsScrollRoot"
      class="min-h-0 min-w-0 flex flex-1 flex-col overflow-x-hidden overflow-y-auto overscroll-contain [overflow-anchor:none] [scrollbar-gutter:stable]"
    >
    <TabsContent
      v-if="shouldRenderSettingsSection('overview')"
      value="overview"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-overview"
      class="space-y-6"
      :aria-label="t('settings.navOverview')"
    >
    <h2 class="sr-only">{{ t("settings.navOverview") }}</h2>
    <SettingsOverviewSection :dashboard-stats="dashboardStats" />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('general')"
      value="general"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-general"
      class="space-y-8"
      :aria-label="t('settings.navGeneral')"
    >
    <h2 class="sr-only">{{ t("settings.navGeneral") }}</h2>
    <SettingsGeneralSection
      v-model:locale="locale"
      :theme-preference="themePreference"
      :launch-at-login="launchAtLogin"
      :launch-at-login-saving="launchAtLoginSaving"
      :launch-at-login-disabled="launchAtLoginDisabled"
      :launch-at-login-unavailable-hint="launchAtLoginUnavailableHint"
      :launch-at-login-error="launchAtLoginError"
      :auto-save-ready="settingsAutoSaveReady"
      @change-theme="setThemeFromSelect"
      @change-launch-at-login="onLaunchAtLoginChange"
    />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('library')"
      value="library"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-library"
      class="space-y-6"
      :aria-label="t('settings.navLibrary')"
    >
    <h2 class="sr-only">{{ t("settings.navLibrary") }}</h2>
      <div class="flex w-full flex-col gap-6">
        <p
          v-if="scanFeedbackError"
          class="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive"
          role="alert"
        >
          {{ scanFeedbackError }}
        </p>
      <div class="break-inside-avoid">
        <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
          <CardHeader class="space-y-3 pb-2">
            <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
              <span
                class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
                aria-hidden="true"
              >
                <Database class="size-[1.15rem]" />
              </span>
              {{ t("settings.storageCardTitle") }}
            </CardTitle>
            <CardDescription
              class="text-xs leading-relaxed text-pretty text-muted-foreground"
            >
              {{ t("settings.storageCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 pt-2">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="flex min-w-0 flex-1 flex-col gap-3">
                <p class="font-medium">{{ t("settings.libraryPaths") }}</p>
              </div>
              <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
                <template v-if="!libraryPathsBatchMode">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="gap-1.5 rounded-xl"
                    @click="enterLibraryPathsBatchMode"
                  >
                    <ListChecks class="size-4 opacity-80" aria-hidden="true" />
                    {{ t("library.batchManage") }}
                  </Button>
                </template>
                <template v-else>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="gap-1.5 rounded-xl"
                    :disabled="libraryPathsList.length === 0"
                    @click="selectAllMetadataPaths"
                  >
                    <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
                    {{ t("settings.selectAllPaths") }}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="gap-1.5 rounded-xl"
                    @click="clearMetadataPathSelection"
                  >
                    {{ t("settings.clearSelection") }}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    class="gap-1.5 rounded-xl"
                    :disabled="!hasMetadataPathSelection || metadataRefreshBusy"
                    @click="runMetadataRefreshForSelected"
                  >
                    <Sparkles class="size-4 shrink-0" aria-hidden="true" />
                    {{
                      metadataRefreshBusy ? t("settings.submitting") : t("settings.refreshMetadata")
                    }}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
                    @click="exitLibraryPathsBatchMode"
                  >
                    <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
                    {{ t("library.batchExitToolbar") }}
                  </Button>
                </template>
              </div>
            </div>

            <Dialog v-model:open="removePathDialogOpen">
                <DialogContent
                  :class="cn('rounded-3xl border-border/50 sm:max-w-md', SETTINGS_CONTROL_H32_CLASS)"
                >
                  <DialogHeader>
                    <DialogTitle>{{ t("settings.removePathConfirmTitle") }}</DialogTitle>
                    <DialogDescription>
                      <div class="space-y-2">
                        <p class="text-pretty">
                          {{
                            t("settings.removePathConfirmDesc", {
                              title: removePathPending?.title ?? "—",
                            })
                          }}
                        </p>
                        <p
                          v-if="removePathPending?.path"
                          class="break-all font-mono text-xs text-muted-foreground"
                        >
                          {{ removePathPending.path }}
                        </p>
                      </div>
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter class="gap-3">
                    <DialogClose as-child>
                      <Button
                        type="button"
                        variant="outline"
                        class="rounded-2xl"
                        :disabled="removePathBusy"
                      >
                        {{ t("common.cancel") }}
                      </Button>
                    </DialogClose>
                    <Button
                      type="button"
                      variant="destructive"
                      class="rounded-2xl"
                      :disabled="removePathBusy || !removePathPending"
                      @click="confirmRemoveLibraryPath"
                    >
                      {{
                        removePathBusy
                          ? t("settings.removePathConfirmWorking")
                          : t("settings.removePathConfirmAction")
                      }}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>

            <p v-if="metadataRefreshSuccess" class="text-sm text-primary">
              {{ metadataRefreshSuccess }}
            </p>
            <p
              v-if="metadataRefreshError"
              class="text-sm text-destructive"
              role="alert"
            >
              {{ metadataRefreshError }}
            </p>

            <div class="flex flex-col gap-3">
              <div
                v-for="path in libraryPathsList"
                :key="path.id"
                class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
              >
                <template v-if="editingLibraryPathId === path.id">
                  <div class="flex flex-col gap-3">
                    <div class="flex flex-col gap-3">
                      <p class="text-xs font-medium text-muted-foreground">{{ t("settings.pathReadonly") }}</p>
                      <p class="break-all font-mono text-sm text-muted-foreground">{{ path.path }}</p>
                    </div>
                    <div class="flex flex-col gap-3">
                      <label class="text-sm font-medium" :for="`edit-title-${path.id}`">{{
                        t("settings.pathTitleLabel")
                      }}</label>
                      <Input
                        :id="`edit-title-${path.id}`"
                        v-model="editLibraryTitleDraft"
                        class="rounded-xl"
                        :placeholder="t('settings.displayName')"
                        autocomplete="off"
                        @keydown.enter.prevent="saveLibraryPathTitle(path.id)"
                      />
                      <p class="text-xs text-muted-foreground">
                        {{ t("settings.editTitleHint") }}
                      </p>
                      <p v-if="editTitleError" class="text-sm text-destructive">
                        {{ editTitleError }}
                      </p>
                    </div>
                    <div class="flex flex-wrap gap-3">
                      <Button
                        type="button"
                        class="rounded-2xl"
                        :disabled="editTitleBusy"
                        @click="saveLibraryPathTitle(path.id)"
                      >
                        {{ editTitleBusy ? t("common.saving") : t("settings.saveTitle") }}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        class="rounded-2xl"
                        :disabled="editTitleBusy"
                        @click="cancelEditLibraryTitle"
                      >
                        {{ t("common.cancel") }}
                      </Button>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div class="flex min-w-0 flex-1 items-start gap-3">
                      <input
                        v-if="libraryPathsBatchMode"
                        type="checkbox"
                        class="mt-1 size-4 shrink-0 cursor-pointer rounded border border-input accent-primary"
                        :checked="isMetadataPathChecked(path.path)"
                        :aria-label="t('settings.includeInMetadataRefresh', { title: path.title })"
                        @change="toggleMetadataPathSelection(path.path)"
                      />
                      <div class="flex min-w-0 flex-1 flex-col gap-3">
                        <p class="font-medium">{{ path.title }}</p>
                        <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
                      </div>
                    </div>
                    <div class="library-path-toolbar flex flex-wrap items-center gap-2">
                      <SettingsLibraryPathActions
                        :path="path"
                        :reveal-busy="revealPathBusy === path.id"
                        :scan-busy="scanPathBusy === path.path"
                        @reveal="revealLibraryPath"
                        @edit="startEditLibraryTitle"
                        @rescan="rescanPath($event.path)"
                        @remove="openRemovePathConfirm"
                      />
                    </div>
                  </div>
                </template>
              </div>
            </div>

            <div class="flex flex-wrap justify-start gap-2 pt-1">
              <Dialog v-model:open="addPathDialogOpen">
                <DialogTrigger as-child>
                  <Button type="button" class="rounded-2xl">
                    <FolderPlus data-icon="inline-start" />
                    {{ t("settings.addPath") }}
                  </Button>
                </DialogTrigger>

                <DialogContent
                  :class="cn('rounded-3xl border-border/50 sm:max-w-md', SETTINGS_CONTROL_H32_CLASS)"
                >
                  <DialogHeader>
                    <DialogTitle>{{ t("settings.addPathDialogTitle") }}</DialogTitle>
                    <DialogDescription>
                      {{ t("settings.addPathDialogDesc") }}
                      <span class="font-mono text-xs">D:\Media\JAV</span> 或
                      <span class="font-mono text-xs">/home/user/Videos</span>。
                    </DialogDescription>
                  </DialogHeader>

                  <div class="flex flex-col gap-3">
                    <div class="flex flex-col gap-3">
                      <label class="text-sm font-medium" for="new-lib-path">{{ t("settings.absolutePath") }}</label>
                      <div class="flex flex-col gap-3 sm:flex-row sm:items-stretch">
                        <Input
                          id="new-lib-path"
                          v-model="newPath"
                          class="rounded-xl sm:min-w-0 sm:flex-1"
                          placeholder="D:\Media\JAV\Library"
                          autocomplete="off"
                          @input="clearPathAddError"
                        />
                        <Button
                          type="button"
                          variant="secondary"
                          class="rounded-2xl sm:shrink-0"
                          :disabled="pickDirectoryBusy"
                          @click="browseForDirectory"
                        >
                          <FolderOpen data-icon="inline-start" />
                          {{ pickDirectoryBusy ? t("settings.picking") : t("settings.pickFolder") }}
                        </Button>
                      </div>
                      <p
                        v-if="directoryHintDisplay"
                        class="text-sm leading-relaxed text-muted-foreground whitespace-pre-line"
                      >
                        {{ directoryHintDisplay }}
                      </p>
                      <p
                        v-if="newPath.trim() && !isAbsoluteLibraryPath(newPath)"
                        class="text-sm text-destructive"
                      >
                        {{ t("settings.notAbsolute") }}
                      </p>
                      <p v-if="pathAddError" class="text-sm text-destructive">
                        {{ pathAddError }}
                      </p>
                    </div>
                    <div class="flex flex-col gap-3">
                      <label class="text-sm font-medium" for="new-lib-title">{{ t("settings.optionalPathTitle") }}</label>
                      <Input
                        id="new-lib-title"
                        v-model="newPathTitle"
                        class="rounded-xl"
                        :placeholder="t('settings.displayName')"
                        autocomplete="off"
                      />
                    </div>
                  </div>

                  <DialogFooter>
                    <DialogClose as-child>
                      <Button type="button" variant="outline" class="rounded-2xl">
                        {{ t("common.cancel") }}
                      </Button>
                    </DialogClose>
                    <Button
                      type="button"
                      class="rounded-2xl"
                      :disabled="addBusy || !canSaveNewPath"
                      :title="
                        addBusy || canSaveNewPath ? undefined : t('settings.savePathDisabledTitle')
                      "
                      @click="submitAddPath"
                    >
                      {{ addBusy ? t("common.saving") : t("settings.savePath") }}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          </CardContent>
        </Card>
      </div>

      <SettingsOrganizeSection
        :organize-library="organizeLibrary"
        :organize-library-saving="organizeLibrarySaving"
        :organize-library-error="organizeLibraryError"
        @change-organize-library="onOrganizeLibraryChange"
      />

      </div>
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('metadata')"
      value="metadata"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-metadata"
      class="space-y-6"
      :aria-label="t('settings.navMetadata')"
    >
    <h2 class="sr-only">{{ t("settings.navMetadata") }}</h2>
      <div class="break-inside-avoid">
        <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
          <CardHeader class="space-y-3 pb-2">
            <CardTitle
              class="flex flex-wrap items-center gap-2.5 text-lg font-semibold tracking-tight"
            >
              <span class="flex min-w-0 items-center gap-2.5">
                <span
                  class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
                  aria-hidden="true"
                >
                  <Sparkles class="size-4" />
                </span>
                <span class="min-w-0">{{ t("settings.metadataMovieProviderTitle") }}</span>
              </span>
            </CardTitle>
            <CardDescription
              class="text-xs leading-relaxed text-pretty text-muted-foreground"
            >
              {{ t("settings.metadataMovieProviderDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 pt-2">
            <SettingsMetadataAutomationSection
              :use-web-api="useWebApi"
              :provider-ping-all-busy="providerPingAllBusy"
              :provider-ping-one-name="providerPingOneName"
              :provider-health-ping-all-summary="providerHealthPingAllSummary"
              :provider-health-ping-error="providerHealthPingError"
              :auto-library-watch="autoLibraryWatch"
              :auto-library-watch-saving="autoLibraryWatchSaving"
              :auto-library-watch-error="autoLibraryWatchError"
              :auto-actor-profile-scrape="autoActorProfileScrape"
              :auto-actor-profile-scrape-saving="autoActorProfileScrapeSaving"
              :auto-actor-profile-scrape-error="autoActorProfileScrapeError"
              @ping-all-providers="pingAllMetadataProviders"
              @change-auto-library-watch="onAutoLibraryWatchChange"
              @change-auto-actor-profile-scrape="onAutoActorProfileScrapeChange"
            />

            <SettingsMetadataModeSection
              :metadata-movie-mode-ui="metadataMovieModeUi"
              :metadata-movie-saving="metadataMovieSaving"
              :metadata-movie-chain-saving="metadataMovieChainSaving"
              :provider-ping-all-busy="providerPingAllBusy"
              :can-pick-specified-metadata="canPickSpecifiedMetadata"
              :can-use-metadata-chain-mode="canUseMetadataChainMode"
              @select-auto="onMetadataMovieModeAuto"
              @select-specified="onMetadataMovieModeSpecified"
              @select-chain="onMetadataMovieModeChain"
            />

            <!-- Single Provider Selection -->
            <SettingsMetadataProviderSelectSection
              v-if="metadataMovieModeUi === 'specified' && canPickSpecifiedMetadata"
              :use-web-api="useWebApi"
              :metadata-movie-provider="metadataMovieProvider"
              :metadata-movie-select-options="metadataMovieSelectOptions"
              :metadata-movie-saving="metadataMovieSaving"
              :provider-ping-all-busy="providerPingAllBusy"
              :provider-ping-one-name="providerPingOneName"
              :current-provider-health="
                healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '') ?? null
              "
              @select-provider="onMetadataMovieSelect"
              @ping-provider="pingOneMetadataProvider"
            />

            <SettingsMetadataProviderChainSection
              v-if="metadataMovieModeUi === 'chain'"
              :use-web-api="useWebApi"
              :can-pick-specified-metadata="canPickSpecifiedMetadata"
              :provider-chain-draft="providerChainDraft"
              :available-providers-for-chain="availableProvidersForChain"
              :selected-provider-to-add="selectedProviderToAdd"
              :chain-drag-from-index="chainDragFromIndex"
              :metadata-movie-chain-saving="metadataMovieChainSaving"
              :metadata-movie-chain-error="metadataMovieChainError"
              :provider-ping-all-busy="providerPingAllBusy"
              :provider-ping-one-name="providerPingOneName"
              :provider-health-by-name="providerHealthByName"
              @drag-start="onChainDragStart"
              @drag-over="onChainDragOver"
              @drop-provider="onChainDrop"
              @drag-end="onChainDragEnd"
              @ping-provider="pingOneMetadataProvider"
              @remove-provider="removeProviderFromChain"
              @update:selected-provider-to-add="selectedProviderToAdd = $event"
              @add-provider="addProviderToChain"
              @save-provider-chain="saveProviderChain"
            />

            <p
              v-if="!canPickSpecifiedMetadata && metadataMovieModeUi !== 'chain'"
              class="text-sm text-muted-foreground"
            >
              {{ t("settings.metadataMovieProviderNoList") }}
            </p>
            <p
              v-if="metadataMovieSaving"
              class="text-xs text-muted-foreground motion-safe:animate-pulse"
            >
              {{ t("settings.metadataMovieProviderSyncing") }}
            </p>
            <p v-if="metadataMovieError" class="text-sm text-destructive">
              {{ metadataMovieError }}
            </p>

            <div
              class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="min-w-0 flex flex-col gap-3">
                <p class="text-sm font-semibold">{{ t("settings.triggerScrape") }}</p>
                <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                  {{ t("settings.triggerScrapeHint") }}
                </p>
              </div>
              <Button
                type="button"
                variant="default"
                class="h-11 shrink-0 rounded-2xl px-5 font-medium"
                :disabled="triggerScrapeCardBusy"
                @click="runTriggerScrapeAllLibraryRoots"
              >
                <RefreshCw
                  data-icon="inline-start"
                  class="size-4"
                  :class="{ 'motion-safe:animate-spin': triggerScrapeCardBusy }"
                />
                {{
                  triggerScrapeCardBusy
                    ? t("settings.triggerScrapeRunning")
                    : t("settings.triggerScrapeRunButton")
                }}
              </Button>
            </div>
            <p
              v-if="triggerScrapeCardSuccess"
              class="text-sm text-primary"
            >
              {{ triggerScrapeCardSuccess }}
            </p>
            <p
              v-if="triggerScrapeCardError"
              class="text-sm text-destructive"
              role="alert"
            >
              {{ triggerScrapeCardError }}
            </p>
          </CardContent>
        </Card>
      </div>
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('network')"
      value="network"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-network"
      class="space-y-6"
      :aria-label="t('settings.navNetwork')"
    >
    <h2 class="sr-only">{{ t("settings.navNetwork") }}</h2>
      <SettingsNetworkSection
        v-model:proxy-enabled="proxyEnabledDraft"
        v-model:proxy-scheme="proxySchemeDraft"
        v-model:proxy-host="proxyHostDraft"
        v-model:proxy-port="proxyPortDraft"
        v-model:proxy-username="proxyUsernameDraft"
        v-model:proxy-password="proxyPasswordDraft"
        v-model:proxy-auth-expanded="proxyAuthExpanded"
        :use-web-api="useWebApi"
        :proxy-saving="proxySaving"
        :proxy-outbound-ping-busy="proxyOutboundPingBusy"
        :proxy-javbus-busy="proxyJavbusBusy"
        :proxy-google-busy="proxyGoogleBusy"
        :proxy-status-message="proxyStatusMessage"
        @save-proxy="saveProxySettings"
        @test-proxy-javbus="testProxyJavbus"
        @test-proxy-google="testProxyGoogle"
      />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('curated')"
      value="curated"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-curated"
      class="space-y-6"
      :aria-label="t('settings.navCurated')"
    >
    <h2 class="sr-only">{{ t("settings.navCurated") }}</h2>
      <SettingsCuratedSection
        v-model:curated-save-mode="curatedSaveMode"
        :capture-shortcut-label="curatedCaptureShortcutLabel"
        :directory-supported="curatedDirectorySupported"
        :curated-frame-export-format="curatedFrameExportFormat"
        :curated-export-format-options="curatedExportFormatOptions"
        :curated-export-format-saving="curatedExportFormatSaving"
        :curated-export-dir-label="curatedExportDirLabel"
        :curated-export-pick-busy="curatedExportPickBusy"
        :curated-export-error="curatedExportError"
        :curated-export-format-error="curatedExportFormatError"
        @change-export-format="onCuratedExportFormatSelect"
        @pick-export-directory="pickCuratedExportDirectory"
        @clear-export-directory="clearCuratedExportDirectory"
      />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('playback')"
      value="playback"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-playback"
      class="space-y-6"
      :aria-label="t('settings.navPlayback')"
    >
    <h2 class="sr-only">{{ t("settings.navPlayback") }}</h2>
      <SettingsPlaybackSection :auto-save-ready="settingsAutoSaveReady" />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('maintenance')"
      value="maintenance"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-maintenance"
      class="space-y-6"
      :aria-label="t('settings.navMaintenance')"
    >
    <h2 class="sr-only">{{ t("settings.navMaintenance") }}</h2>
      <SettingsMaintenanceSection
        :full-scan-busy="fullScanBusy"
        @run-full-scan="runFullScan"
      />
    </section>
    </TabsContent>

    <TabsContent
      v-if="shouldRenderSettingsSection('about')"
      value="about"
      class="mt-0 min-w-0 flex-1 outline-none"
    >
    <section
      id="settings-section-about"
      class="space-y-6"
      :aria-label="t('settings.navAbout')"
    >
    <h2 class="sr-only">{{ t("settings.navAbout") }}</h2>
    <SettingsAboutSection
      :is-vite-dev="isViteDev"
      :use-web-api="useWebApi"
      :vite-mode="viteMode"
      :about-health="aboutHealth"
      :about-health-loading="aboutHealthLoading"
      :about-health-error="aboutHealthError"
      :backend-version-display="aboutBackendVersionDisplay"
      :backend-version-status="aboutBackendVersionStatus"
      @refresh-health="void loadAboutHealth()"
    />
    </section>
    </TabsContent>
    </div>
  </Tabs>
</template>
