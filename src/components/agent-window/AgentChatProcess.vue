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
    </div>
  </div>
</template>
