<script setup lang="ts">
import { computed } from "vue"
import { renderAgentMarkdown } from "@/lib/agent-markdown"
import { cn } from "@/lib/utils"

const props = defineProps<{
  source: string
  class?: string
}>()

const html = computed(() => renderAgentMarkdown(props.source))
</script>

<template>
  <div
    class="agent-md break-words"
    :class="cn(props.class)"
    data-agent-markdown
    v-html="html"
  />
</template>

<style scoped>
@reference "../../style.css";

.agent-md :deep(> :first-child) {
  margin-top: 0;
}
.agent-md :deep(> :last-child) {
  margin-bottom: 0;
}
.agent-md :deep(p) {
  @apply my-2 leading-relaxed;
}
.agent-md :deep(h1),
.agent-md :deep(h2),
.agent-md :deep(h3),
.agent-md :deep(h4) {
  @apply mt-3 mb-1 font-semibold tracking-tight text-foreground;
}
.agent-md :deep(h1) {
  @apply text-base;
}
.agent-md :deep(h2) {
  @apply text-sm;
}
.agent-md :deep(h3),
.agent-md :deep(h4) {
  @apply text-sm;
}
.agent-md :deep(ul) {
  @apply my-2 list-disc pl-5;
}
.agent-md :deep(ol) {
  @apply my-2 list-decimal pl-5;
}
.agent-md :deep(li) {
  @apply my-0.5 leading-relaxed;
}
.agent-md :deep(li > ul),
.agent-md :deep(li > ol) {
  @apply my-1;
}
.agent-md :deep(blockquote) {
  @apply my-2 border-l-2 border-border pl-3 text-muted-foreground;
}
.agent-md :deep(hr) {
  @apply my-3 border-border;
}
.agent-md :deep(pre) {
  @apply my-2 overflow-x-auto rounded-lg border border-border/60 bg-muted/40 p-3 font-mono text-[13px] leading-relaxed;
}
.agent-md :deep(:not(pre) > code) {
  @apply rounded-md bg-muted px-1 py-0.5 font-mono text-[13px];
}
.agent-md :deep(pre code) {
  @apply bg-transparent p-0 text-[13px];
}
.agent-md :deep(a) {
  @apply text-primary underline-offset-2 hover:underline;
}
.agent-md :deep(table) {
  @apply my-2 w-full border-collapse text-sm;
}
.agent-md :deep(th),
.agent-md :deep(td) {
  @apply border border-border/60 px-2 py-1 text-left;
}
.agent-md :deep(th) {
  @apply bg-muted/40 font-medium;
}
.agent-md :deep(strong) {
  @apply font-semibold;
}
.agent-md :deep(input[type="checkbox"]) {
  @apply mr-1.5 align-middle;
}
</style>
