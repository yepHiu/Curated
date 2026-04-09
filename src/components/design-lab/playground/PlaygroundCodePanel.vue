<script setup lang="ts">
import type { PlaygroundCodeOutput } from "@/lib/design-lab/playground-definitions"

defineProps<{
  output: PlaygroundCodeOutput
}>()
</script>

<template>
  <div class="rounded-2xl border border-border/70 bg-surface-elevated p-4 lg:p-5">
    <div class="mb-4">
      <h3 class="text-sm font-semibold text-foreground">Generated Code</h3>
      <p class="text-xs leading-5 text-muted-foreground">
        Snippets should be directly reusable in Vue templates and grounded in current token usage.
      </p>
    </div>

    <pre class="overflow-x-auto rounded-2xl border border-border/70 bg-background/90 p-4 text-xs leading-6 text-foreground"><code>{{ output.vueSnippet }}</code></pre>

    <div class="mt-4 space-y-2">
      <h4 class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Token Notes</h4>
      <ul class="space-y-2 text-sm leading-6 text-muted-foreground">
        <li v-for="note in output.tokenNotes" :key="note">
          {{ note }}
        </li>
      </ul>
    </div>
  </div>
</template>
