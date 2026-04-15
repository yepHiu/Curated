<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Keyboard, RotateCcw } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  getCuratedCaptureKeyCode,
  resetCuratedCaptureKeyCode,
  setCuratedCaptureKeyCode,
} from "@/lib/curated-frames/settings-storage"
import {
  DEFAULT_CURATED_CAPTURE_KEY_CODE,
  formatCuratedCaptureKeyLabel,
  isCuratedCaptureKeyReserved,
  isSupportedCuratedCaptureKeyCode,
} from "@/lib/player-shortcuts"

const { t } = useI18n()

const currentCode = ref(getCuratedCaptureKeyCode())
const listening = ref(false)
const errorMessage = ref("")

const currentLabel = computed(() => formatCuratedCaptureKeyLabel(currentCode.value))
const reservedLabels = computed(() => ["Space", "←", "→", "↑", "↓", "Esc", "J", "K", "L", "M", "F", "P"])
const statusText = computed(() =>
  listening.value ? t("settings.curatedShortcutListening") : t("settings.curatedShortcutIdle"),
)

function stopListening() {
  if (!listening.value) return
  listening.value = false
  window.removeEventListener("keydown", onCaptureKeydown, true)
}

function startListening() {
  errorMessage.value = ""
  if (listening.value) return
  listening.value = true
  window.addEventListener("keydown", onCaptureKeydown, true)
}

function onCaptureKeydown(event: KeyboardEvent) {
  if (!listening.value) return
  if (event.ctrlKey || event.metaKey || event.altKey) return

  event.preventDefault()
  event.stopPropagation()

  if (event.code === "Escape") {
    stopListening()
    return
  }

  if (isCuratedCaptureKeyReserved(event.code)) {
    errorMessage.value = t("settings.curatedShortcutReserved")
    stopListening()
    return
  }

  if (!isSupportedCuratedCaptureKeyCode(event.code)) {
    errorMessage.value = t("settings.curatedShortcutInvalid")
    stopListening()
    return
  }

  setCuratedCaptureKeyCode(event.code)
  currentCode.value = getCuratedCaptureKeyCode()
  errorMessage.value = ""
  stopListening()
}

function resetShortcut() {
  errorMessage.value = ""
  resetCuratedCaptureKeyCode()
  currentCode.value = DEFAULT_CURATED_CAPTURE_KEY_CODE
}

onBeforeUnmount(() => {
  stopListening()
})
</script>

<template>
  <section class="rounded-2xl border border-border/50 bg-muted/20 p-4">
    <div class="flex flex-col gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2 text-sm font-medium text-foreground">
          <Keyboard class="size-4" aria-hidden="true" />
          <span>{{ t("settings.curatedShortcutTitle") }}</span>
        </div>
        <p class="text-xs leading-5 text-muted-foreground">
          {{ t("settings.curatedShortcutBody") }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <div
          data-curated-shortcut-current
          class="inline-flex min-w-16 items-center justify-center rounded-lg border border-border bg-background px-3 py-1.5 font-mono text-sm font-semibold text-foreground shadow-xs"
        >
          {{ currentLabel }}
        </div>

        <Button
          type="button"
          variant="secondary"
          data-curated-shortcut-record
          :disabled="listening"
          @click="startListening"
        >
          {{ t("settings.curatedShortcutRecord") }}
        </Button>

        <Button
          type="button"
          variant="ghost"
          data-curated-shortcut-reset
          @click="resetShortcut"
        >
          <RotateCcw class="mr-2 size-4" aria-hidden="true" />
          {{ t("settings.curatedShortcutReset") }}
        </Button>
      </div>

      <p
        data-curated-shortcut-status
        class="text-xs text-muted-foreground"
      >
        {{ statusText }}
      </p>

      <p
        v-if="errorMessage"
        data-curated-shortcut-error
        class="text-xs text-destructive"
        role="alert"
      >
        {{ errorMessage }}
      </p>

      <div class="space-y-1">
        <p class="text-xs font-medium text-foreground">
          {{ t("settings.curatedShortcutReservedLabel") }}
        </p>
        <p class="text-xs leading-5 text-muted-foreground">
          {{ reservedLabels.join(" · ") }}
        </p>
      </div>
    </div>
  </section>
</template>
