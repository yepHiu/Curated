<script setup lang="ts">
import { useI18n } from "vue-i18n"

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
    <div class="flex flex-col gap-3">
      <span class="text-sm font-semibold text-foreground">{{
        t("settings.metadataMovieProviderMode")
      }}</span>
      <p
        class="text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
      >
        {{ t("settings.metadataMovieProviderModeTooltip") }}
      </p>
    </div>
    <label
      class="flex cursor-pointer items-center gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]"
    >
      <input
        class="size-4 shrink-0 accent-primary"
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
      class="flex items-center gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors"
      :class="
        canPickSpecifiedMetadata
          ? 'cursor-pointer hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]'
          : 'cursor-not-allowed opacity-60'
      "
    >
      <input
        class="size-4 shrink-0 accent-primary"
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
      class="flex items-center gap-3 rounded-xl border border-transparent px-3 py-2 transition-colors"
      :class="
        canUseMetadataChainMode
          ? 'cursor-pointer hover:bg-muted/35 has-[:checked]:border-primary/40 has-[:checked]:bg-primary/[0.06]'
          : 'cursor-not-allowed opacity-60'
      "
    >
      <input
        class="size-4 shrink-0 accent-primary"
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
