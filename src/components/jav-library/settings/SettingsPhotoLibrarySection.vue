<script setup lang="ts">
import { useServerLocalAccess } from "@/composables/use-server-local-access"
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { FolderOpen } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { PhotoCacheSettings, PhotoLibrarySetting, PhotoViewerSettings } from "@/domain/photo/types"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { usePhotoLibraryService } from "@/services/photo-library-service"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import SettingsPhotoCacheSection from "./SettingsPhotoCacheSection.vue"
import SettingsPhotoLibraryPathsSection from "./SettingsPhotoLibraryPathsSection.vue"
import SettingsPhotoViewerSection from "./SettingsPhotoViewerSection.vue"

const { t } = useI18n()
const { isServerLocal } = useServerLocalAccess()
const photoService = usePhotoLibraryService()

const photoLibraryEnabled = computed(() => photoService.photoLibraryEnabled.value)
const autoPhotoLibraryWatch = computed(() => photoService.autoPhotoLibraryWatch.value)
const photoLibraryPaths = computed(() => photoService.photoLibraryPaths.value)
const defaultPhotoImportLibraryPathId = computed(
  () => photoService.defaultPhotoImportLibraryPathId.value,
)
const photoViewer = computed(() => photoService.photoViewer.value)
const photoCache = computed(() => photoService.photoCache.value)

const enableBusy = ref(false)
const autoWatchBusy = ref(false)
const pathBusy = ref(false)
const pathScanBusy = ref<string | null>(null)
const defaultPathBusy = ref(false)
const viewerBusy = ref(false)
const cacheBusy = ref(false)
const enableError = ref("")
const autoWatchError = ref("")
const pathError = ref("")
const viewerError = ref("")
const cacheError = ref("")
const addPathDialogOpen = ref(false)
const newPath = ref("")
const newPathTitle = ref("")
const directoryHint = ref("")
const pickDirectoryBusy = ref(false)

function errorMessage(err: unknown): string {
  if (err instanceof HttpClientError && err.apiError?.message) {
    return err.apiError.message
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return t("settings.errSaveTitle")
}

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

onMounted(() => {
  void photoService.refreshSettings().catch((err) => {
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

/** 持久化写真 Beta 开关；保存失败时保留当前状态。 */
async function setPhotoLibraryEnabled(value: boolean) {
  if (enableBusy.value) return
  enableError.value = ""
  try {
    enableBusy.value = true
    await photoService.setPhotoLibraryEnabled(value)
  } catch (err) {
    enableError.value = errorMessage(err)
  } finally {
    enableBusy.value = false
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
      document.getElementById("new-photo-lib-path")?.focus()
    }
  } finally {
    pickDirectoryBusy.value = false
  }
}

async function submitAddPath() {
  pathError.value = ""
  const trimmed = newPath.value.trim()
  if (!trimmed) {
    pathError.value = t("settings.photoLibraryPathRequired")
    return
  }
  if (!isAbsoluteLibraryPath(trimmed)) {
    pathError.value = t("settings.photoLibraryPathAbsoluteRequired")
    return
  }
  try {
    pathBusy.value = true
    await photoService.addPhotoLibraryPath(trimmed, newPathTitle.value.trim() || undefined)
    await photoService.refreshSettings()
    newPath.value = ""
    newPathTitle.value = ""
    addPathDialogOpen.value = false
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    pathBusy.value = false
  }
}

async function removePhotoPath(id: string) {
  pathError.value = ""
  try {
    pathBusy.value = true
    await photoService.removePhotoLibraryPath(id)
    await photoService.refreshSettings()
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    pathBusy.value = false
  }
}

async function scanPhotoPath(path: PhotoLibrarySetting) {
  pathError.value = ""
  const target = path.path.trim()
  if (!target) return
  try {
    pathScanBusy.value = target
    await photoService.scanPhotos([target])
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
    await photoService.setDefaultPhotoImportLibraryPathId(id)
  } catch (err) {
    pathError.value = errorMessage(err)
  } finally {
    defaultPathBusy.value = false
  }
}

async function changeAutoPhotoLibraryWatch(value: boolean) {
  autoWatchError.value = ""
  try {
    autoWatchBusy.value = true
    await photoService.setAutoPhotoLibraryWatch(value)
  } catch (err) {
    autoWatchError.value = errorMessage(err)
  } finally {
    autoWatchBusy.value = false
  }
}

async function patchViewer(patch: Partial<PhotoViewerSettings>) {
  viewerError.value = ""
  try {
    viewerBusy.value = true
    await photoService.patchPhotoViewer(patch)
  } catch (err) {
    viewerError.value = errorMessage(err)
  } finally {
    viewerBusy.value = false
  }
}

async function patchCache(maxBytes: number) {
  cacheError.value = ""
  const patch: Partial<PhotoCacheSettings> = { maxBytes }
  try {
    cacheBusy.value = true
    await photoService.patchPhotoCache(patch)
  } catch (err) {
    cacheError.value = errorMessage(err)
  } finally {
    cacheBusy.value = false
  }
}

</script>

<template>
  <div class="flex flex-col gap-4">
    <section
      class="rounded-xl border border-border bg-card p-6 shadow-sm"
      data-photo-enable-section
      :aria-busy="enableBusy"
    >
      <FieldGroup>
        <Field orientation="horizontal" class="items-center justify-between">
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <SettingsHint :text="photoLibraryEnabled ? t('settings.photoLibraryEnabledDesc') : t('settings.photoLibraryDisabledDesc')">
              <FieldLabel for="photo-library-enabled" class="text-lg font-semibold">
                {{ t("settings.photoLibraryEnable") }}
              </FieldLabel>
            </SettingsHint>
            <SettingsScopeBadge scope="server" />
            <Badge variant="secondary">Beta</Badge>
          </div>
          <Switch
            v-if="isServerLocal"
            id="photo-library-enabled"
            data-photo-beta-switch
            :model-value="photoLibraryEnabled"
            :disabled="enableBusy"
            :aria-label="t('settings.photoLibraryEnable')"
            @update:model-value="setPhotoLibraryEnabled"
          />
        </Field>
      </FieldGroup>
      <p v-if="enableBusy" class="mt-3 text-xs text-muted-foreground motion-safe:animate-pulse">
        {{ t("settings.photoLibrarySyncing") }}
      </p>
      <p v-if="enableError" role="alert" class="mt-3 text-sm text-destructive">{{ enableError }}</p>
    </section>

    <Card v-if="photoLibraryEnabled" data-photo-storage-settings class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <FolderOpen class="size-4" />
        </span>
        <CardTitle class="flex min-w-0 flex-wrap items-center gap-2 text-lg tracking-tight">
          <span>{{ t("settings.storagePathSettings") }}</span>
          <SettingsScopeBadge scope="server" />
        </CardTitle>
      </CardHeader>
      <CardContent class="flex flex-col gap-3 pt-0">
        <SettingsPhotoLibraryPathsSection
          :paths="photoLibraryPaths"
          :default-import-library-path-id="defaultPhotoImportLibraryPathId"
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
          @scan-path="scanPhotoPath"
          @remove-path="removePhotoPath"
          @change-default-import-path="changeDefaultPath"
        />

        <template v-if="isServerLocal">
          <div
            data-photo-auto-watch
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
            :aria-busy="autoWatchBusy"
          >
            <div class="flex min-w-0 flex-col gap-1">
              <SettingsHint :text="t('settings.photoAutoWatchDesc')">
                <p class="text-sm font-semibold text-foreground">
                  {{ t("settings.photoAutoWatchTitle") }}
                </p>
              </SettingsHint>
              <p
                v-if="autoWatchBusy"
                class="text-xs text-muted-foreground motion-safe:animate-pulse"
              >
                {{ t("settings.photoAutoWatchSyncing") }}
              </p>
            </div>
            <Switch
              data-photo-auto-watch-switch
              class="motion-safe:transition-colors motion-safe:duration-200"
              :model-value="autoPhotoLibraryWatch"
              :disabled="autoWatchBusy"
              :aria-label="t('settings.photoAutoWatchTitle')"
              @update:model-value="changeAutoPhotoLibraryWatch"
            />
          </div>
          <p v-if="autoWatchError" class="text-sm text-destructive">{{ autoWatchError }}</p>
        </template>
      </CardContent>
    </Card>

    <template v-if="photoLibraryEnabled && isServerLocal">
      <SettingsPhotoViewerSection
        :viewer="photoViewer"
        :saving="viewerBusy"
        :error="viewerError"
        @patch-viewer="patchViewer"
      />
      <SettingsPhotoCacheSection
        :max-bytes="photoCache.maxBytes"
        :saving="cacheBusy"
        :error="cacheError"
        @change-max-bytes="patchCache"
      />
    </template>
  </div>
</template>
