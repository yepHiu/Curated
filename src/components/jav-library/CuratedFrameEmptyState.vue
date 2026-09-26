<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  variant: "library" | "filtered"
  showClearFilter: boolean
}>()

const emit = defineEmits<{
  clearFilter: []
}>()

const { t } = useI18n()

const isLibraryEmpty = computed(() => props.variant === "library")
</script>

<template>
  <MediaEmptyState :filtered="!isLibraryEmpty" :description="t('curated.empty')">
    <template v-if="!isLibraryEmpty && showClearFilter" #default>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="min-h-11 rounded-full sm:min-h-8"
        @click="emit('clearFilter')"
      >
        {{ t("curated.tagFilterAll") }}
      </Button>
    </template>
  </MediaEmptyState>
</template>
