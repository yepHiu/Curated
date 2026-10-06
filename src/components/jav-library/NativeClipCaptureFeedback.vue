<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Circle, Film, Loader2 } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import type { DesktopPlaybackClip } from "../../../electron/playback-contract"
const props = defineProps<{ recording: boolean; elapsedSec: number; progress: number; clip?: DesktopPlaybackClip; busy: boolean }>()
defineEmits<{ cancel: []; retry: []; dismiss: [] }>()
const { t } = useI18n()
const message = computed(() => props.recording ? t("player.clipRecording") : props.clip?.phase === "processing" ? t(props.clip.error === "CLIP_CANCEL_FAILED" ? "curated.clipCancelFailed" : "player.clipProcessing")
  : props.clip?.phase === "saved" ? t("player.clipSavedToLibrary") : props.clip?.phase === "cancelled" ? t("common.cancelled")
  : props.clip?.error === "CLIP_TIMED_OUT" ? t("player.clipExportTimedOut") : props.clip?.error === "CLIP_STATUS_UNAVAILABLE" ? t("player.clipStatusUnavailable") : t("player.clipExportFailed"))
</script>
<template>
  <div data-native-clip-feedback class="absolute left-1/2 z-20 w-[min(26rem,calc(100%-2rem))] -translate-x-1/2 rounded-xl border border-border bg-background px-4 py-3 text-foreground shadow-lg" @click.stop @pointerdown.stop @keydown.stop>
    <div class="flex flex-wrap items-center gap-2 text-sm font-semibold">
      <Circle v-if="recording" class="size-3 shrink-0 fill-danger text-danger" aria-hidden="true" />
      <Loader2 v-else-if="clip?.phase === 'processing'" class="size-4 shrink-0 animate-spin motion-reduce:animate-none" aria-hidden="true" />
      <Film v-else class="size-4 shrink-0 text-primary" aria-hidden="true" />
      <span role="status" aria-live="polite" class="min-w-0 flex-1">{{ message }}</span>
      <span v-if="recording" class="font-mono tabular-nums text-muted-foreground">{{ elapsedSec.toFixed(1) }}s / 6s</span>
      <Button v-if="recording || clip?.phase === 'processing'" size="sm" variant="ghost" :disabled="busy" @click="$emit('cancel')">{{ t('common.cancel') }}</Button>
      <Button v-else-if="clip?.phase === 'error'" size="sm" variant="outline" :disabled="busy" @click="$emit('retry')">{{ t('common.retry') }}</Button>
      <Button v-if="!recording && clip?.phase !== 'processing'" size="sm" variant="ghost" @click="$emit('dismiss')">{{ t('common.close') }}</Button>
    </div>
    <div v-if="recording || clip?.phase === 'processing'" role="progressbar" :aria-label="message" :aria-valuenow="Math.round(recording ? progress * 100 : clip?.progress ?? 0)" aria-valuemin="0" aria-valuemax="100" class="mt-2 h-1 overflow-hidden rounded-full bg-muted">
      <div class="h-full origin-left rounded-full bg-primary transition-transform duration-200 ease-linear motion-reduce:transition-none" :style="{ transform: `scaleX(${recording ? progress : (clip?.progress ?? 0) / 100})` }" />
    </div>
  </div>
</template>
