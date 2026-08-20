<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Plus, Trash2 } from "lucide-vue-next"
import type { AIChatSessionDTO } from "@/api/types"
import { groupAgentSessions, type AgentSessionGroupKey } from "@/lib/agent-session-groups"

const props = defineProps<{
  sessions: AIChatSessionDTO[]
  activeId: string
}>()

const emit = defineEmits<{
  create: []
  select: [id: string]
  delete: [id: string]
}>()

const { t } = useI18n()
const groups = computed(() => groupAgentSessions(props.sessions))

function groupLabel(key: AgentSessionGroupKey) {
  return t(`agentWindow.group.${key}`)
}

function sessionTitle(session: AIChatSessionDTO) {
  const title = session.title?.trim()
  return title || t("agentWindow.sessionPlaceholder")
}
</script>

<template>
  <aside
    class="flex h-full w-[220px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
    data-agent-window-sidebar
  >
    <div class="p-2">
      <button
        type="button"
        class="flex min-h-11 w-full items-center gap-2 rounded-lg px-2 text-sm text-sidebar-foreground hover:bg-sidebar-accent md:min-h-9"
        data-agent-window-new
        @click="emit('create')"
      >
        <Plus class="size-4 shrink-0" />
        {{ t("agentWindow.newChat") }}
      </button>
    </div>
    <div
      class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-2 pb-2"
      data-agent-window-sessions
    >
      <section v-for="group in groups" :key="group.key" class="mb-2">
        <p class="px-2 pt-2 pb-1 text-[11px] font-medium text-muted-foreground">
          {{ groupLabel(group.key) }}
        </p>
        <div class="flex flex-col gap-0.5">
          <div
            v-for="session in group.items"
            :key="session.id"
            class="group relative flex items-center"
          >
            <button
              type="button"
              class="flex min-h-11 min-w-0 flex-1 items-center rounded-lg px-2 pr-9 text-left text-[13px] leading-snug md:min-h-8"
              :class="
                session.id === activeId
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                  : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground'
              "
              :data-agent-window-session="session.id"
              @click="emit('select', session.id)"
            >
              <span class="min-w-0 flex-1 truncate">{{ sessionTitle(session) }}</span>
            </button>
            <button
              type="button"
              class="absolute right-0.5 flex size-9 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-destructive md:size-7 md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100"
              :aria-label="t('agentWindow.deleteChat')"
              :data-agent-window-delete="session.id"
              @click.stop="emit('delete', session.id)"
            >
              <Trash2 class="size-3.5" />
            </button>
          </div>
        </div>
      </section>
    </div>
  </aside>
</template>
