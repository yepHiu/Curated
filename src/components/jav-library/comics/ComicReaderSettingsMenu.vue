<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { ComicReaderSettings } from "@/domain/comic/types"

const props = defineProps<{
  preferences: ComicReaderSettings
}>()

const { t } = useI18n()
</script>

<template>
  <div class="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl border border-border/70 bg-background/90 px-3 py-2 text-xs text-muted-foreground shadow-lg shadow-black/10 backdrop-blur">
    <span>{{ t("comics.readerMode") }}: {{ props.preferences.mode }}</span>
    <span>{{ t("comics.readerFit") }}: {{ props.preferences.fit }}</span>
    <span>{{ t("comics.readerDirection") }}: {{ props.preferences.direction }}</span>
  </div>
</template>
