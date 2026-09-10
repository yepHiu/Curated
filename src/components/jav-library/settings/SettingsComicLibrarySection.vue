<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { ComicCacheStatusDTO } from "@/api/types"
import type {
  ComicCacheSettings,
  ComicLibrarySetting,
  ComicReaderSettings,
} from "@/domain/comic/types"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { useComicLibraryService } from "@/services/comic-library-service"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import SettingsComicCacheSection from "./SettingsComicCacheSection.vue"
import SettingsComicLibraryPathsSection from "./SettingsComicLibraryPathsSection.vue"
import SettingsComicReaderSection from "./SettingsComicReaderSection.vue"

const { t } = useI18n()
const comicService = useComicLibraryService()

const comicLibraryEnabled = computed(() => comicService.comicLibraryEnabled.value)
const autoComicLibraryWatch = computed(() => comicService.autoComicLibraryWatch.value)
const comicLibraryPaths = computed(() => comicService.comicLibraryPaths.value)
const defaultComicImportLibraryPathId = computed(
  () => comicService.defaultComicImportLibraryPathId.value,
)
const comicReader = computed(() => comicService.comicReader.value)
const comicCache = computed(() => comicService.comicCache.value)

const enableBusy = ref(false)
const autoWatchBusy = ref(false)
const pathBusy = ref(false)
const pathScanBusy = ref<string | null>(null)
const defaultPathBusy = ref(false)
const readerBusy = ref(false)
const cacheBusy = ref(false)
const cacheCleanupBusy = ref(false)
const enableError = ref("")
const autoWatchError = ref("")
const pathError = ref("")
const readerError = ref("")
const cacheError = ref("")
const comicCacheStatus = ref<ComicCacheStatusDTO | null>(null)
const addPathDialogOpen = ref(false)
const newPath = ref("")
const newPathTitle = ref("")
const directoryHint = ref("")
const pickDirectoryBusy = ref(false)

const canSaveNewPath = computed(() => {
  const trimmed = newPath.value.trim()
  return trimmed.length > 0 && isAbsoluteLibraryPath(trimmed)
})

const directoryHintDisplay = computed(() => {
  const hint = directoryHint.value.trim()
  if (!hint) return ""
  if (!canSaveNewPath.value) {
    return `${hint}\n\n${t("settings.pickFolderHintSaveSuffix")}`
  }
  return hint
})

function errorMessage(err: unknown, fallbackKey = "settings.errSaveTitle"): string {
  if (err instanceof HttpClientError && err.apiError?.message) {
    return err.apiError.message
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return t(fallbackKey)
}

onMounted(() => {
  void comicService.refreshSettings().catch((err) => {
    enableError.value = errorMessage(err)
  })
})

watch(addPathDialogOpen, (open) => {
  if (!open) {
    newPath.value = ""
    newPathTitle.value = ""
    pathError.value = ""
    directoryHint.value = ""
  }
})

async function enableComicLibrary() {
  enableError.value = ""
  if (comicLibraryPaths.value.length === 0) {
    enableError.value = t("settings.comicLibraryPathRequired")
    return
  }
  try {
    enableBusy.value = true
    await comicService.setComicLibraryEnabled(true)
  } catch (err) {
    enableError.value = errorMessage(err)
  } finally {
    enableBusy.value = false
  }
}

async function disableComicLibrary() {
  enableError.value = ""
  try {
    enableBusy.value = true
    await comicService.setComicLibraryEnabled(false)
  } catch (err) {
    enableError.value = errorMessage(err)
  } finally {
    enableBusy.value = false
  }
}

async function changeAutoComicLibraryWatch(value: boolean) {
  autoWatchError.value = ""
  try {
    autoWatchBusy.value = true
    await comicService.setAutoComicLibraryWatch(value)
  } catch (err) {
    autoWatchError.value = errorMessage(err)
  } finally {
    autoWatchBusy.value = false
  }
}

function clearPathAddError() {
  pathError.value = ""
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
      document.getElementById("new-comic-lib-path")?.focus()
    }
  } finally {
    pickDirectoryBusy.value = false
  }
}

async function submitAddPath() {
  pathError.value = ""
  const trimmed = newPath.value.trim()
  if (!trimmed) {
    pathError.value = t("settings.comicLibraryPathRequired")
    return
  }
  if (!isAbsoluteLibraryPath(trimmed)) {
    pathError.value = t("settings.comicLibraryPathAbsoluteRequired")
    return
  }
  try {
    pathBusy.value = true
    await comicService.addComicLibraryPath(trimmed, newPathTitle.value.trim() || undefined)
    await comicService.refreshSettings()
    newPath.value = ""
    newPathTitle.value = ""
    addPathDialogOpen.value = false
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    pathBusy.value = false
  }
}

async function removeComicPath(id: string) {
  pathError.value = ""
  try {
    pathBusy.value = true
    await comicService.removeComicLibraryPath(id)
    await comicService.refreshSettings()
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    pathBusy.value = false
  }
}

async function scanComicPath(path: ComicLibrarySetting) {
  pathError.value = ""
  const target = path.path.trim()
  if (!target) return
  try {
    pathScanBusy.value = target
    await comicService.scanComics([target])
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    pathScanBusy.value = null
  }
}

async function changeDefaultPath(id: string) {
  pathError.value = ""
  try {
    defaultPathBusy.value = true
    await comicService.setDefaultComicImportLibraryPathId(id)
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    defaultPathBusy.value = false
  }
}

async function patchReader(patch: Partial<ComicReaderSettings>) {
  readerError.value = ""
  try {
    readerBusy.value = true
    await comicService.patchComicReader(patch)
  } catch (err) {
    readerError.value = errorMessage(err)
  } finally {
    readerBusy.value = false
  }
}

async function patchCache(maxBytes: number) {
  cacheError.value = ""
  const patch: Partial<ComicCacheSettings> = { maxBytes }
  try {
    cacheBusy.value = true
    await comicService.patchComicCache(patch)
  } catch (err) {
    cacheError.value = errorMessage(err)
  } finally {
    cacheBusy.value = false
  }
}

async function cleanupCache() {
  cacheError.value = ""
  try {
    cacheCleanupBusy.value = true
    comicCacheStatus.value = await comicService.cleanupComicCache()
  } catch (err) {
    cacheError.value = errorMessage(err)
  } finally {
    cacheCleanupBusy.value = false
  }
}
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <BookOpen class="size-4" />
        </span>
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.comicLibraryTitle") }}
        </CardTitle>
        <CardDescription
          class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
        >
          {{ t("settings.comicLibraryDesc") }}
        </CardDescription>
      </CardHeader>

      <CardContent class="flex flex-col gap-3 pt-0">
        <div
          class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
          :aria-busy="enableBusy"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <p class="text-sm font-semibold text-foreground">
              {{
                comicLibraryEnabled
                  ? t("settings.comicLibraryEnabledTitle")
                  : t("settings.comicLibraryDisabledTitle")
              }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{
                comicLibraryEnabled
                  ? t("settings.comicLibraryEnabledDesc")
                  : t("settings.comicLibraryDisabledDesc")
              }}
            </p>
            <p
              v-if="enableBusy"
              class="text-xs text-muted-foreground motion-safe:animate-pulse"
            >
              {{ t("settings.comicLibrarySyncing") }}
            </p>
          </div>
          <Button
            v-if="comicLibraryEnabled"
            data-comic-disable
            type="button"
            size="sm"
            variant="outline"
            class="h-8 shrink-0"
            :disabled="enableBusy"
            @click="disableComicLibrary"
          >
            {{ t("settings.comicLibraryDisable") }}
          </Button>
          <Button
            v-else
            data-comic-enable
            type="button"
            size="sm"
            class="h-8 shrink-0"
            :disabled="enableBusy"
            @click="enableComicLibrary"
          >
            {{ t("settings.comicLibraryEnable") }}
          </Button>
        </div>

        <p v-if="enableError" class="text-sm text-destructive">{{ enableError }}</p>

        <SettingsComicLibraryPathsSection
          :paths="comicLibraryPaths"
          :default-import-library-path-id="defaultComicImportLibraryPathId"
          v-model:add-path-dialog-open="addPathDialogOpen"
          v-model:new-path="newPath"
          v-model:new-path-title="newPathTitle"
          :pick-directory-busy="pickDirectoryBusy"
          :directory-hint-display="directoryHintDisplay"
          :add-busy="pathBusy"
          :can-save-new-path="canSaveNewPath"
          :default-saving="defaultPathBusy"
          :scan-path-busy="pathScanBusy"
          :path-add-error="pathError"
          dialog-content-class="rounded-3xl border-border/50 sm:max-w-md"
          @clear-error="clearPathAddError"
          @browse="browseForDirectory"
          @submit="submitAddPath"
          @scan-path="scanComicPath"
          @remove-path="removeComicPath"
          @change-default-import-path="changeDefaultPath"
        />

        <template v-if="comicLibraryEnabled">
          <div
            data-comic-auto-watch
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
            :aria-busy="autoWatchBusy"
          >
            <div class="flex min-w-0 flex-col gap-1">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.comicAutoWatchTitle") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.comicAutoWatchDesc") }}
              </p>
              <p
                v-if="autoWatchBusy"
                class="text-xs text-muted-foreground motion-safe:animate-pulse"
              >
                {{ t("settings.comicAutoWatchSyncing") }}
              </p>
            </div>
            <Switch
              data-comic-auto-watch-switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              :model-value="autoComicLibraryWatch"
              :disabled="autoWatchBusy"
              :aria-label="t('settings.comicAutoWatchTitle')"
              @update:model-value="changeAutoComicLibraryWatch"
            />
          </div>
          <p v-if="autoWatchError" class="text-sm text-destructive">{{ autoWatchError }}</p>

          <SettingsComicReaderSection
            :reader="comicReader"
            :saving="readerBusy"
            :error="readerError"
            @patch-reader="patchReader"
          />
          <SettingsComicCacheSection
            :max-bytes="comicCache.maxBytes"
            :status="comicCacheStatus"
            :saving="cacheBusy"
            :cleanup-busy="cacheCleanupBusy"
            :error="cacheError"
            @change-max-bytes="patchCache"
            @cleanup="cleanupCache"
          />
        </template>
      </CardContent>
    </Card>
  </div>
</template>
