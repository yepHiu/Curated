<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { ComicCacheStatusDTO } from "@/api/types"
import type { ComicCacheSettings, ComicReaderSettings } from "@/domain/comic/types"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { useComicLibraryService } from "@/services/comic-library-service"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import SettingsComicCacheSection from "./SettingsComicCacheSection.vue"
import SettingsComicLibraryPathsSection from "./SettingsComicLibraryPathsSection.vue"
import SettingsComicReaderSection from "./SettingsComicReaderSection.vue"

const { t } = useI18n()
const comicService = useComicLibraryService()

const comicLibraryEnabled = computed(() => comicService.comicLibraryEnabled.value)
const comicLibraryPaths = computed(() => comicService.comicLibraryPaths.value)
const defaultComicImportLibraryPathId = computed(
  () => comicService.defaultComicImportLibraryPathId.value,
)
const comicReader = computed(() => comicService.comicReader.value)
const comicCache = computed(() => comicService.comicCache.value)

const enableBusy = ref(false)
const pathBusy = ref(false)
const defaultPathBusy = ref(false)
const readerBusy = ref(false)
const cacheBusy = ref(false)
const cacheCleanupBusy = ref(false)
const enableError = ref("")
const pathError = ref("")
const readerError = ref("")
const cacheError = ref("")
const comicCacheStatus = ref<ComicCacheStatusDTO | null>(null)

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

async function addComicPath(path: string, title: string) {
  pathError.value = ""
  const trimmed = path.trim()
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
    await comicService.addComicLibraryPath(trimmed, title.trim() || undefined)
    await comicService.refreshSettings()
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
          :add-busy="pathBusy"
          :default-saving="defaultPathBusy"
          :error="pathError"
          @add-path="addComicPath"
          @remove-path="removeComicPath"
          @change-default-import-path="changeDefaultPath"
        />

        <template v-if="comicLibraryEnabled">
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
