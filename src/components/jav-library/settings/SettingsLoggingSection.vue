<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { watchDebounced } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { Activity, FolderOpen, ScrollText } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useSettingsScrollPreserve } from "@/composables/use-settings-scroll-preserve"
import {
  CLIENT_LOG_LEVEL_OPTIONS,
  getClientLogLevelName,
  setClientLogLevel,
  type ClientLogLevelName,
} from "@/lib/app-logger"
import { pickLibraryDirectory } from "@/lib/pick-directory"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  autoSaveReady: boolean
}>()

const { t } = useI18n()
const libraryService = useLibraryService()
const { withPreservedScroll } = useSettingsScrollPreserve()

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const BACKEND_LOG_LEVEL_OPTIONS = ["trace", "debug", "info", "warn", "error"] as const
const BACKEND_LOG_MAX_AGE_PRESET_VALUES = ["0", "1", "5", "10", "30"] as const

const backendLogDirDraft = ref("")
const backendLogMaxAgeDaysChoice = ref("0")
const backendLogLevelDraft = ref("info")
const backendLogSaving = ref(false)
const backendLogError = ref("")
const backendLogDirPickHint = ref("")
const pickBackendLogDirBusy = ref(false)
const clientLogLevelUi = ref<ClientLogLevelName>(getClientLogLevelName())
const backendLogSavedFlash = ref(false)
let backendLogSavedFlashTimer: ReturnType<typeof setTimeout> | null = null
let backendLogSavePromise: Promise<void> | null = null
let backendLogSaveQueued = false

function syncBackendLogMaxAgeDaysChoiceFromDto(d: number | undefined) {
  if (d === undefined || d <= 0) {
    backendLogMaxAgeDaysChoice.value = "0"
    return
  }
  backendLogMaxAgeDaysChoice.value = String(d)
}

const backendLogMaxAgeSelectItems = computed(() => {
  const items = [
    { value: "0", label: t("settings.backendLogMaxAgeOptDefault") },
    { value: "1", label: t("settings.backendLogMaxAgeOpt1") },
    { value: "5", label: t("settings.backendLogMaxAgeOpt5") },
    { value: "10", label: t("settings.backendLogMaxAgeOpt10") },
    { value: "30", label: t("settings.backendLogMaxAgeOpt30") },
  ]
  const cur = backendLogMaxAgeDaysChoice.value
  if (
    cur &&
    !BACKEND_LOG_MAX_AGE_PRESET_VALUES.includes(cur as (typeof BACKEND_LOG_MAX_AGE_PRESET_VALUES)[number])
  ) {
    const n = Number.parseInt(cur, 10)
    if (Number.isFinite(n) && n > 0) {
      items.push({ value: cur, label: t("settings.backendLogMaxAgeOptCustom", { n }) })
    }
  }
  return items
})

function syncBackendLogDraftFromService() {
  const b = libraryService.backendLog.value
  backendLogDirDraft.value = (b.logDir ?? "").trim()
  syncBackendLogMaxAgeDaysChoiceFromDto(b.logMaxAgeDays)
  const lvl = (b.logLevel ?? "info").trim() || "info"
  backendLogLevelDraft.value = (BACKEND_LOG_LEVEL_OPTIONS as readonly string[]).includes(lvl)
    ? lvl
    : "info"
}

function onClientLogLevelSelect(v: unknown) {
  if (typeof v === "string" && (CLIENT_LOG_LEVEL_OPTIONS as readonly string[]).includes(v)) {
    setClientLogLevel(v as ClientLogLevelName)
    clientLogLevelUi.value = v as ClientLogLevelName
  }
}

async function pickBackendLogDirectory() {
  backendLogDirPickHint.value = ""
  backendLogError.value = ""
  pickBackendLogDirBusy.value = true
  try {
    const outcome = await pickLibraryDirectory()
    if (outcome.status === "ok") {
      backendLogDirDraft.value = outcome.path
      return
    }
    if (outcome.status === "hint") {
      backendLogDirPickHint.value = outcome.message
      return
    }
    if (outcome.status === "unsupported") {
      backendLogDirPickHint.value = t("settings.errPickUnsupported")
    }
  } finally {
    pickBackendLogDirBusy.value = false
  }
}

function flashBackendLogSaved() {
  backendLogSavedFlash.value = true
  if (backendLogSavedFlashTimer) clearTimeout(backendLogSavedFlashTimer)
  backendLogSavedFlashTimer = setTimeout(() => {
    backendLogSavedFlash.value = false
    backendLogSavedFlashTimer = null
  }, 2200)
}

async function performSaveBackendLogSettings() {
  backendLogError.value = ""
  backendLogDirPickHint.value = ""
  const maxAge = Number.parseInt(backendLogMaxAgeDaysChoice.value, 10)
  if (!Number.isFinite(maxAge) || maxAge < 0) {
    backendLogError.value = t("settings.backendLogMaxAgeInvalid")
    return
  }
  try {
    await withPreservedScroll(async () => {
      backendLogSaving.value = true
      try {
        await libraryService.patchBackendLog({
          logDir: backendLogDirDraft.value.trim(),
          logMaxAgeDays: maxAge,
          logLevel: backendLogLevelDraft.value.trim() || "info",
        })
        syncBackendLogDraftFromService()
        flashBackendLogSaved()
      } finally {
        backendLogSaving.value = false
      }
    })
  } catch (err) {
    console.error("[settings] save backend log failed", err)
    if (err instanceof HttpClientError && err.apiError?.message) {
      backendLogError.value = err.apiError.message
    } else if (err instanceof Error && err.message) {
      backendLogError.value = err.message
    } else {
      backendLogError.value = t("settings.errSaveTitle")
    }
  }
}

async function saveBackendLogSettings() {
  if (backendLogSavePromise) {
    backendLogSaveQueued = true
    return backendLogSavePromise
  }

  backendLogSavePromise = (async () => {
    do {
      backendLogSaveQueued = false
      await performSaveBackendLogSettings()
    } while (backendLogSaveQueued)
  })()

  try {
    await backendLogSavePromise
  } finally {
    backendLogSavePromise = null
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
    if (!props.autoSaveReady || !useWebApi) {
      return
    }
    if (backendLogDraftMatchesServer()) {
      return
    }
    await saveBackendLogSettings()
  },
  { debounce: 550, maxWait: 5000 },
)

onMounted(() => {
  syncBackendLogDraftFromService()
  clientLogLevelUi.value = getClientLogLevelName()
})

onBeforeUnmount(() => {
  if (backendLogSavedFlashTimer) clearTimeout(backendLogSavedFlashTimer)
})
</script>

<template>
  <div id="settings-section-logging" class="flex flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="space-y-2 pb-0">
          <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <ScrollText class="size-4" />
            </span>
            {{ t("settings.backendLogTitle") }}
          </CardTitle>
          <CardDescription
            class="text-xs leading-relaxed text-pretty text-muted-foreground"
          >
            {{ t("settings.backendLogDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <p
            v-if="!useWebApi"
            class="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-sm text-muted-foreground"
          >
            {{ t("settings.backendLogMockHint") }}
          </p>
          <div class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium">{{ t("settings.backendLogDir") }}</p>
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
                <Input
                  id="backend-log-dir"
                  v-model="backendLogDirDraft"
                  type="text"
                  autocomplete="off"
                  class="min-w-0 rounded-xl border-border/50 sm:flex-1"
                  :placeholder="t('settings.backendLogDirPlaceholder')"
                  :disabled="backendLogSaving || pickBackendLogDirBusy"
                  @input="backendLogDirPickHint = ''"
                />
                <Button
                  type="button"
                  variant="secondary"
                  class="rounded-2xl sm:shrink-0"
                  :disabled="backendLogSaving || pickBackendLogDirBusy"
                  @click="pickBackendLogDirectory"
                >
                  <FolderOpen data-icon="inline-start" aria-hidden="true" />
                  {{
                    pickBackendLogDirBusy ? t("settings.picking") : t("settings.pickFolder")
                  }}
                </Button>
              </div>
              <p
                v-if="backendLogDirPickHint"
                class="whitespace-pre-line text-sm leading-relaxed text-muted-foreground"
              >
                {{ backendLogDirPickHint }}
              </p>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div class="flex min-w-0 flex-col gap-3">
                <p class="text-sm font-medium">{{ t("settings.backendLogMaxAge") }}</p>
                <Select
                  v-model="backendLogMaxAgeDaysChoice"
                  :disabled="backendLogSaving"
                >
                  <SelectTrigger
                    class="h-9 w-full min-w-0 rounded-xl border-border/50"
                    :aria-label="t('settings.backendLogMaxAge')"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent class="rounded-xl border-border/50">
                    <SelectItem
                      v-for="item in backendLogMaxAgeSelectItems"
                      :key="`blog-maxage-${item.value}`"
                      class="rounded-lg"
                      :value="item.value"
                    >
                      {{ item.label }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="flex min-w-0 flex-col gap-3">
                <p class="text-sm font-medium">{{ t("settings.backendLogLevel") }}</p>
                <Select
                  v-model="backendLogLevelDraft"
                  :disabled="backendLogSaving"
                >
                  <SelectTrigger
                    class="h-9 w-full min-w-0 rounded-xl border-border/50"
                    :aria-label="t('settings.backendLogLevel')"
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent class="rounded-xl border-border/50">
                    <SelectItem
                      v-for="lvl in BACKEND_LOG_LEVEL_OPTIONS"
                      :key="`blog-${lvl}`"
                      class="rounded-lg"
                      :value="lvl"
                    >
                      {{ lvl }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <Button
            v-if="!useWebApi"
            type="button"
            class="w-fit rounded-lg"
            :disabled="backendLogSaving"
            @click="saveBackendLogSettings"
          >
            {{ backendLogSaving ? t("common.saving") : t("settings.backendLogSave") }}
          </Button>
          <p
            v-else-if="useWebApi && backendLogSaving"
            class="text-xs text-muted-foreground motion-safe:animate-pulse"
          >
            {{ t("settings.proxySyncing") }}
          </p>
          <p
            v-else-if="useWebApi && backendLogSavedFlash"
            class="text-xs text-muted-foreground"
          >
            {{ t("settings.autoPersistSaved") }}
          </p>
          <p v-if="backendLogError" class="text-sm text-destructive">
            {{ backendLogError }}
          </p>
        </CardContent>
      </Card>
    </div>

    <div class="break-inside-avoid">
      <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="space-y-3 pb-2">
          <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <Activity class="size-4" />
            </span>
            {{ t("settings.clientLogTitle") }}
          </CardTitle>
          <CardDescription
            class="text-xs leading-relaxed text-pretty text-muted-foreground"
          >
            {{ t("settings.clientLogDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-2">
          <div
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <p class="text-sm font-medium text-foreground">{{ t("settings.clientLogLevel") }}</p>
            <Select
              :model-value="clientLogLevelUi"
              @update:model-value="onClientLogLevelSelect"
            >
              <SelectTrigger
                class="h-9 w-full min-w-[11rem] shrink-0 rounded-xl border-border/50 sm:w-44"
                :aria-label="t('settings.clientLogLevel')"
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent class="rounded-xl border-border/50">
                <SelectItem
                  v-for="lvl in CLIENT_LOG_LEVEL_OPTIONS"
                  :key="`clog-${lvl}`"
                  class="rounded-lg"
                  :value="lvl"
                >
                  {{ lvl }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
