<script setup lang="ts">
import SettingsHint from "./SettingsHint.vue"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { PlayCircle } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import type {
  HardwareEncoderPreference,
  NativePlayerPreset,
  PatchPlayerSettingsBody,
} from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
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
import { Switch } from "@/components/ui/switch"
import { useServerLocalAccess } from "@/composables/use-server-local-access"
import { useSettingsScrollPreserve } from "@/composables/use-settings-scroll-preserve"
import {
  defaultNativePlayerBackendCommand,
  defaultNativePlayerBrowserTemplate,
  getStoredNativePlayerBrowserTemplate,
  normalizeNativePlayerPresetForBrowserLaunch,
  persistNativePlayerBrowserTemplate,
  resolveNativePlayerBrowserTemplate,
} from "@/lib/native-player-launch"
import { normalizeHardwareEncoderPreference } from "@/lib/playback-settings-normalize"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  autoSaveReady: boolean
}>()

const { t } = useI18n()
const libraryService = useLibraryService()
const { isServerLocal } = useServerLocalAccess()
const { withPreservedScroll } = useSettingsScrollPreserve()

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"

const PLAYBACK_HARDWARE_ENCODER_OPTIONS: readonly HardwareEncoderPreference[] = [
  "auto",
  "amf",
  "qsv",
  "nvenc",
  "videotoolbox",
  "software",
]
const PLAYBACK_NATIVE_PLAYER_PRESET_OPTIONS: readonly NativePlayerPreset[] = [
  "custom",
  "potplayer",
  "mpv",
]

const playbackHardwareDecodeDraft = ref(true)
const playbackHardwareEncoderDraft = ref<HardwareEncoderPreference>("auto")
const playbackNativePlayerPresetDraft = ref<NativePlayerPreset>("custom")
const playbackNativePlayerEnabledDraft = ref(false)
const playbackNativePlayerProtocolTemplateDraft = ref(defaultNativePlayerBrowserTemplate("custom"))
const playbackStreamPushEnabledDraft = ref(true)
const playbackFfmpegCommandDraft = ref("ffmpeg")
const playbackPreferNativePlayerDraft = ref(false)
const playbackSeekForwardStepDraft = ref("10")
const playbackSeekBackwardStepDraft = ref("10")
const playbackSaving = ref(false)
const playbackError = ref("")
const playbackSavedFlash = ref(false)
let playbackSavedFlashTimer: ReturnType<typeof setTimeout> | null = null
let playbackSavePromise: Promise<void> | null = null
let playbackSaveQueued = false

function syncPlaybackDraftFromService() {
  const player = libraryService.playerSettings.value
  playbackHardwareDecodeDraft.value = player.hardwareDecode !== false
  playbackHardwareEncoderDraft.value = normalizeHardwareEncoderPreference(player.hardwareEncoder)
  playbackNativePlayerPresetDraft.value = normalizeNativePlayerPresetForBrowserLaunch(
    player.nativePlayerPreset,
    player.nativePlayerCommand,
  )
  playbackNativePlayerEnabledDraft.value = Boolean(player.nativePlayerEnabled)
  playbackNativePlayerProtocolTemplateDraft.value = resolveNativePlayerBrowserTemplate(
    playbackNativePlayerPresetDraft.value,
    getStoredNativePlayerBrowserTemplate(),
  )
  playbackStreamPushEnabledDraft.value = player.streamPushEnabled !== false
  playbackFfmpegCommandDraft.value = (player.ffmpegCommand ?? "ffmpeg").trim() || "ffmpeg"
  playbackPreferNativePlayerDraft.value = Boolean(player.preferNativePlayer)
  playbackSeekForwardStepDraft.value = String(Math.max(1, Number(player.seekForwardStepSec ?? 10)))
  playbackSeekBackwardStepDraft.value = String(Math.max(1, Number(player.seekBackwardStepSec ?? 10)))
}

function playbackNativePlayerDefaultCommand(preset: NativePlayerPreset | undefined): string {
  return defaultNativePlayerBackendCommand(preset)
}

function playbackNativePlayerPresetLabel(value: NativePlayerPreset): string {
  switch (value) {
    case "potplayer":
      return t("settings.playbackNativePlayerPresetPotplayer")
    case "custom":
      return t("settings.playbackNativePlayerPresetCustom")
    case "mpv":
    default:
      return t("settings.playbackNativePlayerPresetMpv")
  }
}

const playbackNativePlayerProtocolTemplatePlaceholder = computed(() => {
  if (playbackNativePlayerPresetDraft.value === "custom") {
    return t("settings.playbackNativePlayerCommandPlaceholderCustom")
  }
  return (
    defaultNativePlayerBrowserTemplate(playbackNativePlayerPresetDraft.value) ||
    t("settings.playbackNativePlayerCommandPlaceholder")
  )
})

function onPlaybackNativePlayerPresetChange(value: unknown) {
  const nextPreset = normalizeNativePlayerPresetForBrowserLaunch(
    typeof value === "string" ? (value as NativePlayerPreset) : undefined,
  )
  const prevPreset = playbackNativePlayerPresetDraft.value
  const currentTemplate = playbackNativePlayerProtocolTemplateDraft.value.trim()
  const previousDefault = defaultNativePlayerBrowserTemplate(prevPreset)
  playbackNativePlayerPresetDraft.value = nextPreset
  if (!currentTemplate || currentTemplate === previousDefault) {
    playbackNativePlayerProtocolTemplateDraft.value = defaultNativePlayerBrowserTemplate(nextPreset)
  }
}

function playbackHardwareEncoderLabel(value: HardwareEncoderPreference): string {
  switch (value) {
    case "amf":
      return t("settings.playbackHardwareEncoderAmf")
    case "qsv":
      return t("settings.playbackHardwareEncoderQsv")
    case "nvenc":
      return t("settings.playbackHardwareEncoderNvenc")
    case "videotoolbox":
      return t("settings.playbackHardwareEncoderVideotoolbox")
    case "software":
      return t("settings.playbackHardwareEncoderSoftware")
    case "auto":
    default:
      return t("settings.playbackHardwareEncoderAuto")
  }
}

/** 远端只提交可见偏好，避免携带过期的底层服务端配置或原生命令。 */
function buildPlaybackPatchFromDraft(): PatchPlayerSettingsBody | null {
  const forward = Number.parseInt(playbackSeekForwardStepDraft.value, 10)
  const backward = Number.parseInt(playbackSeekBackwardStepDraft.value, 10)
  if (!Number.isFinite(forward) || forward <= 0) {
    playbackError.value = t("settings.playbackSeekForwardInvalid")
    return null
  }
  if (!Number.isFinite(backward) || backward <= 0) {
    playbackError.value = t("settings.playbackSeekBackwardInvalid")
    return null
  }
  const player = libraryService.playerSettings.value
  const currentPreset = normalizeNativePlayerPresetForBrowserLaunch(
    player.nativePlayerPreset,
    player.nativePlayerCommand,
  )
  const currentCommand = (player.nativePlayerCommand ?? "").trim()
  const currentDefault = playbackNativePlayerDefaultCommand(currentPreset)
  const nextDefault = playbackNativePlayerDefaultCommand(playbackNativePlayerPresetDraft.value)
  const nextBackendCommand =
    !currentCommand || currentCommand === currentDefault || currentPreset !== playbackNativePlayerPresetDraft.value
      ? nextDefault
      : currentCommand
  return {
    ...(isServerLocal.value ? {
      hardwareDecode: playbackHardwareDecodeDraft.value,
      hardwareEncoder: normalizeHardwareEncoderPreference(playbackHardwareEncoderDraft.value),
      nativePlayerCommand: nextBackendCommand,
      streamPushEnabled: playbackStreamPushEnabledDraft.value,
      ...(!playbackStreamPushEnabledDraft.value ? { forceStreamPush: false } : {}),
      ffmpegCommand: playbackFfmpegCommandDraft.value.trim() || "ffmpeg",
    } : {}),
    nativePlayerPreset: playbackNativePlayerPresetDraft.value,
    nativePlayerEnabled: playbackNativePlayerEnabledDraft.value,
    preferNativePlayer: playbackPreferNativePlayerDraft.value,
    seekForwardStepSec: forward,
    seekBackwardStepSec: backward,
  }
}

/** 隐藏字段不参与脏值检测，防止远端触发无关的自动保存。 */
function playbackDraftMatchesServer(): boolean {
  const player = libraryService.playerSettings.value
  return (
    (!isServerLocal.value || (
      playbackHardwareDecodeDraft.value === (player.hardwareDecode !== false) &&
      normalizeHardwareEncoderPreference(playbackHardwareEncoderDraft.value) ===
        normalizeHardwareEncoderPreference(player.hardwareEncoder) &&
      playbackStreamPushEnabledDraft.value === (player.streamPushEnabled !== false) &&
      (playbackFfmpegCommandDraft.value.trim() || "ffmpeg") ===
        ((player.ffmpegCommand ?? "ffmpeg").trim() || "ffmpeg")
    )) &&
    playbackNativePlayerPresetDraft.value ===
      normalizeNativePlayerPresetForBrowserLaunch(player.nativePlayerPreset, player.nativePlayerCommand) &&
    playbackNativePlayerEnabledDraft.value === (player.nativePlayerEnabled !== false) &&
    playbackPreferNativePlayerDraft.value === Boolean(player.preferNativePlayer) &&
    Number.parseInt(playbackSeekForwardStepDraft.value, 10) ===
      Math.max(1, Number(player.seekForwardStepSec ?? 10)) &&
    Number.parseInt(playbackSeekBackwardStepDraft.value, 10) ===
      Math.max(1, Number(player.seekBackwardStepSec ?? 10))
  )
}

function playbackBrowserTemplateMatchesPersisted(): boolean {
  return playbackNativePlayerProtocolTemplateDraft.value.trim() ===
    resolveNativePlayerBrowserTemplate(
      playbackNativePlayerPresetDraft.value,
      getStoredNativePlayerBrowserTemplate(),
    )
}

function flashPlaybackSaved() {
  playbackSavedFlash.value = true
  if (playbackSavedFlashTimer) clearTimeout(playbackSavedFlashTimer)
  playbackSavedFlashTimer = setTimeout(() => {
    playbackSavedFlash.value = false
    playbackSavedFlashTimer = null
  }, 2200)
}

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
    await withPreservedScroll(async () => {
      playbackSaving.value = true
      try {
        if (shouldPatchServer && patch) {
          await libraryService.patchPlayerSettings(patch)
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
    })
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

watchDebounced(
  () =>
    [
      playbackHardwareDecodeDraft.value,
      playbackHardwareEncoderDraft.value,
      playbackNativePlayerPresetDraft.value,
      playbackNativePlayerEnabledDraft.value,
      playbackNativePlayerProtocolTemplateDraft.value,
      playbackStreamPushEnabledDraft.value,
      playbackFfmpegCommandDraft.value,
      playbackPreferNativePlayerDraft.value,
      playbackSeekForwardStepDraft.value,
      playbackSeekBackwardStepDraft.value,
    ] as const,
  async () => {
    if (!props.autoSaveReady) {
      return
    }
    if (playbackDraftMatchesServer() && playbackBrowserTemplateMatchesPersisted()) {
      return
    }
    await savePlaybackSettings()
  },
  { debounce: 550, maxWait: 5000 },
)

/**
 * 父页在 onMounted 末尾才把 autoSaveReady 置为 true（此前已 await refreshSettings）。
 * 若在此处用 onMounted 同步草稿，会早于 GET /api/settings，playerSettings 仍是占位值，
 * 例如 forceStreamPush 会与 library-config.cfg 不一致。
 */
watch(
  () => props.autoSaveReady,
  (ready) => {
    if (ready) {
      syncPlaybackDraftFromService()
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (playbackSavedFlashTimer) clearTimeout(playbackSavedFlashTimer)
})
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <PlayCircle class="size-[1.15rem]" />
          </span>
          <SettingsHint :text="t('settings.playbackCardDesc')">
            <CardTitle class="flex flex-wrap items-center gap-2 min-w-0 text-lg tracking-tight">
              <span>{{ t("settings.playbackCardTitle") }}</span>
              <SettingsScopeBadge scope="mixed" />
            </CardTitle>
          </SettingsHint>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <div
            v-if="isServerLocal"
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.hardwareDecodeHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.hardwareDecode") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <Switch v-model="playbackHardwareDecodeDraft" />
          </div>

          <div
            v-if="isServerLocal"
            class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
          >
            <div class="min-w-0 flex-1 flex flex-col gap-3">
              <SettingsHint :text="t('settings.playbackHardwareEncoderHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackHardwareEncoder") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <div class="flex w-full min-w-0 justify-stretch sm:w-auto sm:shrink-0 sm:justify-end">
              <Select v-model="playbackHardwareEncoderDraft" :disabled="!playbackHardwareDecodeDraft">
                <SelectTrigger size="sm" class="w-full min-w-0 sm:w-fit">
                  <SelectValue :placeholder="t('settings.playbackHardwareEncoderAuto')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="option in PLAYBACK_HARDWARE_ENCODER_OPTIONS"
                    :key="option"
                    :value="option"
                  >
                    {{ playbackHardwareEncoderLabel(option) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div
            v-if="isServerLocal"
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.playbackStreamPushEnabledHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackStreamPushEnabled") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <Switch v-model="playbackStreamPushEnabledDraft" />
          </div>

          <div v-if="isServerLocal" class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.playbackFfmpegCommandHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackFfmpegCommand") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <Input
              v-model="playbackFfmpegCommandDraft"
              :placeholder="t('settings.playbackFfmpegCommandPlaceholder')"
            />
          </div>

          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.playbackNativePlayerEnabledHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackNativePlayerEnabled") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <Switch v-model="playbackNativePlayerEnabledDraft" />
          </div>

          <div
            class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
          >
            <div class="min-w-0 flex-1 flex flex-col gap-3">
              <SettingsHint :text="t('settings.playbackNativePlayerPresetHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackNativePlayerPreset") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <div class="flex w-full min-w-0 justify-stretch sm:w-auto sm:shrink-0 sm:justify-end">
              <Select
                :model-value="playbackNativePlayerPresetDraft"
                @update:model-value="onPlaybackNativePlayerPresetChange"
              >
                <SelectTrigger size="sm" class="w-full min-w-0 sm:w-fit">
                  <SelectValue :placeholder="t('settings.playbackNativePlayerPresetMpv')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="option in PLAYBACK_NATIVE_PLAYER_PRESET_OPTIONS"
                    :key="option"
                    :value="option"
                  >
                    {{ playbackNativePlayerPresetLabel(option) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4">
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.playbackNativePlayerCommandHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackNativePlayerCommand") }}</span>
                  <SettingsScopeBadge scope="client" />
                </p>
              </SettingsHint>
            </div>
            <Input
              v-model="playbackNativePlayerProtocolTemplateDraft"
              :placeholder="playbackNativePlayerProtocolTemplatePlaceholder"
            />
          </div>

          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-border/50 bg-muted/5 p-4"
          >
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <SettingsHint :text="t('settings.playbackPreferNativePlayerHint')">
                <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                  <span>{{ t("settings.playbackPreferNativePlayer") }}</span>
                  <SettingsScopeBadge scope="server" />
                </p>
              </SettingsHint>
            </div>
            <Switch v-model="playbackPreferNativePlayerDraft" />
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <div
              class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
            >
              <div class="min-w-0 flex-1 flex flex-col gap-3">
                <SettingsHint :text="t('settings.playbackSeekBackwardStepHint')">
                  <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                    <span>{{ t("settings.playbackSeekBackwardStep") }}</span>
                    <SettingsScopeBadge scope="server" />
                  </p>
                </SettingsHint>
              </div>
              <div class="flex w-full min-w-0 justify-stretch sm:w-auto sm:shrink-0 sm:justify-end">
                <Input
                  v-model="playbackSeekBackwardStepDraft"
                  type="number"
                  min="1"
                  inputmode="numeric"
                  class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50 sm:max-w-[7rem]"
                />
              </div>
            </div>

            <div
              class="flex min-w-0 flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
            >
              <div class="min-w-0 flex-1 flex flex-col gap-3">
                <SettingsHint :text="t('settings.playbackSeekForwardStepHint')">
                  <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-foreground">
                    <span>{{ t("settings.playbackSeekForwardStep") }}</span>
                    <SettingsScopeBadge scope="server" />
                  </p>
                </SettingsHint>
              </div>
              <div class="flex w-full min-w-0 justify-stretch sm:w-auto sm:shrink-0 sm:justify-end">
                <Input
                  v-model="playbackSeekForwardStepDraft"
                  type="number"
                  min="1"
                  inputmode="numeric"
                  class="h-8 min-h-8 max-h-8 py-0 text-sm rounded-xl border-border/50 sm:max-w-[7rem]"
                />
              </div>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <Button
              v-if="!useWebApi"
              type="button"
              class="rounded-lg"
              :disabled="playbackSaving"
              @click="savePlaybackSettings"
            >
              {{ playbackSaving ? t("common.saving") : t("common.save") }}
            </Button>
            <p
              v-if="playbackSaving"
              class="text-xs text-muted-foreground motion-safe:animate-pulse"
            >
              {{ t("settings.playbackSyncing") }}
            </p>
            <p v-else-if="playbackSavedFlash" class="text-xs text-muted-foreground">
              {{ t("settings.autoPersistSaved") }}
            </p>
          </div>

          <p v-if="playbackError" class="text-sm text-destructive">
            {{ playbackError }}
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
