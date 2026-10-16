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
        <g v-if="state === 'running'" class="task-orbit" stroke="none" fill="currentColor">
          <circle v-for="n in 5" :key="n" class="task-orbit-dot" cx="12" cy="4" r="1.55" />
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
.task-orbit-dot { transform-origin: 12px 12px; animation: task-orbit 1.8s cubic-bezier(.35, 0, .65, 1) infinite; }
.task-orbit-dot:nth-child(2) { animation-delay: -120ms; }
.task-orbit-dot:nth-child(3) { animation-delay: -240ms; }
.task-orbit-dot:nth-child(4) { animation-delay: -360ms; }
.task-orbit-dot:nth-child(5) { animation-delay: -480ms; }
@keyframes task-orbit {
  0% { transform: rotate(0deg); opacity: .2; }
  20%, 75% { opacity: 1; }
  100% { transform: rotate(360deg); opacity: .2; }
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
  .task-node-enter-active .task-mark, .task-orbit-dot { animation: none; }
  .task-orbit-dot:nth-child(2) { transform: rotate(45deg); opacity: .8; }
  .task-orbit-dot:nth-child(3) { transform: rotate(90deg); opacity: .6; }
  .task-orbit-dot:nth-child(4) { transform: rotate(135deg); opacity: .4; }
  .task-orbit-dot:nth-child(5) { transform: rotate(180deg); opacity: .2; }
}
</style>
