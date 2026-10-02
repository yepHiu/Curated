<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Sparkles } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { useTagOrganization, organizationProgressText, organizationProgressValue } from "@/composables/use-tag-organization"

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
const { t } = useI18n()
const state = useTagOrganization()
const label = computed(() => state.active.value ? organizationProgressText(state.active.value) : t("topics.historyCount", { count: state.jobs.value.length }))
</script>

<template>
  <section v-if="state.active.value || state.jobs.value.length" class="mb-2 min-w-0" data-sidebar-tag-organization>
    <Button v-if="props.compact" variant="ghost" size="icon" class="relative mx-auto flex size-11 rounded-lg" :title="label" :aria-label="`${t('topics.organize')}: ${label}`" @click="state.dialogOpen.value = true">
      <Sparkles />
      <Progress v-if="state.active.value" :model-value="organizationProgressValue(state.active.value)" class="absolute inset-x-1.5 bottom-1 h-0.5 w-auto" aria-hidden="true" />
    </Button>
    <button v-else type="button" class="flex min-h-11 w-full min-w-0 flex-col gap-2 rounded-lg border border-border/60 bg-background/45 px-3 py-2.5 text-left transition-colors hover:bg-sidebar-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60" :aria-label="`${t('topics.organize')}: ${label}`" @click="state.dialogOpen.value = true">
      <span class="flex min-w-0 items-center justify-between gap-2 text-xs">
        <span class="flex min-w-0 items-center gap-2 font-medium"><Sparkles class="size-3.5 shrink-0 text-primary" aria-hidden="true" /><span class="truncate">{{ t("topics.organize") }}</span></span>
        <span v-if="state.active.value" class="shrink-0 tabular-nums text-muted-foreground">{{ Math.round(organizationProgressValue(state.active.value)) }}%</span>
      </span>
      <span class="truncate text-xs text-muted-foreground">{{ state.connected.value ? label : t("topics.disconnected") }}</span>
      <Progress v-if="state.active.value" :model-value="organizationProgressValue(state.active.value)" class="h-1" :aria-label="label" />
      <span v-if="state.active.value" class="text-xs text-muted-foreground">{{ t("topics.historyCount", { count: state.jobs.value.length }) }}</span>
    </button>
  </section>
</template>
