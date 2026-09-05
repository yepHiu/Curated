<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronRight, Loader2 } from "lucide-vue-next"
import { agentToolI18nKey, uniqueProcessToolNames } from "@/lib/agent-tool-labels"
import type { AgentChatEntry } from "./types"

const props = defineProps<{
  entry: Extract<AgentChatEntry, { kind: "process" }>
}>()

const emit = defineEmits<{
  toggle: []
}>()

const { t } = useI18n()

const visibleTools = computed(() => uniqueProcessToolNames(props.entry.tools.map((item) => item.name)))
const pendingTool = computed(() => [...props.entry.tools].reverse().find((item) => item.pending))
const failed = computed(() => props.entry.tools.some((item) => item.ok === false))
const busy = computed(() => props.entry.thinkingActive || Boolean(pendingTool.value))

const headline = computed(() => {
  if (pendingTool.value) {
    return t("agentWindow.toolRunningNamed", { name: t(agentToolI18nKey(pendingTool.value.name)) })
  }
  if (props.entry.thinkingActive) {
    return t("agentWindow.thinking")
  }
  if (failed.value) {
    return t("agentWindow.toolFailed")
  }
  if (props.entry.tools.some((item) => item.ok === undefined)) {
    return t("agentWindow.toolUnknown")
  }
  if (visibleTools.value.length > 0) {
    return t("agentWindow.toolOk")
  }
  return t("agentWindow.thought")
})
</script>

<template>
  <div class="text-xs text-muted-foreground" data-agent-process>
    <button
      type="button"
      class="flex min-h-8 w-full items-center gap-1.5 rounded-lg px-1 py-1 text-left hover:bg-muted/40"
      data-agent-process-toggle
      @click="emit('toggle')"
    >
      <Loader2
        v-if="busy"
        class="size-3.5 shrink-0 motion-safe:animate-spin"
        aria-hidden="true"
      />
      <ChevronRight
        v-else
        class="size-3.5 shrink-0 transition-transform"
        :class="entry.open ? 'rotate-90' : ''"
        aria-hidden="true"
      />
      <span class="min-w-0 truncate font-medium">{{ headline }}</span>
    </button>
    <div
      v-if="(busy && entry.thinking) || entry.open"
      class="mt-1 space-y-1.5 px-1 pb-1"
      data-agent-process-body
    >
      <p
        v-if="entry.thinking"
        class="whitespace-pre-wrap break-words leading-relaxed text-muted-foreground"
        data-agent-thinking
      >
        {{ entry.thinking }}
      </p>
      <p v-if="entry.open && !busy && visibleTools.length" class="leading-relaxed" data-agent-process-tools>
        {{ visibleTools.map((name) => t(agentToolI18nKey(name))).join(" · ") }}
      </p>
      <div v-if="entry.open && entry.tools.some((tool) => tool.evidence)" class="space-y-1.5" data-agent-evidence-cards>
        <div v-for="tool in entry.tools.filter((item) => item.evidence)" :key="tool.toolCallId" class="rounded-lg border border-border/60 bg-background/60 px-2 py-1.5">
          <p class="font-medium">{{ tool.evidence?.source === 'local' ? t('agentWindow.evidenceLocal') : tool.evidence?.source === 'provider' ? t('agentWindow.evidenceProvider') : t('agentWindow.evidenceSourcePage') }}</p>
          <p class="mt-0.5 break-words text-[11px] leading-relaxed">
            {{ tool.evidence?.failed ? t('agentWindow.evidenceFailed', { code: tool.evidence.errorCode || 'unknown' }) : tool.evidence?.truncated ? t('agentWindow.evidenceTruncated') : t('agentWindow.evidenceComplete') }}
            <template v-if="tool.evidence?.filters && Object.keys(tool.evidence.filters).length">{{ t('agentWindow.evidenceFilters') }}{{ Object.entries(tool.evidence.filters).map(([key, value]) => `${key}=${value}`).join(' · ') }}</template>
          </p>
        </div>
      </div>
      <div v-if="entry.open && entry.tools.some((tool) => tool.providerRows?.length)" class="space-y-1.5" data-agent-provider-rows>
        <div v-for="tool in entry.tools.filter((item) => item.providerRows?.length)" :key="`${tool.toolCallId}-provider`" class="space-y-1">
          <div v-for="row in tool.providerRows" :key="`${row.provider}-${row.code}-${row.title}`" class="rounded-lg border border-border/60 bg-background/60 px-2 py-1.5">
            <p class="font-medium">{{ row.code }}<span v-if="row.title"> · {{ row.title }}</span></p>
            <p class="mt-0.5 text-[11px] leading-relaxed">{{ row.provider || t('agentWindow.providerFallback') }}<span v-if="row.score"> · {{ t('agentWindow.providerScore', { score: row.score }) }}</span> · {{ row.inLibrary ? t('agentWindow.providerInLibrary') : t('agentWindow.providerOffLibrary') }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
