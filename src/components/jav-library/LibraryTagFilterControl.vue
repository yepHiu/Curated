<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import LibraryFacetPickerPanel, {
  type FacetPickerGroup,
} from "@/components/jav-library/LibraryFacetPickerPanel.vue"
import type { TagCountEntry } from "@/lib/library-stats"

const props = defineProps<{
  metadataTags: readonly TagCountEntry[]
  userTags: readonly TagCountEntry[]
  selectedTags: readonly string[]
}>()

const emit = defineEmits<{
  clear: []
  toggleTag: [tag: string]
}>()

const { t } = useI18n()

const groups = computed((): FacetPickerGroup[] => {
  const next: FacetPickerGroup[] = []
  if (props.userTags.length > 0) {
    next.push({
      id: "user",
      label: t("library.savedViewTagUser"),
      ariaKey: "library.ariaFilterUserTag",
      rows: props.userTags.map((row) => ({ name: row.tag, count: row.count })),
    })
  }
  if (props.metadataTags.length > 0) {
    next.push({
      id: "metadata",
      label: t("library.savedViewTagMetadata"),
      ariaKey: "library.ariaFilterTag",
      rows: props.metadataTags.map((row) => ({ name: row.tag, count: row.count })),
    })
  }
  return next
})
</script>

<template>
  <LibraryFacetPickerPanel
    test-id="library-tag-filter"
    :title="t('library.savedViewTag')"
    :search-label="t('library.savedViewTagSearch')"
    :empty-label="t('library.savedViewTagEmpty')"
    :selected="selectedTags"
    :groups="groups"
    @clear="emit('clear')"
    @toggle="emit('toggleTag', $event)"
  />
</template>
