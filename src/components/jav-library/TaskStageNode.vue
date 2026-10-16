<script setup lang="ts">
import { cn } from "@/lib/utils"

export type TaskStageNodeState = "pending" | "running" | "success" | "error" | "paused"
defineProps<{ state: TaskStageNodeState }>()
</script>

<template>
  <span
    :class="cn('task-stage-node relative inline-flex size-7 shrink-0 items-center justify-center rounded-full border-2', {
      'border-border text-muted-foreground': state === 'pending',
      'border-primary/60 text-primary': state === 'running',
      'border-success/70 text-success': state === 'success',
      'border-destructive/70 text-destructive': state === 'error',
      'border-muted-foreground/60 text-muted-foreground': state === 'paused',
    })"
    :data-node-state="state"
    aria-hidden="true"
  >
    <Transition name="task-node">
      <svg :key="state" class="absolute size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
        <g v-if="state === 'running'" class="task-spinner">
          <circle class="task-spinner-arc" cx="12" cy="12" r="8" pathLength="100" />
        </g>
        <path v-else-if="state === 'success'" class="task-mark" pathLength="1" d="m5 12 4.5 4.5L19 7" />
        <g v-else-if="state === 'error'">
          <path class="task-mark" pathLength="1" d="m7 7 10 10" />
          <path class="task-mark task-mark-second" pathLength="1" d="M17 7 7 17" />
        </g>
        <path v-else-if="state === 'paused'" d="M9 8v8m6-8v8" />
        <circle v-else cx="12" cy="12" r="2" fill="currentColor" stroke="none" />
      </svg>
    </Transition>
  </span>
</template>

<style scoped>
.task-stage-node { transition: color 240ms ease, border-color 240ms ease; }
.task-spinner { transform-origin: 12px 12px; animation: task-spin 2s linear infinite; }
.task-spinner-arc { stroke-dasharray: 18 82; stroke-dashoffset: 0; animation: task-arc 1.5s ease-in-out infinite; }
@keyframes task-spin { to { transform: rotate(360deg); } }
@keyframes task-arc {
  0% { stroke-dasharray: 18 82; stroke-dashoffset: 0; }
  50% { stroke-dasharray: 72 28; stroke-dashoffset: -18; }
  100% { stroke-dasharray: 18 82; stroke-dashoffset: -100; }
}
.task-node-enter-active { transition: opacity 240ms ease, transform 440ms cubic-bezier(.2, .8, .2, 1); }
.task-node-leave-active { transition: opacity 160ms ease, transform 160ms ease; }
.task-node-enter-from { opacity: 0; transform: scale(.65); }
.task-node-leave-to { opacity: 0; transform: scale(.8); }
.task-node-enter-active .task-mark { stroke-dasharray: 1; animation: task-draw 360ms ease-out both; }
.task-node-enter-active .task-mark-second { animation-delay: 70ms; }
@keyframes task-draw { from { stroke-dashoffset: 1; } to { stroke-dashoffset: 0; } }
@media (prefers-reduced-motion: reduce) {
  .task-stage-node, .task-node-enter-active, .task-node-leave-active { transition: none; }
  .task-node-enter-active .task-mark, .task-spinner, .task-spinner-arc { animation: none; }
  .task-spinner-arc { stroke-dasharray: 65 35; }
}
</style>
