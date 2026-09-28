<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronDown, DatabaseBackup, ExternalLink, FolderOpen, RotateCw, ShieldCheck } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type { BackupRestorePreflightDTO, BackupVerificationDTO } from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Input } from "@/components/ui/input"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Separator } from "@/components/ui/separator"
import { pushAppToast } from "@/composables/use-app-toast"
import { buildBackupFilename, ensureBackupExtension, joinBackupDestination } from "@/lib/backup-path"
import { isAbsoluteLibraryPath } from "@/lib/path-validation"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{ supported: boolean }>()
const { t, locale } = useI18n()
const libraryService = useLibraryService()
const backupDirectoryDraft = ref("")
const backupPathDraft = ref("")
const directoryError = ref("")
const directoryPersistenceError = ref("")
const pathError = ref("")
const createError = ref("")
const restoreError = ref("")
const busyAction = ref<"pick" | "create" | "preflight" | null>(null)
const createdPath = ref("")
const createdVerification = ref<BackupVerificationDTO | null>(null)
const preflight = ref<BackupRestorePreflightDTO | null>(null)
const restoreOpen = ref(false)
// Browser folder pickers cannot return the absolute path required by Server.
const canPickDirectory = typeof window.javLibrary?.pickDirectory === "function"
const busy = computed(() => busyAction.value !== null)

let lastSyncedDirectory = ""
watch(() => libraryService.backupDirectory.value, (value) => {
  const next = value.trim()
  const current = backupDirectoryDraft.value.trim()
  if (current === "" || current === lastSyncedDirectory) backupDirectoryDraft.value = next
  lastSyncedDirectory = next
}, { immediate: true })

watch(backupPathDraft, () => {
  preflight.value = null
  pathError.value = ""
  restoreError.value = ""
}, { flush: "sync" })
watch(backupDirectoryDraft, () => { directoryError.value = "" })

function formatError(error: unknown): string {
  if (error instanceof HttpClientError && error.apiError?.message) return error.apiError.message
  if (error instanceof Error && error.message.trim()) return error.message
  return t("settings.backupUnknownError")
}

async function pickBackupDirectory() {
  if (!props.supported || busy.value || !canPickDirectory) return
  directoryError.value = ""
  busyAction.value = "pick"
  try {
    const outcome = await pickLibraryDirectory()
    if (outcome.status === "ok") backupDirectoryDraft.value = outcome.path
    else if (outcome.status === "hint") directoryError.value = outcome.message
    else if (outcome.status === "unsupported") directoryError.value = t("settings.backupPickerUnsupported")
  } catch (error) {
    directoryError.value = formatError(error)
  } finally {
    busyAction.value = null
  }
}

async function createAndVerifyBackup() {
  if (!props.supported || busy.value) return
  directoryError.value = ""
  const directory = backupDirectoryDraft.value.trim()
  if (!directory || !isAbsoluteLibraryPath(directory)) {
    directoryError.value = t("settings.backupDirectoryAbsolute")
    return
  }
  backupDirectoryDraft.value = directory
  const path = joinBackupDestination(directory, buildBackupFilename())
  createError.value = ""
  directoryPersistenceError.value = ""
  createdPath.value = ""
  createdVerification.value = null
  busyAction.value = "create"
  try {
    await libraryService.createBackup(path)
    createdPath.value = path
    try {
      await libraryService.setBackupDirectory(directory)
    } catch (error) {
      directoryPersistenceError.value = t("settings.backupDirectorySaveFailed", { error: formatError(error) })
    }
    createdVerification.value = await libraryService.verifyBackup(path)
    if (!createdVerification.value.valid) {
      createError.value = t("settings.backupCreatedVerificationFailed")
      return
    }
    pushAppToast(t("settings.backupCreatedToast"), { variant: "success", durationMs: 3600 })
  } catch (error) {
    createError.value = createdPath.value
      ? `${t("settings.backupCreatedVerificationFailed")} ${formatError(error)}`
      : formatError(error)
  } finally {
    busyAction.value = null
  }
}

function useCreatedBackup() {
  if (busy.value || !createdPath.value) return
  backupPathDraft.value = createdPath.value
  restoreOpen.value = true
}

async function preflightExistingBackup() {
  if (!props.supported || busy.value) return
  preflight.value = null
  restoreError.value = ""
  pathError.value = ""
  const path = ensureBackupExtension(backupPathDraft.value)
  if (!path || !isAbsoluteLibraryPath(path)) {
    pathError.value = t("settings.backupPathAbsolute")
    return
  }
  backupPathDraft.value = path
  busyAction.value = "preflight"
  try {
    // Preflight already verifies the package; users need only one check action.
    const result = await libraryService.preflightBackupRestore(path)
    if (backupPathDraft.value === path) preflight.value = result
  } catch (error) {
    restoreError.value = formatError(error)
  } finally {
    busyAction.value = null
  }
}

const restoreIssues = computed(() => {
  if (!preflight.value) return []
  return [...new Set([...preflight.value.verification.errors, ...preflight.value.errors])]
})
const restoreWarnings = computed(() => {
  if (!preflight.value) return []
  return [...new Set([...preflight.value.verification.warnings, ...preflight.value.warnings])]
})

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: index === 0 ? 0 : 1 }).format(value / 1024 ** index)} ${units[index]}`
}
</script>

<template>
  <section
    aria-labelledby="settings-backup-title"
    class="flex min-w-0 flex-col gap-4 rounded-lg border border-border/50 bg-muted/5 p-4"
    data-settings-maintenance-block="backup"
    :aria-busy="busy"
  >
    <div class="flex min-w-0 flex-wrap items-center gap-2">
      <SettingsHint :text="t('settings.backupCardDesc')">
        <h3 id="settings-backup-title" class="min-w-0 text-sm font-semibold text-foreground">
          {{ t("settings.backupCardTitle") }}
        </h3>
      </SettingsHint>
      <Badge v-if="!supported" variant="secondary">{{ t("settings.backupWebRequired") }}</Badge>
    </div>
    <p class="text-xs leading-relaxed text-muted-foreground">{{ t("settings.backupScopeNotice") }}</p>

    <form class="flex min-w-0 flex-col gap-3" @submit.prevent="createAndVerifyBackup">
      <FieldGroup>
        <Field class="gap-2" :data-invalid="Boolean(directoryError)" :data-disabled="!supported || busy">
          <SettingsHint :text="t('settings.backupDirectoryHint')">
            <FieldLabel for="settings-backup-directory">{{ t("settings.backupDirectoryLabel") }}</FieldLabel>
          </SettingsHint>
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <Input
              id="settings-backup-directory"
              v-model="backupDirectoryDraft"
              :disabled="!supported || busy"
              :aria-invalid="Boolean(directoryError)"
              aria-describedby="settings-backup-directory-help"
              :placeholder="t('settings.backupDirectoryPlaceholder')"
              autocomplete="off"
              class="min-w-0 flex-1"
              data-settings-backup-directory
            />
            <Button
              v-if="canPickDirectory"
              type="button" variant="outline" size="sm"
              class="h-auto min-h-11 rounded-full sm:h-8 sm:min-h-8"
              :disabled="!supported || busy"
              data-settings-comfortable-control data-settings-backup-pick
              @click="pickBackupDirectory"
            >
              <FolderOpen data-icon="inline-start" />{{ t("settings.backupPickDirectory") }}
            </Button>
          </div>
          <p v-if="directoryError" id="settings-backup-directory-help" class="text-xs text-destructive" role="alert">{{ directoryError }}</p>
          <p v-else-if="!canPickDirectory" id="settings-backup-directory-help" class="text-xs text-muted-foreground">{{ t("settings.backupBrowserPathHint") }}</p>
          <span v-else id="settings-backup-directory-help" class="sr-only">{{ t("settings.backupDirectoryHint") }}</span>
        </Field>
      </FieldGroup>
      <div class="flex justify-end">
        <Button
          type="submit" size="sm" class="h-auto min-h-11 rounded-full sm:h-8 sm:min-h-8"
          :disabled="!supported || busy" data-settings-comfortable-control data-settings-backup-create
        >
          <RotateCw v-if="busyAction === 'create'" data-icon="inline-start" class="motion-safe:animate-spin" />
          <DatabaseBackup v-else data-icon="inline-start" />
          {{ t(busyAction === 'create' ? 'settings.backupCreating' : 'settings.backupCreate') }}
        </Button>
      </div>
    </form>

    <div v-if="createdPath || createError || directoryPersistenceError" class="flex min-w-0 flex-col gap-2" aria-live="polite" data-settings-backup-receipt>
      <template v-if="createdPath">
        <div class="flex flex-wrap items-center gap-2">
          <Badge :variant="createdVerification?.valid ? 'success' : 'warning'">
            {{ t(createdVerification?.valid ? 'settings.backupCreatedToast' : 'settings.backupCreatedUnchecked') }}
          </Badge>
          <span class="text-xs text-muted-foreground">{{ t("settings.backupSavedTo") }}</span>
        </div>
        <code class="select-all break-all text-xs text-foreground" data-settings-backup-created-path>{{ createdPath }}</code>
      </template>
      <p v-if="createError" class="break-words text-sm text-destructive" role="alert">{{ createError }}</p>
      <p v-if="directoryPersistenceError" class="break-words text-sm text-warning" role="alert">{{ directoryPersistenceError }}</p>
      <ul v-if="createdVerification?.errors.length" class="list-disc break-words ps-5 text-xs text-destructive">
        <li v-for="error in createdVerification.errors" :key="error">{{ error }}</li>
      </ul>
      <ul v-if="createdVerification?.warnings.length" class="list-disc break-words ps-5 text-xs text-muted-foreground">
        <li v-for="warning in createdVerification.warnings" :key="warning">{{ warning }}</li>
      </ul>
    </div>

    <Separator />
    <Collapsible v-model:open="restoreOpen" class="flex min-w-0 flex-col gap-3">
      <CollapsibleTrigger as-child>
        <Button type="button" variant="ghost" size="sm" class="h-auto min-h-11 self-start rounded-full sm:h-8 sm:min-h-8" data-settings-comfortable-control data-settings-backup-existing>
          <ChevronDown data-icon="inline-start" :class="{ 'rotate-180': restoreOpen }" />
          {{ t("settings.backupExistingTitle") }}
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent class="flex min-w-0 flex-col gap-3">
        <p class="text-xs leading-relaxed text-muted-foreground">{{ t("settings.backupRestoreBoundary") }}</p>
        <form class="flex min-w-0 flex-col gap-3" @submit.prevent="preflightExistingBackup">
          <FieldGroup>
            <Field class="gap-2" :data-invalid="Boolean(pathError)" :data-disabled="!supported || busy">
              <SettingsHint :text="t('settings.backupPathHint')">
                <FieldLabel for="settings-backup-path">{{ t("settings.backupPathLabel") }}</FieldLabel>
              </SettingsHint>
              <Input
                id="settings-backup-path" v-model="backupPathDraft" :disabled="!supported || busy"
                :aria-invalid="Boolean(pathError)" aria-describedby="settings-backup-path-help"
                :placeholder="t('settings.backupPathPlaceholder')" autocomplete="off" data-settings-backup-path
              />
              <p v-if="pathError" id="settings-backup-path-help" class="text-xs text-destructive" role="alert">{{ pathError }}</p>
              <span v-else id="settings-backup-path-help" class="sr-only">{{ t("settings.backupPathHint") }}</span>
            </Field>
          </FieldGroup>
          <div class="flex flex-wrap justify-end gap-2">
            <Button v-if="createdPath" type="button" variant="outline" size="sm" class="h-auto min-h-11 rounded-full sm:h-8 sm:min-h-8" :disabled="!supported || busy" data-settings-comfortable-control data-settings-backup-use-created @click="useCreatedBackup">
              {{ t("settings.backupUseCreated") }}
            </Button>
            <Button type="submit" size="sm" class="h-auto min-h-11 rounded-full sm:h-8 sm:min-h-8" :disabled="!supported || busy" data-settings-comfortable-control data-settings-backup-preflight>
              <RotateCw v-if="busyAction === 'preflight'" data-icon="inline-start" class="motion-safe:animate-spin" />
              <ShieldCheck v-else data-icon="inline-start" />
              {{ t(busyAction === 'preflight' ? 'settings.backupChecking' : 'settings.backupPreflight') }}
            </Button>
          </div>
        </form>
        <p v-if="restoreError" class="break-words text-sm text-destructive" role="alert">{{ restoreError }}</p>
        <div v-if="preflight" class="flex min-w-0 flex-col gap-3" aria-live="polite" data-settings-backup-check-result>
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="preflight.canRestore ? 'success' : 'danger'">
              {{ t(preflight.canRestore ? 'settings.backupPreflightReady' : 'settings.backupPreflightBlocked') }}
            </Badge>
            <span class="text-xs text-muted-foreground">{{ t("settings.backupCapacitySummary", {
              required: formatBytes(preflight.requiredBytes),
              available: preflight.availableBytesKnown ? formatBytes(preflight.availableBytes) : '—',
            }) }}</span>
          </div>
          <ul v-if="restoreIssues.length" class="list-disc break-words ps-5 text-xs text-destructive">
            <li v-for="error in restoreIssues" :key="error">{{ error }}</li>
          </ul>
          <ul v-if="restoreWarnings.length" class="list-disc break-words ps-5 text-xs text-muted-foreground">
            <li v-for="warning in restoreWarnings" :key="warning">{{ warning }}</li>
          </ul>
          <template v-if="preflight.canRestore">
            <p class="text-sm font-medium">{{ t("settings.backupRestoreNextTitle") }}</p>
            <ol class="flex list-decimal flex-col gap-2 ps-5 text-xs leading-relaxed text-muted-foreground">
              <li>{{ t("settings.backupRestoreStepSave") }}</li>
              <li>{{ t("settings.backupRestoreStepQuit") }}</li>
              <li>{{ t("settings.backupRestoreStepRun") }}</li>
            </ol>
            <p class="text-xs text-muted-foreground">{{ t("settings.backupRestoreTarget") }}</p>
            <code class="select-all break-all text-xs">{{ preflight.targetDatabase }}</code>
            <Button as-child variant="outline" size="sm" class="h-auto min-h-11 self-end rounded-full sm:h-8 sm:min-h-8" data-settings-comfortable-control>
              <a href="https://github.com/yepHiu/Curated/blob/master/docs/guide.md#3-backup-restore-and-path-migration" target="_blank" rel="noopener noreferrer">
                <ExternalLink data-icon="inline-start" />{{ t("settings.backupRestoreGuide") }}
              </a>
            </Button>
          </template>
        </div>
      </CollapsibleContent>
    </Collapsible>
  </section>
</template>
