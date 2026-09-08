<script setup lang="ts">
import { ref } from "vue"
import { Loader2, Settings } from "lucide-vue-next"
import { RouterLink } from "vue-router"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import AgentChatMovieCard from "./AgentChatMovieCard.vue"
import AgentChatProcess from "./AgentChatProcess.vue"
import AgentChatConfirm from "./AgentChatConfirm.vue"
import AgentMarkdown from "./AgentMarkdown.vue"
import AgentChatResolution from "./AgentChatResolution.vue"
import AgentChatOutcome from "./AgentChatOutcome.vue"
import type { AgentChatEntry } from "./types"
import { confirmationOutcome } from "./confirmation-outcome"

defineProps<{
  entries: AgentChatEntry[]
  providerUnconfigured: boolean
  errorMessage: string
  wide?: boolean
  hasOlder?: boolean
  loadingOlder?: boolean
  historyDisabled?: boolean
}>()

const emit = defineEmits<{
  loadOlder: []
  close: []
  openMovie: [movieId: string]
  applyConfirm: [id: string]
  discardConfirm: [id: string]
  selectEntity: [entryId: string, candidate: import("@/api/types").AIEntityCandidateDTO]
}>()

const listRef = ref<HTMLElement | null>(null)
const { t } = useI18n()

defineExpose({
  captureAnchor() {
    const element = listRef.value
    const height = element?.scrollHeight ?? 0
    const top = element?.scrollTop ?? 0
    return () => { if (element) element.scrollTop = top + element.scrollHeight - height }
  },
  scrollToEnd() {
    if (listRef.value) {
      listRef.value.scrollTop = listRef.value.scrollHeight
    }
  },
})
</script>

<template>
  <div
    ref="listRef"
    class="agent-window-messages-scroller min-h-0 flex-1 overflow-y-auto overscroll-contain"
    data-agent-window-messages
    aria-live="polite"
  >
    <div
      class="mx-auto w-full space-y-5 py-4"
      :class="wide ? 'max-w-[52rem] px-6' : 'px-4'"
      data-agent-window-messages-inner
    >
    <Button v-if="hasOlder" variant="ghost" size="sm" :disabled="loadingOlder || historyDisabled" data-agent-load-older @click="emit('loadOlder')">
      {{ t(loadingOlder ? 'agentWindow.historyLoading' : 'agentWindow.historyOlder') }}
    </Button>
    <p
      v-if="entries.length === 0 && !providerUnconfigured && !errorMessage"
      class="px-1 py-10 text-center text-sm text-muted-foreground"
    >
      {{ t("agentWindow.emptyHint") }}
    </p>
    <template v-for="(entry, index) in entries" :key="entry.id">
      <AgentChatProcess
        v-if="entry.kind === 'process' && (entry.thinkingActive || entry.thinking || entry.tools.length)"
        :entry="entry"
        @toggle="entry.open = !entry.open"
      />
      <div v-else-if="entry.kind === 'user'" class="flex justify-end" data-agent-entry="user">
        <div
          class="max-w-[min(85%,28rem)] rounded-2xl bg-secondary px-3.5 py-2 text-secondary-foreground"
          data-agent-user-bubble
        >
          <AgentMarkdown class="text-[15px]" :source="entry.content" />
        </div>
      </div>
      <div v-else-if="entry.kind === 'assistant'" class="space-y-3 text-sm text-foreground" data-agent-entry="assistant">
        <AgentMarkdown v-if="entry.content" :source="entry.content" />
        <Loader2
          v-else-if="!entry.movies?.length && entries[index - 1]?.kind !== 'process'"
          class="size-4 text-muted-foreground motion-safe:animate-spin"
          aria-hidden="true"
        />
        <div v-if="entry.kind === 'assistant' && entry.movies?.length" class="space-y-2" data-agent-movie-slate>
          <AgentChatMovieCard
            v-for="movie in entry.movies"
            :key="movie.movieId"
            :movie="movie"
            @open="emit('openMovie', $event)"
          />
        </div>
      </div>
      <AgentChatConfirm
        v-else-if="entry.kind === 'confirm'"
        :entry="entry"
        @apply="emit('applyConfirm', entry.id)"
        @discard="emit('discardConfirm', entry.id)"
        @open-movie="emit('openMovie', $event)"
      />
      <AgentChatResolution
        v-else-if="entry.kind === 'resolution'"
        :resolution="entry.resolution"
        :selected="entry.selected"
        @select="emit('selectEntity', entry.id, $event)"
      />
      <AgentChatOutcome v-else-if="entry.kind === 'outcome'" :outcome="confirmationOutcome(entries, index, entry.outcome)" />
    </template>

    <div
      v-if="providerUnconfigured"
      class="flex flex-col gap-2 rounded-xl border border-border/60 bg-muted/40 px-3 py-2.5 text-sm"
      data-agent-window-unconfigured
    >
      <p class="leading-relaxed text-muted-foreground">{{ t("agentWindow.unconfigured") }}</p>
      <Button as-child variant="outline" size="sm" class="w-fit rounded-full">
        <RouterLink :to="{ name: 'settings', query: { section: 'ai' } }" @click="emit('close')">
          <Settings data-icon="inline-start" class="size-4" />
          {{ t("agentWindow.openSettings") }}
        </RouterLink>
      </Button>
    </div>
    <p
      v-else-if="errorMessage"
      class="rounded-xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm leading-relaxed text-destructive"
      data-agent-window-error
    >
      {{ t("agentWindow.errorPrefix") }}{{ errorMessage }}
    </p>
    </div>
  </div>
</template>

<style scoped>
@reference "../../style.css";

.agent-window-messages-scroller {
  scrollbar-width: thin;
  scrollbar-color: color-mix(in oklab, var(--muted-foreground) 42%, transparent) transparent;
}

.agent-window-messages-scroller::-webkit-scrollbar {
  width: 8px;
}

.agent-window-messages-scroller::-webkit-scrollbar-track {
  margin-block: 10px;
  background: transparent;
}

.agent-window-messages-scroller::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background-color: color-mix(in oklab, var(--muted-foreground) 38%, transparent);
  background-clip: padding-box;
}

.agent-window-messages-scroller::-webkit-scrollbar-thumb:hover {
  background-color: color-mix(in oklab, var(--muted-foreground) 58%, transparent);
}
</style>
