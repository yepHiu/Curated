<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import HomeSectionRow from "@/components/jav-library/HomeSectionRow.vue"
import type { HomepageContinueEntry } from "@/lib/homepage-portal"

const props = defineProps<{
  entries: HomepageContinueEntry[]
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()

const { t } = useI18n()

const movies = computed(() => props.entries.map((entry) => entry.movie))
</script>

<template>
  <HomeSectionRow
    :title="t('home.sectionContinueTitle')"
    :movies="movies"
    @open-details="emit('openDetails', $event)"
    @open-player="emit('openPlayer', $event)"
  />
</template>
