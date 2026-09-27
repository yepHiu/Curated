<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Badge } from "@/components/ui/badge"
import { useCurrentLocalDayKey } from "@/lib/current-local-day-key"
import { getMovieReleaseStatus } from "@/lib/movie-release-status"

const props = defineProps<{ releaseDate?: string }>()
const { t } = useI18n()
const today = useCurrentLocalDayKey()
const status = computed(() => getMovieReleaseStatus(props.releaseDate, today.value))
</script>

<template>
  <Badge
    v-if="status"
    :variant="status === 'released' ? 'success' : 'secondary'"
    :data-release-status="status"
  >
    {{ t(`detailPanel.releaseStatus.${status}`) }}
  </Badge>
</template>
