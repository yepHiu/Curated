<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIChatOutcomeDTO } from "@/api/types"

const props = defineProps<{ outcome: AIChatOutcomeDTO }>()
const { t } = useI18n()

const reasonKeys: Record<string, string> = {
  tool_step_limit: "agentWindow.outcomeReasons.toolStepLimit",
  confirmation_required: "agentWindow.outcomeReasons.confirmationRequired",
  write_applied: "agentWindow.outcomeReasons.writeApplied",
  confirmation_unavailable: "agentWindow.outcomeReasons.confirmationUnavailable",
  confirmation_discarded: "agentWindow.outcomeReasons.confirmationDiscarded",
}
const label = computed(() => t(props.outcome.reasonCode === "write_applied" ? "agentWindow.outcomeSaved" : `agentWindow.outcome.${props.outcome.status}`))
const reason = computed(() => {
  const key = reasonKeys[props.outcome.reasonCode ?? ""]
  return key ? t(key) : props.outcome.reason
})

const tone = computed(() => ({
  completed: "border-border/70 bg-muted/35",
  partial: "border-warning/35 bg-warning/10",
  needs_input: "border-warning/35 bg-warning/10",
  needs_confirmation: "border-info/35 bg-info/10",
  cancelled: "border-border/70 bg-muted/35",
  failed: "border-destructive/35 bg-destructive/10",
}[props.outcome.status]))
</script>

<template>
  <section class="rounded-xl border px-3 py-2 text-xs" :class="tone" data-agent-outcome role="status" aria-live="polite">
    <p class="font-medium text-foreground">{{ label }}</p>
    <p v-if="reason" class="mt-1 leading-relaxed text-muted-foreground">{{ reason }}</p>
    <p v-if="outcome.retryable && outcome.reasonCode !== 'tool_step_limit'" class="mt-1 text-muted-foreground">{{ t("agentWindow.outcomeRetryable") }}</p>
  </section>
</template>
