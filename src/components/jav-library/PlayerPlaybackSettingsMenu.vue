<script setup lang="ts">
import { computed } from "vue"
import { Settings2 } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type PlaybackModeOption = "direct" | "hls"

const props = defineProps<{
  disabled?: boolean
  playbackRate: number
  playbackMode: PlaybackModeOption
  canSwitchToDirect?: boolean
  switchingMode?: boolean
}>()

const emit = defineEmits<{
  "update:playbackRate": [value: number]
  "update:playbackMode": [value: PlaybackModeOption]
}>()

const { t } = useI18n()

const playbackRates = [0.75, 1, 1.25, 1.5, 2]

const radioItemClass = [
  "rounded-lg text-white/90 transition-colors",
  "focus:bg-white/12 focus:text-white",
  "data-[state=checked]:bg-primary/18 data-[state=checked]:font-medium data-[state=checked]:text-primary",
  "data-[state=checked]:focus:bg-primary/22 data-[state=checked]:focus:text-primary data-[disabled]:opacity-45",
].join(" ")

const directUnavailable = computed(() => props.canSwitchToDirect === false)

function formatRateLabel(value: number): string {
  return Number.isInteger(value) ? `${value}x` : `${value.toFixed(2).replace(/0+$/, "").replace(/\.$/, "")}x`
}

function selectPlaybackRate(value: number) {
  if (!Number.isFinite(value) || value <= 0 || value === props.playbackRate) return
  emit("update:playbackRate", value)
}

function selectPlaybackMode(value: PlaybackModeOption) {
  if (value === props.playbackMode) return
  emit("update:playbackMode", value)
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        variant="secondary"
        size="icon"
        class="size-9 shrink-0 rounded-full border-white/10 bg-white/8 text-white backdrop-blur-md hover:bg-white/14"
        :disabled="disabled"
        :aria-label="t('player.playbackSettingsAria')"
      >
        <Settings2 class="size-4 shrink-0" aria-hidden="true" />
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent
      align="end"
      class="w-64 rounded-2xl border-white/18 bg-neutral-950/95 text-white shadow-[0_24px_70px_rgba(0,0,0,0.62)] backdrop-blur-xl backdrop-saturate-150"
      @click.stop
    >
      <DropdownMenuLabel class="text-xs font-semibold uppercase tracking-[0.08em] text-white/70">
        {{ t("player.playbackSettings") }}
      </DropdownMenuLabel>

      <DropdownMenuSeparator class="bg-white/10" />

      <div class="px-2 pb-1 pt-1.5">
        <p class="mb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-white/58">
          {{ t("player.playbackSpeed") }}
        </p>
        <DropdownMenuGroup>
          <DropdownMenuRadioGroup :model-value="String(playbackRate)">
            <DropdownMenuRadioItem
              v-for="rate in playbackRates"
              :key="rate"
              :value="String(rate)"
              :class="radioItemClass"
              @select="selectPlaybackRate(rate)"
            >
              {{ formatRateLabel(rate) }}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </div>

      <DropdownMenuSeparator class="bg-white/10" />

      <div class="px-2 pb-2 pt-1.5">
        <p class="mb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-white/58">
          {{ t("player.playbackMode") }}
        </p>
        <DropdownMenuGroup>
          <DropdownMenuRadioGroup :model-value="playbackMode">
            <DropdownMenuRadioItem
              value="direct"
              :disabled="directUnavailable || switchingMode"
              :class="radioItemClass"
              @select="selectPlaybackMode('direct')"
            >
              <div class="flex min-w-0 flex-col gap-0.5">
                <span>{{ t("player.playbackModeDirect") }}</span>
                <span v-if="directUnavailable" class="text-xs text-white/55">
                  {{ t("player.playbackModeDirectUnavailable") }}
                </span>
              </div>
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem
              value="hls"
              :disabled="switchingMode"
              :class="radioItemClass"
              @select="selectPlaybackMode('hls')"
            >
              {{ t("player.playbackModeHls") }}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
