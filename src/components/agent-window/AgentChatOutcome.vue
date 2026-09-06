<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIChatOutcomeDTO } from "@/api/types"

const props = defineProps<{ outcome: AIChatOutcomeDTO }>()
const { t } = useI18n()

const label = computed(() => t(`agentWindow.outcome.${props.outcome.status}`))

const tone = computed(() => ({
  completed: "border-border/70 bg-muted/35",
  partial: "border-amber-500/35 bg-amber-500/8",
  needs_input: "border-amber-500/35 bg-amber-500/8",
  cancelled: "border-border/70 bg-muted/35",
  failed: "border-destructive/35 bg-destructive/10",
}[props.outcome.status]))
</script>

<template>
  <section class="rounded-xl border px-3 py-2 text-xs" :class="tone" data-agent-outcome>
    <p class="font-medium text-foreground">{{ label }}</p>
    <p v-if="outcome.reason" class="mt-1 leading-relaxed text-muted-foreground">{{ outcome.reason }}</p>
    <p v-if="outcome.retryable" class="mt-1 text-muted-foreground">{{ t("agentWindow.outcomeRetryable") }}</p>
  </section>
</template>
