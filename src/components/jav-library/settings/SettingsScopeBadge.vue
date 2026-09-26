<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Badge } from "@/components/ui/badge"

const props = defineProps<{ scope: "server" | "desktop" | "client" | "mixed" }>()
const { t } = useI18n()
// Client preferences live in this renderer's storage, including its server-origin partition.
const isDesktop = typeof window !== "undefined" && Boolean(window.javLibrary)
const scopes = computed(() => props.scope === "mixed" ? ["server", "client"] as const : [props.scope])
function label(scope: string) {
  return t(`settings.scope.${scope === "client" ? (isDesktop ? "desktop" : "browser") : scope}`)
}
</script>

<template>
  <span class="inline-flex max-w-full flex-wrap items-center gap-1" data-settings-scope>
    <Badge
      v-for="item in scopes"
      :key="item"
      variant="outline"
      class="border-border/60 bg-muted/20 px-1.5 py-0 text-[10px] font-medium leading-5 tracking-normal text-muted-foreground"
      :data-scope="item"
      :title="t(`settings.scope.${item}Hint`)"
      :aria-label="`${label(item)}: ${t(`settings.scope.${item}Hint`)}`"
    >{{ label(item) }}</Badge>
  </span>
</template>
