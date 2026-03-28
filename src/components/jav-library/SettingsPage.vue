<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { ProviderHealthDTO, ProviderHealthStatus, ProxySettingsDTO } from "@/api/types"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { useSettingsScrollPreserve } from "@/composables/use-settings-scroll-preserve"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import {
  Activity,
  ChevronDown,
  FolderOpen,
  Info,
  FolderPlus,
  GripVertical,
  ImageDown,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  ScanSearch,
  Sparkles,
  Trash2,
  X,
} from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import {
  getStoredDirectoryHandle,
  setStoredDirectoryHandle,
  supportsFileSystemAccess,
} from "@/lib/curated-frames/db"
import {
  getCuratedFrameSaveMode,
  setCuratedFrameSaveMode,
} from "@/lib/curated-frames/settings-storage"
import { useLibraryService } from "@/services/library-service"
import SettingsPageNav from "@/components/jav-library/SettingsPageNav.vue"

const { t, locale } = useI18n()
const libraryService = useLibraryService()
const scanTaskTracker = useScanTaskTracker()
const { withPreservedScroll, withSyncPreservedScroll } = useSettingsScrollPreserve()
/** Plain object services don't unwrap nested ComputedRefs in templates */
const libraryPathsList = computed(() => libraryService.libraryPaths.value)
const hardwareDecode = ref(true)

const addPathDialogOpen = ref(false)
const newPath = ref("")
const newPathTitle = ref("")
const addBusy = ref(false)
const scanPathBusy = ref<string | null>(null)
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
/** 后台保存中：仅作轻提示，不禁用开关以免打断动画、体感卡顿 */
const organizeLibrarySaving = ref(false)
const organizeLibraryError = ref("")
const extendedLibraryImportSaving = ref(false)
const extendedLibraryImportError = ref("")
const autoLibraryWatchSaving = ref(false)
const autoLibraryWatchError = ref("")
const metadataMovieSaving = ref(false)
const metadataMovieError = ref("")

const proxyEnabledDraft = ref(false)
const proxyUrlDraft = ref("")
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

const proxyOutboundPingBusy = computed(
  () => proxyJavbusBusy.value || proxyGoogleBusy.value,
)

/** 代理用户名/密码折叠；有已保存认证信息时默认展开 */
const proxyAuthExpanded = ref(false)

function syncProxyDraftFromService() {
  const p = libraryService.proxy.value
  proxyEnabledDraft.value = Boolean(p.enabled)
  proxyUrlDraft.value = (p.url ?? "").trim()
  proxyUsernameDraft.value = (p.username ?? "").trim()
  proxyPasswordDraft.value = p.password ?? ""
  const hasAuth =
    !!(p.username?.trim() || (p.password ?? "").length > 0)
  if (hasAuth) {
    proxyAuthExpanded.value = true
  }
}

async function saveProxySettings() {
  proxyError.value = ""
  if (proxyEnabledDraft.value && !proxyUrlDraft.value.trim()) {
    proxyError.value = t("settings.proxyUrlRequired")
    return
  }
  const body: ProxySettingsDTO = {
    enabled: proxyEnabledDraft.value,
    url: proxyUrlDraft.value.trim() || undefined,
    username: proxyUsernameDraft.value.trim() || undefined,
    password: proxyPasswordDraft.value || undefined,
  }
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
  } catch (err) {
    console.error("[settings] save proxy failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      proxyError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      proxyError.value = err.message
    } else {
      proxyError.value = t("settings.errSaveTitle")
    }
  }
}

async function testProxyJavbus() {
  proxyError.value = ""
  proxyJavbusResult.value = ""
  proxyJavbusResultOk.value = null
  if (proxyEnabledDraft.value && !proxyUrlDraft.value.trim()) {
    proxyError.value = t("settings.proxyUrlRequired")
    return
  }
  if (!useWebApi) {
    return
  }
  await withPreservedScroll(async () => {
    proxyJavbusBusy.value = true
      try {
        const body = {
          proxy: {
            enabled: proxyEnabledDraft.value,
            url: proxyUrlDraft.value.trim() || undefined,
            username: proxyUsernameDraft.value.trim() || undefined,
            password: proxyPasswordDraft.value || undefined,
          } satisfies ProxySettingsDTO,
        }
        const res = await api.pingProxyJavbus(body)
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
  proxyError.value = ""
  proxyGoogleResult.value = ""
  proxyGoogleResultOk.value = null
  if (proxyEnabledDraft.value && !proxyUrlDraft.value.trim()) {
    proxyError.value = t("settings.proxyUrlRequired")
    return
  }
  if (!useWebApi) {
    return
  }
  await withPreservedScroll(async () => {
    proxyGoogleBusy.value = true
      try {
        const body = {
          proxy: {
            enabled: proxyEnabledDraft.value,
            url: proxyUrlDraft.value.trim() || undefined,
            username: proxyUsernameDraft.value.trim() || undefined,
            password: proxyPasswordDraft.value || undefined,
          } satisfies ProxySettingsDTO,
        }
        const res = await api.pingProxyGoogle(body)
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
const extendedLibraryImport = computed(() => libraryService.extendedLibraryImport.value)
const autoLibraryWatch = computed(() => libraryService.autoLibraryWatch.value)

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
const metadataMovieMode = computed<"auto" | "specified" | "chain">(() => {
  const chain = libraryService.metadataMovieProviderChain.value
  if (chain.length > 0) return "chain"
  return metadataMovieProvider.value === "" ? "auto" : "specified"
})

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

function providerHealthStatusClass(status: ProviderHealthStatus): string {
  if (status === "ok") return "border-emerald-500/40 bg-emerald-500/10 text-emerald-200"
  if (status === "degraded") return "border-amber-500/40 bg-amber-500/10 text-amber-100"
  return "border-destructive/40 bg-destructive/10 text-destructive"
}

function providerHealthStatusLabel(status: ProviderHealthStatus): string {
  if (status === "ok") return t("settings.providerHealthStatusOk")
  if (status === "degraded") return t("settings.providerHealthStatusDegraded")
  return t("settings.providerHealthStatusFail")
}

async function pingAllMetadataProviders() {
  if (!useWebApi) return
  providerHealthPingError.value = ""
  providerHealthPingAllSummary.value = ""
  try {
    await withPreservedScroll(async () => {
      providerPingAllBusy.value = true
      try {
        const res = await api.pingAllProviders()
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
        const dto = await api.pingProvider(name.trim())
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

function moveProviderInChain(index: number, direction: "up" | "down") {
  withSyncPreservedScroll(() => {
    const draft = [...providerChainDraft.value]
    if (direction === "up" && index > 0) {
      ;[draft[index - 1], draft[index]] = [draft[index], draft[index - 1]]
    } else if (direction === "down" && index < draft.length - 1) {
      ;[draft[index], draft[index + 1]] = [draft[index + 1], draft[index]]
    }
    providerChainDraft.value = draft
  })
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

function onMetadataMovieModeChain() {
  if (metadataMovieModeUi.value === "chain") return
  withSyncPreservedScroll(() => {
    metadataMovieModeUi.value = "chain"
    metadataMovieChainError.value = ""
    metadataMovieError.value = ""
    initProviderChainDraft()
  })
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

async function removePath(id: string) {
  try {
    await withPreservedScroll(() => libraryService.removeLibraryPath(id))
  } catch (err) {
    console.error("[settings] remove library path failed", err)
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

async function onExtendedLibraryImportChange(next: boolean) {
  extendedLibraryImportError.value = ""
  try {
    await withPreservedScroll(async () => {
      extendedLibraryImportSaving.value = true
      try {
        await libraryService.setExtendedLibraryImport(next)
      } finally {
        extendedLibraryImportSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] extended library import toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      extendedLibraryImportError.value = err.apiError.message
    } else {
      extendedLibraryImportError.value = t("settings.errSaveTitle")
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
  <div class="mx-auto flex w-full max-w-7xl flex-col gap-6 pb-2 lg:flex-row lg:items-start lg:gap-8">
    <SettingsPageNav />
    <div class="min-w-0 flex flex-1 flex-col gap-6">
    <section
      id="settings-section-overview"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navOverview')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navOverview") }}
    </h2>
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <Card
        v-for="stat in dashboardStats"
        :key="stat.labelKey"
        class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5"
      >
        <CardHeader class="gap-1">
          <CardDescription>{{ t(stat.labelKey) }}</CardDescription>
          <CardTitle class="text-2xl">{{ stat.value }}</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">{{ t(stat.detailKey) }}</p>
        </CardContent>
      </Card>
    </div>
    </section>

    <section
      id="settings-section-general"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navGeneral')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navGeneral") }}
    </h2>
    <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
      <CardHeader
        class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:space-y-0"
      >
        <div class="min-w-0 flex-1 space-y-1">
          <CardTitle>{{ t("settings.language") }}</CardTitle>
          <CardDescription>{{ t("settings.languageHint") }}</CardDescription>
        </div>
        <Select v-model="locale">
          <SelectTrigger
            size="sm"
            class="h-8 w-full min-w-[9.5rem] shrink-0 rounded-xl border-border/70 sm:ml-4 sm:w-40"
            :aria-label="t('settings.language')"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent align="end" class="rounded-xl border-border/70">
            <SelectItem value="zh-CN">{{ t("settings.langZh") }}</SelectItem>
            <SelectItem value="en">{{ t("settings.langEn") }}</SelectItem>
            <SelectItem value="ja">{{ t("settings.langJa") }}</SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
    </Card>
    </section>

    <section
      id="settings-section-library"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navLibrary')"
    >
    <div class="flex flex-col gap-2 px-1">
      <div class="flex flex-col gap-1">
        <h2 class="text-3xl font-semibold tracking-tight">{{ t("settings.pageTitle") }}</h2>
        <p class="text-sm text-muted-foreground">
          {{ t("settings.pageSubtitle") }}
        </p>
        <p
          v-if="scanFeedbackError"
          class="text-sm text-destructive"
          role="alert"
        >
          {{ scanFeedbackError }}
        </p>
      </div>
    </div>

      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.storageCardTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.storageCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex flex-col gap-1">
                <p class="font-medium">{{ t("settings.libraryPaths") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.libraryPathsHint") }}
                  <span class="font-mono text-xs">D:\Media\JAV</span> 或
                  <span class="font-mono text-xs">/home/user/Videos</span>。
                </p>
              </div>
              <Dialog v-model:open="addPathDialogOpen">
                <DialogTrigger as-child>
                  <Button type="button" class="rounded-2xl">
                    <FolderPlus data-icon="inline-start" />
                    {{ t("settings.addPath") }}
                  </Button>
                </DialogTrigger>

                <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>{{ t("settings.addPathDialogTitle") }}</DialogTitle>
                    <DialogDescription>
                      {{ t("settings.addPathDialogDesc") }}
                      <span class="font-mono text-xs">D:\Media\JAV</span> 或
                      <span class="font-mono text-xs">/home/user/Videos</span>。
                    </DialogDescription>
                  </DialogHeader>

                  <div class="flex flex-col gap-4">
                    <div class="flex flex-col gap-2">
                      <label class="text-sm font-medium" for="new-lib-path">{{ t("settings.absolutePath") }}</label>
                      <div class="flex flex-col gap-2 sm:flex-row sm:items-stretch">
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
                    <div class="flex flex-col gap-2">
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

            <div
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/20 p-4 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
            >
              <div class="flex min-w-0 flex-col gap-1">
                <p class="text-sm font-medium">{{ t("settings.metadataSectionTitle") }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ t("settings.metadataSectionHint") }}
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="rounded-xl"
                  @click="selectAllMetadataPaths"
                >
                  {{ t("settings.selectAllPaths") }}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="rounded-xl"
                  @click="clearMetadataPathSelection"
                >
                  {{ t("settings.clearSelection") }}
                </Button>
                <Button
                  type="button"
                  class="rounded-xl"
                  :disabled="!hasMetadataPathSelection || metadataRefreshBusy"
                  @click="runMetadataRefreshForSelected"
                >
                  <Sparkles data-icon="inline-start" class="size-4" />
                  {{ metadataRefreshBusy ? t("settings.submitting") : t("settings.refreshMetadata") }}
                </Button>
              </div>
            </div>
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
                class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-background/50 p-4"
              >
                <template v-if="editingLibraryPathId === path.id">
                  <div class="flex flex-col gap-3">
                    <div class="flex flex-col gap-1">
                      <p class="text-xs font-medium text-muted-foreground">{{ t("settings.pathReadonly") }}</p>
                      <p class="break-all font-mono text-sm text-muted-foreground">{{ path.path }}</p>
                    </div>
                    <div class="flex flex-col gap-2">
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
                    <div class="flex flex-wrap gap-2">
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
                        type="checkbox"
                        class="mt-1 size-4 shrink-0 cursor-pointer rounded border border-input accent-primary"
                        :checked="isMetadataPathChecked(path.path)"
                        :aria-label="t('settings.includeInMetadataRefresh', { title: path.title })"
                        @change="toggleMetadataPathSelection(path.path)"
                      />
                      <div class="flex min-w-0 flex-1 flex-col gap-1">
                        <p class="font-medium">{{ path.title }}</p>
                        <p class="break-all text-sm text-muted-foreground">{{ path.path }}</p>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        class="rounded-2xl"
                        @click="startEditLibraryTitle(path)"
                      >
                        <Pencil data-icon="inline-start" />
                        {{ t("settings.editTitle") }}
                      </Button>
                      <Button
                        type="button"
                        variant="secondary"
                        class="rounded-2xl"
                        :disabled="scanPathBusy === path.path"
                        @click="rescanPath(path.path)"
                      >
                        <RefreshCw
                          data-icon="inline-start"
                          :class="scanPathBusy === path.path ? 'animate-spin' : ''"
                        />
                        {{ t("settings.rescan") }}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        class="rounded-2xl"
                        :aria-label="t('settings.removePathAria', { title: path.title })"
                        @click="removePath(path.id)"
                      >
                        <Trash2 class="size-4" />
                      </Button>
                    </div>
                  </div>
                </template>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.organizeTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.organizeDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div
              class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="organizeLibrarySaving"
            >
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <p class="font-medium">{{ t("settings.organizeSwitch") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.organizeHint") }}
                </p>
                <p
                  v-if="organizeLibrarySaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.organizeSyncing") }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="organizeLibrary"
                @update:model-value="onOrganizeLibraryChange"
              />
            </div>
            <p v-if="organizeLibraryError" class="text-sm text-destructive">
              {{ organizeLibraryError }}
            </p>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-metadata"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navMetadata')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navMetadata") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.metadataMovieProviderTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.metadataMovieProviderDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <div
              v-if="useWebApi"
              class="flex flex-col gap-2 rounded-2xl border border-border/70 bg-muted/15 p-4"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="min-w-0">
                  <p class="text-sm font-medium">{{ t("settings.providerHealthTitle") }}</p>
                  <p class="mt-0.5 text-xs text-muted-foreground">
                    {{ t("settings.providerHealthHint") }}
                  </p>
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="shrink-0 rounded-xl"
                  :disabled="providerPingAllBusy || providerPingOneName != null"
                  @click="pingAllMetadataProviders"
                >
                  <RefreshCw
                    class="mr-1.5 size-4"
                    :class="{ 'motion-safe:animate-spin': providerPingAllBusy }"
                  />
                  {{
                    providerPingAllBusy
                      ? t("settings.providerHealthPinging")
                      : t("settings.providerHealthPingAll")
                  }}
                </Button>
              </div>
              <p v-if="providerHealthPingAllSummary" class="text-xs text-muted-foreground">
                {{ providerHealthPingAllSummary }}
              </p>
              <p v-if="providerHealthPingError" class="text-sm text-destructive">
                {{ providerHealthPingError }}
              </p>
            </div>
            <p
              v-else
              class="rounded-2xl border border-border/60 bg-muted/10 px-4 py-3 text-sm text-muted-foreground"
            >
              {{ t("settings.providerHealthMockHint") }}
            </p>

            <div
              class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="autoLibraryWatchSaving"
            >
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <p class="font-medium">{{ t("settings.autoScrape") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.autoScrapeHint") }}
                </p>
                <p
                  v-if="autoLibraryWatchSaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.autoLibraryWatchSyncing") }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="autoLibraryWatch"
                @update:model-value="onAutoLibraryWatchChange"
              />
            </div>
            <p v-if="autoLibraryWatchError" class="text-sm text-destructive">
              {{ autoLibraryWatchError }}
            </p>

            <fieldset
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="metadataMovieSaving || metadataMovieChainSaving || providerPingAllBusy"
            >
              <legend class="px-1 text-sm font-medium">{{ t("settings.metadataMovieProviderMode") }}</legend>
              <label class="flex cursor-pointer items-start gap-3 rounded-xl p-2 hover:bg-muted/40">
                <input
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="metadata-movie-mode"
                  value="auto"
                  :disabled="metadataMovieSaving"
                  :checked="metadataMovieModeUi === 'auto'"
                  @change="onMetadataMovieModeAuto"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.metadataMovieProviderAuto") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.metadataMovieProviderAutoHint") }}
                  </span>
                </span>
              </label>
              <label
                class="flex items-start gap-3 rounded-xl p-2"
                :class="
                  canPickSpecifiedMetadata
                    ? 'cursor-pointer hover:bg-muted/40'
                    : 'cursor-not-allowed opacity-60'
                "
              >
                <input
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="metadata-movie-mode"
                  value="specified"
                  :checked="metadataMovieModeUi === 'specified'"
                  :disabled="metadataMovieSaving || !canPickSpecifiedMetadata"
                  @change="onMetadataMovieModeSpecified"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.metadataMovieProviderSpecified") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.metadataMovieProviderSpecifiedHint") }}
                  </span>
                </span>
              </label>
              <label
                class="flex items-start gap-3 rounded-xl p-2"
                :class="
                  canUseMetadataChainMode
                    ? 'cursor-pointer hover:bg-muted/40'
                    : 'cursor-not-allowed opacity-60'
                "
              >
                <input
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="metadata-movie-mode"
                  value="chain"
                  :checked="metadataMovieModeUi === 'chain'"
                  :disabled="metadataMovieSaving || metadataMovieChainSaving || !canUseMetadataChainMode"
                  @change="onMetadataMovieModeChain"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.metadataMovieProviderChain") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.metadataMovieProviderChainHint") }}
                  </span>
                </span>
              </label>
            </fieldset>

            <!-- Single Provider Selection -->
            <div
              v-if="metadataMovieModeUi === 'specified' && canPickSpecifiedMetadata"
              class="flex flex-col gap-2 rounded-2xl border border-border/70 bg-muted/20 p-4"
            >
              <p class="text-sm font-medium">{{ t("settings.metadataMovieProviderSelectLabel") }}</p>
              <div class="flex flex-wrap items-start gap-2">
                <Select
                  class="min-w-0 flex-1"
                  :model-value="metadataMovieProvider || metadataMovieSelectOptions[0] || ''"
                  :disabled="metadataMovieSaving"
                  @update:model-value="onMetadataMovieSelect"
                >
                  <SelectTrigger class="w-full max-w-md rounded-2xl">
                    <SelectValue :placeholder="t('settings.metadataMovieProviderSelectPh')" />
                  </SelectTrigger>
                  <SelectContent class="rounded-xl border-border/70">
                    <SelectItem
                      v-for="p in metadataMovieSelectOptions"
                      :key="p"
                      class="rounded-lg"
                      :value="p"
                    >
                      {{ p }}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  v-if="useWebApi"
                  type="button"
                  variant="outline"
                  size="icon"
                  class="size-10 shrink-0 rounded-xl"
                  :disabled="
                    providerPingAllBusy ||
                    providerPingOneName != null ||
                    !(metadataMovieProvider || metadataMovieSelectOptions[0])
                  "
                  :aria-label="t('settings.providerHealthPingCurrentAria')"
                  @click="
                    pingOneMetadataProvider(
                      metadataMovieProvider || metadataMovieSelectOptions[0] || '',
                    )
                  "
                >
                  <Activity
                    class="size-4"
                    :class="{
                      'motion-safe:animate-pulse':
                        providerPingOneName ===
                        (metadataMovieProvider || metadataMovieSelectOptions[0] || ''),
                    }"
                  />
                </Button>
              </div>
              <div
                v-if="useWebApi && healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')"
                class="flex flex-wrap items-center gap-2"
              >
                <Badge
                  variant="outline"
                  class="text-xs font-normal"
                  :class="
                    providerHealthStatusClass(
                      healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')!.status,
                    )
                  "
                >
                  {{
                    providerHealthStatusLabel(
                      healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')!.status,
                    )
                  }}
                  ·
                  {{
                    healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')!.latencyMs
                  }}ms
                </Badge>
                <span
                  v-if="healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')?.message"
                  class="text-xs text-muted-foreground"
                >
                  {{ healthForProvider(metadataMovieProvider || metadataMovieSelectOptions[0] || '')?.message }}
                </span>
              </div>
            </div>

            <!-- Provider Chain Management（与 canPickSpecifiedMetadata 解耦：无站点列表时仍显示已保存的链） -->
            <div
              v-if="metadataMovieModeUi === 'chain'"
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/20 p-4"
            >
              <p
                v-if="!canPickSpecifiedMetadata"
                class="rounded-xl border border-amber-500/35 bg-amber-500/10 px-3 py-2 text-sm text-amber-200/95"
              >
                {{ t("settings.metadataMovieProviderChainNoList") }}
              </p>
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium">{{ t("settings.metadataMovieProviderChainLabel") }}</p>
                <span class="text-xs text-muted-foreground">
                  {{ providerChainDraft.length }} {{ t("settings.providersSelected") }}
                </span>
              </div>

              <!-- Provider Chain List -->
              <div class="flex flex-col gap-2">
                <div
                  v-for="(provider, index) in providerChainDraft"
                  :key="provider + index"
                  class="flex flex-wrap items-center gap-2 rounded-xl border border-border/60 bg-background/50 px-3 py-2"
                >
                  <GripVertical class="size-4 shrink-0 text-muted-foreground" />
                  <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ provider }}</span>
                  <div
                    v-if="useWebApi && healthForProvider(provider)"
                    class="flex min-w-0 flex-wrap items-center gap-1.5"
                  >
                    <Badge
                      variant="outline"
                      class="text-[0.65rem] font-normal"
                      :class="providerHealthStatusClass(healthForProvider(provider)!.status)"
                    >
                      {{ providerHealthStatusLabel(healthForProvider(provider)!.status) }}
                      · {{ healthForProvider(provider)!.latencyMs }}ms
                    </Badge>
                  </div>
                  <Button
                    v-if="useWebApi"
                    type="button"
                    variant="outline"
                    size="icon"
                    class="size-8 shrink-0 rounded-lg"
                    :disabled="providerPingAllBusy || providerPingOneName === provider"
                    :aria-label="t('settings.providerHealthPingOneAria', { name: provider })"
                    @click="pingOneMetadataProvider(provider)"
                  >
                    <Activity
                      class="size-4"
                      :class="{ 'motion-safe:animate-pulse': providerPingOneName === provider }"
                    />
                  </Button>
                  <div class="flex shrink-0 items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      class="size-7 rounded-lg"
                      :disabled="index === 0"
                      @click="moveProviderInChain(index, 'up')"
                    >
                      <span class="sr-only">{{ t("common.moveUp") }}</span>
                      <span class="text-xs">↑</span>
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      class="size-7 rounded-lg"
                      :disabled="index === providerChainDraft.length - 1"
                      @click="moveProviderInChain(index, 'down')"
                    >
                      <span class="sr-only">{{ t("common.moveDown") }}</span>
                      <span class="text-xs">↓</span>
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      class="size-7 rounded-lg text-destructive hover:text-destructive"
                      @click="removeProviderFromChain(index)"
                    >
                      <X class="size-4" />
                    </Button>
                  </div>
                </div>

                <!-- Empty State -->
                <div
                  v-if="providerChainDraft.length === 0"
                  class="rounded-xl border border-dashed border-border/60 bg-background/30 px-3 py-6 text-center text-sm text-muted-foreground"
                >
                  {{ t("settings.metadataMovieProviderChainEmpty") }}
                </div>
              </div>

              <!-- Add Provider -->
              <div v-if="availableProvidersForChain.length > 0" class="flex items-center gap-2 pt-2">
                <Select v-model="selectedProviderToAdd">
                  <SelectTrigger class="h-9 flex-1 rounded-xl text-sm">
                    <SelectValue :placeholder="t('settings.selectProviderToAdd')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="p in availableProvidersForChain"
                      :key="p"
                      :value="p"
                    >
                      {{ p }}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-9 rounded-xl px-3"
                  :disabled="!selectedProviderToAdd"
                  @click="addProviderToChain"
                >
                  <Plus class="mr-1 size-4" />
                  {{ t("common.add") }}
                </Button>
              </div>

              <!-- Manual save：与自动保存相同；无变更时为 no-op -->
              <div class="flex flex-col gap-2 pt-2">
                <p class="text-xs text-muted-foreground">
                  {{ t("settings.metadataMovieProviderChainAutoSave") }}
                </p>
                <Button
                  type="button"
                  variant="secondary"
                  class="w-fit rounded-xl"
                  :disabled="metadataMovieChainSaving"
                  @click="saveProviderChain"
                >
                  {{ metadataMovieChainSaving ? t("common.saving") : t("common.save") }}
                </Button>
              </div>

              <p
                v-if="metadataMovieChainSaving"
                class="text-xs text-muted-foreground motion-safe:animate-pulse"
              >
                {{ t("settings.metadataMovieProviderSyncing") }}
              </p>
              <p v-if="metadataMovieChainError" class="text-sm text-destructive">
                {{ metadataMovieChainError }}
              </p>
            </div>

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

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">{{ t("settings.triggerScrape") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.triggerScrapeHint") }}
                </p>
              </div>
              <Button type="button" class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                {{ t("common.run") }}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-network"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navNetwork')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navNetwork") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.proxyTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.proxyDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <p
              v-if="!useWebApi"
              class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-sm text-muted-foreground"
            >
              {{ t("settings.proxyMockHint") }}
            </p>
            <div
              class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="proxySaving"
            >
              <div class="min-w-0 flex-1">
                <p class="font-medium">{{ t("settings.proxyEnabled") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.proxyEnabledHint") }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="proxyEnabledDraft"
                :disabled="proxySaving"
                @update:model-value="proxyEnabledDraft = $event"
              />
            </div>
            <div
              v-if="proxyEnabledDraft"
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/15 p-4"
            >
              <div class="flex flex-col gap-1.5">
                <p class="text-sm font-medium">{{ t("settings.proxyUrl") }}</p>
                <Input
                  v-model="proxyUrlDraft"
                  type="url"
                  autocomplete="off"
                  class="rounded-xl border-border/70"
                  :placeholder="t('settings.proxyUrlPlaceholder')"
                  :disabled="proxySaving"
                />
              </div>
              <div class="flex flex-col gap-2">
                <button
                  type="button"
                  class="flex w-full items-center justify-between gap-2 rounded-xl border border-border/60 bg-background/30 px-3 py-2.5 text-left text-sm font-medium text-foreground transition-colors hover:bg-muted/25 disabled:opacity-60"
                  :disabled="proxySaving"
                  :aria-expanded="proxyAuthExpanded"
                  @click="proxyAuthExpanded = !proxyAuthExpanded"
                >
                  <span>{{ t("settings.proxyAuthToggle") }}</span>
                  <ChevronDown
                    class="size-4 shrink-0 text-muted-foreground transition-transform duration-200 motion-safe:transition-transform"
                    :class="proxyAuthExpanded ? 'rotate-180' : ''"
                    aria-hidden="true"
                  />
                </button>
                <div
                  v-show="proxyAuthExpanded"
                  class="flex flex-col gap-3 border-t border-border/50 pt-3"
                >
                  <div class="flex flex-col gap-1.5">
                    <p class="text-sm font-medium">{{ t("settings.proxyUsername") }}</p>
                    <Input
                      v-model="proxyUsernameDraft"
                      autocomplete="off"
                      class="rounded-xl border-border/70"
                      :disabled="proxySaving"
                    />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <p class="text-sm font-medium">{{ t("settings.proxyPassword") }}</p>
                    <Input
                      v-model="proxyPasswordDraft"
                      type="password"
                      autocomplete="new-password"
                      class="rounded-xl border-border/70"
                      :disabled="proxySaving"
                    />
                  </div>
                </div>
              </div>
            </div>
            <p
              v-if="useWebApi"
              class="text-xs text-muted-foreground"
            >
              {{ t("settings.proxyPingJavbusHint") }}
            </p>
            <p
              v-if="useWebApi"
              class="text-xs text-muted-foreground"
            >
              {{ t("settings.proxyPingGoogleHint") }}
            </p>
            <div class="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                class="rounded-xl"
                :disabled="proxySaving || proxyOutboundPingBusy"
                @click="saveProxySettings"
              >
                {{ proxySaving ? t("common.saving") : t("settings.proxySave") }}
              </Button>
              <Button
                v-if="useWebApi"
                type="button"
                variant="outline"
                class="rounded-xl"
                :disabled="proxySaving || proxyOutboundPingBusy"
                :aria-busy="proxyJavbusBusy"
                @click="testProxyJavbus"
              >
                <Loader2
                  v-if="proxyJavbusBusy"
                  class="mr-2 size-4 motion-safe:animate-spin"
                  aria-hidden="true"
                />
                {{
                  proxyJavbusBusy
                    ? t("settings.proxyPingJavbusTesting")
                    : t("settings.proxyPingJavbus")
                }}
              </Button>
              <Button
                v-if="useWebApi"
                type="button"
                variant="outline"
                class="rounded-xl"
                :disabled="proxySaving || proxyOutboundPingBusy"
                :aria-busy="proxyGoogleBusy"
                @click="testProxyGoogle"
              >
                <Loader2
                  v-if="proxyGoogleBusy"
                  class="mr-2 size-4 motion-safe:animate-spin"
                  aria-hidden="true"
                />
                {{
                  proxyGoogleBusy
                    ? t("settings.proxyPingGoogleTesting")
                    : t("settings.proxyPingGoogle")
                }}
              </Button>
            </div>
            <p
              v-if="proxyJavbusResult && proxyJavbusResultOk !== null"
              class="text-sm"
              :class="
                proxyJavbusResultOk
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-destructive'
              "
            >
              {{ proxyJavbusResult }}
            </p>
            <p
              v-if="proxyGoogleResult && proxyGoogleResultOk !== null"
              class="text-sm"
              :class="
                proxyGoogleResultOk
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-destructive'
              "
            >
              {{ proxyGoogleResult }}
            </p>
            <p
              v-if="proxySaving"
              class="text-xs text-muted-foreground motion-safe:animate-pulse"
            >
              {{ t("settings.proxySyncing") }}
            </p>
            <p v-if="proxyError" class="text-sm text-destructive">
              {{ proxyError }}
            </p>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-libraryBehavior"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navLibraryBehavior')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navLibraryBehavior") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.extendedImportTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.extendedImportDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div
              class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="extendedLibraryImportSaving"
            >
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <p class="font-medium">{{ t("settings.extendedImportSwitch") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.extendedImportHint") }}
                </p>
                <p
                  v-if="extendedLibraryImportSaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  {{ t("settings.extendedImportSyncing") }}
                </p>
              </div>
              <Switch
                class="motion-safe:transition-colors motion-safe:duration-200"
                :model-value="extendedLibraryImport"
                @update:model-value="onExtendedLibraryImportChange"
              />
            </div>
            <p v-if="extendedLibraryImportError" class="text-sm text-destructive">
              {{ extendedLibraryImportError }}
            </p>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-curated"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navCurated')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navCurated") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.curatedCardTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.curatedCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <fieldset class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <legend class="px-1 text-sm font-medium">{{ t("settings.savePolicy") }}</legend>
              <label class="flex cursor-pointer items-start gap-3 rounded-xl p-2 hover:bg-muted/40">
                <input
                  v-model="curatedSaveMode"
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="curated-save-mode"
                  value="app"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.curatedApp") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.curatedAppHint") }}
                  </span>
                </span>
              </label>
              <label class="flex cursor-pointer items-start gap-3 rounded-xl p-2 hover:bg-muted/40">
                <input
                  v-model="curatedSaveMode"
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="curated-save-mode"
                  value="download"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.curatedDownload") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.curatedDownloadHint") }}
                  </span>
                </span>
              </label>
              <label class="flex cursor-pointer items-start gap-3 rounded-xl p-2 hover:bg-muted/40">
                <input
                  v-model="curatedSaveMode"
                  class="mt-1 size-4 shrink-0 accent-primary"
                  type="radio"
                  name="curated-save-mode"
                  value="directory"
                  :disabled="!supportsFileSystemAccess()"
                />
                <span class="min-w-0 flex-1">
                  <span class="font-medium">{{ t("settings.curatedDir") }}</span>
                  <span class="mt-0.5 block text-sm text-muted-foreground">
                    {{ t("settings.curatedDirHint") }}
                  </span>
                  <span
                    v-if="!supportsFileSystemAccess()"
                    class="mt-1 block text-xs text-amber-600 dark:text-amber-500"
                  >
                    {{ t("settings.curatedDirUnsupported") }}
                  </span>
                </span>
              </label>
            </fieldset>

            <div
              v-if="curatedSaveMode === 'directory' && supportsFileSystemAccess()"
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/20 p-4"
            >
              <p class="text-sm font-medium">{{ t("settings.exportFolder") }}</p>
              <p class="text-sm text-muted-foreground">
                {{
                  curatedExportDirLabel
                    ? t("settings.exportChosen", { name: curatedExportDirLabel })
                    : t("settings.exportNone")
                }}
              </p>
              <div class="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  class="rounded-2xl"
                  :disabled="curatedExportPickBusy"
                  @click="pickCuratedExportDirectory"
                >
                  <FolderOpen data-icon="inline-start" />
                  {{ curatedExportPickBusy ? t("settings.picking") : t("settings.pickExportFolder") }}
                </Button>
                <Button
                  v-if="curatedExportDirLabel"
                  type="button"
                  variant="outline"
                  class="rounded-2xl"
                  :disabled="curatedExportPickBusy"
                  @click="clearCuratedExportDirectory"
                >
                  {{ t("settings.clearExportFolder") }}
                </Button>
              </div>
            </div>

            <p v-if="curatedExportError" class="text-sm text-destructive" role="alert">
              {{ curatedExportError }}
            </p>

            <p class="text-xs leading-relaxed text-muted-foreground">
              <ImageDown class="mr-1 inline size-3.5 align-text-bottom opacity-70" aria-hidden="true" />
              {{ t("settings.curatedCorsNote") }}
            </p>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-playback"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navPlayback')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navPlayback") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.playbackCardTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.playbackCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">{{ t("settings.hardwareDecode") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.hardwareDecodeHint") }}
                </p>
              </div>
              <Switch v-model="hardwareDecode" />
            </div>
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-maintenance"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navMaintenance')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navMaintenance") }}
    </h2>
      <div class="flex w-full flex-col gap-6">
      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.manualCardTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.manualCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">{{ t("settings.triggerFullScan") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.triggerFullScanHint") }}
                </p>
              </div>
              <Button
                type="button"
                class="rounded-2xl"
                :disabled="fullScanBusy"
                @click="runFullScan"
              >
                <ScanSearch
                  data-icon="inline-start"
                  :class="fullScanBusy ? 'animate-pulse' : ''"
                />
                {{ t("common.run") }}
              </Button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">{{ t("settings.rebuildCache") }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ t("settings.rebuildCacheHint") }}
                </p>
              </div>
              <Button variant="secondary" class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                {{ t("common.run") }}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
          <CardHeader>
            <CardTitle>{{ t("settings.configCardTitle") }}</CardTitle>
            <CardDescription>
              {{ t("settings.configCardDesc") }}
            </CardDescription>
          </CardHeader>
          <CardContent class="text-sm leading-6 text-muted-foreground">
            {{ t("settings.configCardBody") }}
          </CardContent>
        </Card>
      </div>
      </div>
    </section>

    <section
      id="settings-section-about"
      class="scroll-mt-6 space-y-6"
      :aria-label="t('settings.navAbout')"
    >
    <h2 class="px-1 text-lg font-semibold tracking-tight text-foreground">
      {{ t("settings.navAbout") }}
    </h2>
      <Card class="rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
        <CardHeader class="flex flex-row items-start gap-3 space-y-0 pb-2">
          <Info
            class="mt-0.5 size-5 shrink-0 text-muted-foreground"
            aria-hidden="true"
          />
          <div class="min-w-0 flex-1 space-y-1">
            <CardTitle>{{ t("settings.aboutCardTitle") }}</CardTitle>
            <CardDescription>{{ t("settings.aboutCardDesc") }}</CardDescription>
          </div>
        </CardHeader>
        <CardContent class="space-y-3 text-sm leading-6 text-muted-foreground">
          <p>{{ t("settings.aboutIntro") }}</p>
          <div
            class="rounded-2xl border border-border/70 bg-background/50 px-4 py-3 text-muted-foreground"
            role="status"
          >
            <p class="font-medium text-foreground">
              {{ t("settings.aboutDataModeLabel") }}
            </p>
            <p class="mt-1.5">
              {{
                useWebApi
                  ? t("settings.aboutDataModeWebApi")
                  : t("settings.aboutDataModeMock")
              }}
            </p>
          </div>
          <p class="text-xs text-muted-foreground/90">
            {{ t("settings.aboutBuildEnv", { env: viteMode }) }}
          </p>
          <p v-if="isViteDev" class="text-xs text-muted-foreground/90">
            {{ t("settings.aboutDevProxyHint") }}
          </p>
        </CardContent>
      </Card>
    </section>
    </div>
  </div>
</template>
