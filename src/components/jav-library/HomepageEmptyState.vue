<script setup lang="ts">
import { useI18n } from "vue-i18n"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"

const { t } = useI18n()
</script>

<template>
  <div data-homepage-empty-state class="h-full min-h-0 overflow-y-auto px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]">
    <MediaEmptyState :description="t('home.emptyBody')" />
  </div>
</template>
