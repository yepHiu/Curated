<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import type { AIEntityCandidateDTO, AIEntityResolutionDTO } from "@/api/types"

const props = defineProps<{
  resolution: AIEntityResolutionDTO
  selected?: boolean
}>()

const emit = defineEmits<{
  select: [candidate: AIEntityCandidateDTO]
}>()
const { t } = useI18n()

const title = computed(() => {
  if (props.resolution.status === "ambiguous") return t("agentWindow.resolutionAmbiguous")
  if (props.resolution.status === "unmatched") return t("agentWindow.resolutionUnmatched")
  return t("agentWindow.resolutionMatched")
})

function label(candidate: AIEntityCandidateDTO) {
  if (candidate.kind === "movie") {
    return [candidate.code, candidate.title].filter(Boolean).join(" · ") || candidate.movieId || t("agentWindow.localMovie")
  }
  return candidate.actorName || t("agentWindow.localActor")
}
</script>

<template>
  <section class="rounded-xl border border-amber-500/35 bg-amber-500/8 px-3 py-2.5 text-sm" data-agent-entity-resolution>
    <p class="font-medium text-foreground">{{ title }}</p>
    <p v-if="resolution.reason" class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ resolution.reason }}</p>
    <div v-if="resolution.candidates.length" class="mt-2 flex flex-wrap gap-2">
      <Button
        v-for="candidate in resolution.candidates"
        :key="candidate.movieId || candidate.actorName"
        type="button"
        variant="outline"
        size="sm"
        class="max-w-full rounded-full"
        :disabled="selected || resolution.status !== 'ambiguous'"
        :data-agent-entity-candidate="candidate.movieId || candidate.actorName"
        @click="emit('select', candidate)"
      >
        <span class="truncate">{{ label(candidate) }}</span>
      </Button>
    </div>
  </section>
</template>
