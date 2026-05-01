<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Camera } from "lucide-vue-next"
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
  <div
    class="flex flex-col items-center justify-center gap-3 rounded-3xl border border-dashed border-border/70 bg-muted/20 text-center"
    :class="isLibraryEmpty ? 'py-16' : 'mb-4 py-12'"
  >
    <Camera
      class="text-muted-foreground"
      :class="isLibraryEmpty ? 'size-12' : 'size-10'"
    />
    <p class="text-sm text-muted-foreground">
      {{ isLibraryEmpty ? t("curated.empty") : t("curated.tagFilterNoMatches") }}
    </p>
    <Button
      v-if="!isLibraryEmpty && showClearFilter"
      type="button"
      variant="outline"
      size="sm"
      class="rounded-2xl"
      @click="emit('clearFilter')"
    >
      {{ t("curated.tagFilterAll") }}
    </Button>
  </div>
</template>
