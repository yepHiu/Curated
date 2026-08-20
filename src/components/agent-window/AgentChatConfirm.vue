<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import type { AgentChatEntry } from "./types"

const props = defineProps<{
  entry: Extract<AgentChatEntry, { kind: "confirm" }>
}>()

const emit = defineEmits<{
  apply: []
  discard: []
}>()

const { t } = useI18n()

function asText(value: unknown) {
  if (value == null) return ""
  return String(value)
}
</script>

<template>
  <div
    class="space-y-3 rounded-xl border border-border/60 bg-muted/30 px-3 py-3"
    data-agent-confirm-card
  >
    <p class="text-sm font-medium">{{ t("agentWindow.confirmTitle") }}</p>
    <div
      v-for="(change, index) in props.entry.changes"
      :key="`${change.path}-${index}`"
      class="grid gap-2 text-sm sm:grid-cols-2"
    >
      <div class="rounded-lg bg-background/80 p-2">
        <p class="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">
          {{ t("agentWindow.confirmBefore") }}
        </p>
        <p class="whitespace-pre-wrap text-muted-foreground">{{ asText(change.before) || t("agentWindow.confirmEmpty") }}</p>
      </div>
      <div class="rounded-lg bg-background p-2">
        <p class="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">
          {{ t("agentWindow.confirmAfter") }}
        </p>
        <p class="whitespace-pre-wrap">{{ asText(change.after) || t("agentWindow.confirmEmpty") }}</p>
      </div>
    </div>
    <p v-if="props.entry.status === 'applied'" class="text-sm text-muted-foreground">
      {{ t("agentWindow.confirmApplied") }}
    </p>
    <p v-else-if="props.entry.status === 'discarded'" class="text-sm text-muted-foreground">
      {{ t("agentWindow.confirmDiscarded") }}
    </p>
    <p v-else-if="props.entry.error" class="text-sm text-destructive">{{ props.entry.error }}</p>
    <div v-else class="flex flex-wrap gap-2">
      <Button
        type="button"
        class="min-h-11 rounded-full"
        :disabled="props.entry.status === 'applying'"
        data-agent-confirm-apply
        @click="emit('apply')"
      >
        {{ t("agentWindow.confirmApply") }}
      </Button>
      <Button
        type="button"
        variant="outline"
        class="min-h-11 rounded-full"
        :disabled="props.entry.status === 'applying'"
        data-agent-confirm-discard
        @click="emit('discard')"
      >
        {{ t("agentWindow.confirmDiscard") }}
      </Button>
    </div>
  </div>
</template>
