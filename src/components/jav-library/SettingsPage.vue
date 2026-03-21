<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { HttpClientError } from "@/api/http-client"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import {
  FolderOpen,
  FolderPlus,
  Pencil,
  RefreshCw,
  ScanSearch,
  Sparkles,
  Trash2,
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
import { Switch } from "@/components/ui/switch"
import { useLibraryService } from "@/services/library-service"

const libraryService = useLibraryService()
const scanTaskTracker = useScanTaskTracker()
/** Plain object services don't unwrap nested ComputedRefs in templates */
const libraryPathsList = computed(() => libraryService.libraryPaths.value)
const hardwareDecode = ref(true)
const autoScrape = ref(true)

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

const organizeLibrary = computed(() => libraryService.organizeLibrary.value)

const dashboardStats = computed(() => libraryService.libraryStats.value)

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

onMounted(() => {
  void libraryService.refreshSettings()
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
  editTitleBusy.value = true
  try {
    await libraryService.updateLibraryPathTitle(id, editLibraryTitleDraft.value)
    cancelEditLibraryTitle()
  } catch (err) {
    console.error("[settings] update library title failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      editTitleError.value = err.apiError.message
    } else {
      editTitleError.value = "保存失败，请稍后重试。"
    }
  } finally {
    editTitleBusy.value = false
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
      return
    }
    if (outcome.status === "unsupported") {
      directoryHint.value =
        "当前环境无法打开目录选择器，请手动粘贴绝对路径。集成桌面客户端后可使用系统原生文件夹对话框并自动填入路径。"
    }
  } finally {
    pickDirectoryBusy.value = false
  }
}

async function submitAddPath() {
  pathAddError.value = ""
  const trimmed = newPath.value.trim()
  if (!isAbsoluteLibraryPath(trimmed)) {
    pathAddError.value =
      "请填写绝对路径：Windows 如 D:\\Media\\JAV；macOS/Linux 如 /Users/you/Videos/JAV；不要用相对路径如 folder\\sub。"
    return
  }
  addBusy.value = true
  try {
    await libraryService.addLibraryPath(newPath.value, newPathTitle.value || undefined)
    newPath.value = ""
    newPathTitle.value = ""
    addPathDialogOpen.value = false
  } catch (err) {
    console.error("[settings] add library path failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      pathAddError.value = err.apiError.message
    }
  } finally {
    addBusy.value = false
  }
}

async function removePath(id: string) {
  try {
    await libraryService.removeLibraryPath(id)
  } catch (err) {
    console.error("[settings] remove library path failed", err)
  }
}

async function rescanPath(path: string) {
  scanFeedbackError.value = ""
  scanPathBusy.value = path
  try {
    const task = await libraryService.scanLibraryPaths([path])
    if (task?.taskId) {
      scanTaskTracker.start(task.taskId)
    }
  } catch (err) {
    console.error("[settings] rescan path failed", err)
    if (err instanceof HttpClientError && err.status === 409) {
      scanFeedbackError.value = "已有扫描正在进行，请等待当前扫描结束后再试。"
    } else if (err instanceof HttpClientError && err.apiError?.message) {
      scanFeedbackError.value = err.apiError.message
    } else {
      scanFeedbackError.value = "扫描启动失败，请稍后重试。"
    }
  } finally {
    scanPathBusy.value = null
  }
}

async function onOrganizeLibraryChange(next: boolean) {
  organizeLibraryError.value = ""
  organizeLibrarySaving.value = true
  try {
    await libraryService.setOrganizeLibrary(next)
  } catch (err) {
    console.error("[settings] organize library toggle failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      organizeLibraryError.value = err.apiError.message
    } else {
      organizeLibraryError.value = "保存失败，请稍后重试。"
    }
  } finally {
    organizeLibrarySaving.value = false
  }
}

async function runFullScan() {
  scanFeedbackError.value = ""
  fullScanBusy.value = true
  try {
    const task = await libraryService.scanLibraryPaths()
    if (task?.taskId) {
      scanTaskTracker.start(task.taskId)
    }
  } catch (err) {
    console.error("[settings] full scan failed", err)
    if (err instanceof HttpClientError && err.status === 409) {
      scanFeedbackError.value = "已有扫描正在进行，请等待当前扫描结束后再试。"
    } else if (err instanceof HttpClientError && err.apiError?.message) {
      scanFeedbackError.value = err.apiError.message
    } else {
      scanFeedbackError.value = "扫描启动失败，请稍后重试。"
    }
  } finally {
    fullScanBusy.value = false
  }
}

async function runMetadataRefreshForSelected() {
  metadataRefreshSuccess.value = ""
  metadataRefreshError.value = ""
  const paths = selectedMetadataRefreshPaths.value
  if (paths.length === 0) {
    metadataRefreshError.value = "请勾选一个或多个存储目录。"
    return
  }
  metadataRefreshBusy.value = true
  try {
    const dto = await libraryService.refreshMetadataForLibraryPaths(paths)
    const parts: string[] = [
      `已为 ${dto.queued} 部影片排队重新刮削元数据（后台执行，与详情页「刷新元数据」相同流水线）。`,
    ]
    if (dto.skipped > 0) {
      parts.push(`跳过 ${dto.skipped} 条（读取详情或入队失败）。`)
    }
    if (dto.invalidPaths.length > 0) {
      parts.push(`以下路径未在已配置库目录中匹配：${dto.invalidPaths.join("；")}`)
    }
    metadataRefreshSuccess.value = parts.join(" ")
  } catch (err) {
    console.error("[settings] metadata refresh by paths failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      metadataRefreshError.value = err.apiError.message
    } else {
      metadataRefreshError.value = "批量刷新元数据失败，请确认后端已启动并重试。"
    }
  } finally {
    metadataRefreshBusy.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex max-w-[56rem] flex-col gap-6 pb-2">
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <Card
        v-for="stat in dashboardStats"
        :key="stat.label"
        class="rounded-3xl border-border/70 bg-card/85"
      >
        <CardHeader class="gap-1">
          <CardDescription>{{ stat.label }}</CardDescription>
          <CardTitle class="text-2xl">{{ stat.value }}</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="text-sm text-muted-foreground">{{ stat.detail }}</p>
        </CardContent>
      </Card>
    </div>

    <div class="flex flex-col gap-2 px-1">
      <div class="flex flex-col gap-1">
        <h2 class="text-3xl font-semibold tracking-tight">Library settings</h2>
        <p class="text-sm text-muted-foreground">
          存储路径、播放与元数据相关选项；扫描仅通过手动触发（全库或单路径）。
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

    <div class="w-full columns-1 gap-6">
      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
          <CardHeader>
            <CardTitle>Storage directories</CardTitle>
            <CardDescription>
              可配置多个库根目录。勾选目录后点击「刷新元数据」仅对已入库条目重新刮削（不重新扫盘）；「Rescan」会遍历磁盘并索引新文件。
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Library paths</p>
                <p class="text-sm text-muted-foreground">
                  仅支持<strong class="text-foreground">绝对路径</strong>作为视频来源目录；可添加多个路径。示例：
                  <span class="font-mono text-xs">D:\Media\JAV</span> 或
                  <span class="font-mono text-xs">/home/user/Videos</span>。
                </p>
              </div>
              <Dialog v-model:open="addPathDialogOpen">
                <DialogTrigger as-child>
                  <Button type="button" class="rounded-2xl">
                    <FolderPlus data-icon="inline-start" />
                    Add path
                  </Button>
                </DialogTrigger>

                <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>添加存储目录</DialogTitle>
                    <DialogDescription>
                      填写<strong>绝对路径</strong>作为视频来源；可点击「选择文件夹」打开系统目录界面（网页内通常需再复制地址栏路径）。
                      示例：
                      <span class="font-mono text-xs">D:\Media\JAV</span> 或
                      <span class="font-mono text-xs">/home/user/Videos</span>。
                    </DialogDescription>
                  </DialogHeader>

                  <div class="flex flex-col gap-4">
                    <div class="flex flex-col gap-2">
                      <label class="text-sm font-medium" for="new-lib-path">绝对路径</label>
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
                          {{ pickDirectoryBusy ? "选择中…" : "选择文件夹" }}
                        </Button>
                      </div>
                      <p
                        v-if="directoryHint"
                        class="text-sm leading-relaxed text-muted-foreground"
                      >
                        {{ directoryHint }}
                      </p>
                      <p
                        v-if="newPath.trim() && !isAbsoluteLibraryPath(newPath)"
                        class="text-sm text-destructive"
                      >
                        当前输入不是绝对路径，请从盘符（Windows）或根目录
                        <span class="font-mono">/</span>（Unix）开始填写。
                      </p>
                      <p v-if="pathAddError" class="text-sm text-destructive">
                        {{ pathAddError }}
                      </p>
                    </div>
                    <div class="flex flex-col gap-2">
                      <label class="text-sm font-medium" for="new-lib-title">标题（可选）</label>
                      <Input
                        id="new-lib-title"
                        v-model="newPathTitle"
                        class="rounded-xl"
                        placeholder="显示名称"
                        autocomplete="off"
                      />
                    </div>
                  </div>

                  <DialogFooter>
                    <DialogClose as-child>
                      <Button type="button" variant="outline" class="rounded-2xl">
                        取消
                      </Button>
                    </DialogClose>
                    <Button
                      type="button"
                      class="rounded-2xl"
                      :disabled="addBusy || !canSaveNewPath"
                      @click="submitAddPath"
                    >
                      {{ addBusy ? "保存中…" : "保存路径" }}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>

            <div
              class="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/20 p-4 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
            >
              <div class="flex min-w-0 flex-col gap-1">
                <p class="text-sm font-medium">按目录刷新元数据</p>
                <p class="text-xs text-muted-foreground">
                  在下方列表勾选库根目录后提交；仅对已入库条目排队刮削，不遍历磁盘。新文件请先使用 Rescan。
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
                  全选目录
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="rounded-xl"
                  @click="clearMetadataPathSelection"
                >
                  清空选择
                </Button>
                <Button
                  type="button"
                  class="rounded-xl"
                  :disabled="!hasMetadataPathSelection || metadataRefreshBusy"
                  @click="runMetadataRefreshForSelected"
                >
                  <Sparkles data-icon="inline-start" class="size-4" />
                  {{ metadataRefreshBusy ? "提交中…" : "刷新元数据" }}
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
                      <p class="text-xs font-medium text-muted-foreground">路径（只读）</p>
                      <p class="break-all font-mono text-sm text-muted-foreground">{{ path.path }}</p>
                    </div>
                    <div class="flex flex-col gap-2">
                      <label class="text-sm font-medium" :for="`edit-title-${path.id}`">显示标题</label>
                      <Input
                        :id="`edit-title-${path.id}`"
                        v-model="editLibraryTitleDraft"
                        class="rounded-xl"
                        placeholder="显示名称"
                        autocomplete="off"
                        @keydown.enter.prevent="saveLibraryPathTitle(path.id)"
                      />
                      <p class="text-xs text-muted-foreground">
                        留空并保存将使用路径中的文件夹名作为标题。
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
                        {{ editTitleBusy ? "保存中…" : "保存标题" }}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        class="rounded-2xl"
                        :disabled="editTitleBusy"
                        @click="cancelEditLibraryTitle"
                      >
                        取消
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
                        :aria-label="`将「${path.title}」纳入批量元数据刷新`"
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
                        改标题
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
                        Rescan
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        class="rounded-2xl"
                        :aria-label="`Remove ${path.title}`"
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

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
          <CardHeader>
            <CardTitle>库目录整理</CardTitle>
            <CardDescription>
              开启后，扫描识别番号会将视频移入
              <span class="font-mono text-xs">父目录/番号/番号.扩展名</span>
              ，并把 NFO、封面与预览图写入同一番号文件夹（会移动磁盘文件，请先备份）。
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div
              class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4"
              :aria-busy="organizeLibrarySaving"
            >
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <p class="font-medium">整理入库（organizeLibrary）</p>
                <p class="text-sm text-muted-foreground">
                  关闭时仅更新数据库路径，海报等仍下载到服务端 cache 目录。开关会立即写入
                  <span class="font-mono text-xs">config/library-config.cfg</span>
                  ，重启 javd 后仍保持该值。
                </p>
                <p
                  v-if="organizeLibrarySaving"
                  class="text-xs text-muted-foreground motion-safe:animate-pulse"
                >
                  正在同步到服务端…
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

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Playback preferences</CardTitle>
            <CardDescription>
              Placeholder toggles for the future player and metadata pipeline.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Hardware decode</p>
                <p class="text-sm text-muted-foreground">
                  A placeholder switch for future player preferences.
                </p>
              </div>
              <Switch v-model="hardwareDecode" />
            </div>

            <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Auto scrape metadata</p>
                <p class="text-sm text-muted-foreground">
                  Keep newly discovered titles enriched after each scan.
                </p>
              </div>
              <Switch v-model="autoScrape" />
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Manual actions</CardTitle>
            <CardDescription>
              Control points reserved for scan and metadata workflows.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3">
            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Trigger metadata scrape</p>
                <p class="text-sm text-muted-foreground">
                  Fetch artwork, cast, tags, and summary metadata for indexed titles.
                </p>
              </div>
              <Button class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                Run
              </Button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Trigger full scan</p>
                <p class="text-sm text-muted-foreground">
                  Re-index all configured folders and refresh newly discovered files.
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
                Run
              </Button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-background/50 p-4">
              <div class="flex flex-col gap-1">
                <p class="font-medium">Rebuild poster cache</p>
                <p class="text-sm text-muted-foreground">
                  Regenerate poster derivatives and clear stale preview assets.
                </p>
              </div>
              <Button variant="secondary" class="rounded-2xl">
                <RefreshCw data-icon="inline-start" />
                Run
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="mb-6 break-inside-avoid">
        <Card class="rounded-3xl border-border/70 bg-card/85">
          <CardHeader>
            <CardTitle>Configuration model</CardTitle>
            <CardDescription>
              This page mirrors the document structure while staying front-end only.
            </CardDescription>
          </CardHeader>
          <CardContent class="text-sm leading-6 text-muted-foreground">
            Future desktop integration can bind these controls to typed contracts without
            rewriting the visual structure or the surrounding page shell.
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
