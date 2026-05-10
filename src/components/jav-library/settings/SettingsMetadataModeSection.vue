<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { CircleHelp } from "lucide-vue-next"
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from "reka-ui"

type MetadataMovieMode = "auto" | "specified" | "chain"

defineProps<{
  metadataMovieModeUi: MetadataMovieMode
  metadataMovieSaving: boolean
  metadataMovieChainSaving: boolean
  providerPingAllBusy: boolean
  canPickSpecifiedMetadata: boolean
  canUseMetadataChainMode: boolean
}>()

const emit = defineEmits<{
  selectAuto: []
  selectSpecified: []
  selectChain: []
}>()

const { t } = useI18n()
</script>

<template>
  <fieldset
    class="flex flex-col gap-3 rounded-2xl border border-border/50 bg-muted/[0.11] p-4 dark:bg-muted/10"
    :aria-busy="metadataMovieSaving || metadataMovieChainSaving || providerPingAllBusy"
  >
    <legend class="sr-only">{{ t("settings.metadataMovieProviderMode") }}</legend>
    <div class="mb-0.5 flex items-center gap-3">
      <span class="text-sm font-semibold text-foreground">{{
        t("settings.metadataMovieProviderMode")
      }}</span>
      <TooltipProvider :delay-duration="280">
        <TooltipRoot>
          <TooltipTrigger as-child>
            <button
              type="button"
              class="inline-flex size-8 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              <CircleHelp class="size-4" aria-hidden="true" />
              <span class="sr-only">{{
                t("settings.metadataMovieProviderModeAria")
              }}</span>
            </button>
          </TooltipTrigger>
          <TooltipPortal>
            <TooltipContent
              side="top"
              :side-offset="6"
              class="z-50 max-w-[min(22rem,calc(100vw-2rem))] rounded-xl border border-border/50 bg-popover px-3 py-2 text-xs leading-relaxed text-pretty text-popover-foreground shadow-lg"
            >
              {{ t("settings.metadataMovieProviderModeTooltip") }}
            </TooltipContent>
          </TooltipPortal>
        </TooltipRoot>
      </TooltipProvider>
    </div>
    <label
      class="flex cursor-pointer items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]"
    >
      <input
        class="mt-0.5 size-4 shrink-0 accent-primary"
        type="radio"
        name="metadata-movie-mode"
        value="auto"
        :disabled="metadataMovieSaving"
        :checked="metadataMovieModeUi === 'auto'"
        @change="emit('selectAuto')"
      />
      <span class="min-w-0 flex-1">
        <span class="text-sm font-medium">{{ t("settings.metadataMovieProviderAuto") }}</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
          {{ t("settings.metadataMovieProviderAutoHint") }}
        </span>
      </span>
    </label>
    <label
      class="flex items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors"
      :class="
        canPickSpecifiedMetadata
          ? 'cursor-pointer hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]'
          : 'cursor-not-allowed opacity-60'
      "
    >
      <input
        class="mt-0.5 size-4 shrink-0 accent-primary"
        type="radio"
        name="metadata-movie-mode"
        value="specified"
        :checked="metadataMovieModeUi === 'specified'"
        :disabled="metadataMovieSaving || !canPickSpecifiedMetadata"
        @change="emit('selectSpecified')"
      />
      <span class="min-w-0 flex-1">
        <span class="text-sm font-medium">{{
          t("settings.metadataMovieProviderSpecified")
        }}</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
          {{ t("settings.metadataMovieProviderSpecifiedHint") }}
        </span>
      </span>
    </label>
    <label
      class="flex items-start gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors"
      :class="
        canUseMetadataChainMode
          ? 'cursor-pointer hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]'
          : 'cursor-not-allowed opacity-60'
      "
    >
      <input
        class="mt-0.5 size-4 shrink-0 accent-primary"
        type="radio"
        name="metadata-movie-mode"
        value="chain"
        :checked="metadataMovieModeUi === 'chain'"
        :disabled="metadataMovieSaving || metadataMovieChainSaving || !canUseMetadataChainMode"
        @change="emit('selectChain')"
      />
      <span class="min-w-0 flex-1">
        <span class="text-sm font-medium">{{ t("settings.metadataMovieProviderChain") }}</span>
        <span class="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
          {{ t("settings.metadataMovieProviderChainHint") }}
        </span>
      </span>
    </label>
  </fieldset>
</template>
