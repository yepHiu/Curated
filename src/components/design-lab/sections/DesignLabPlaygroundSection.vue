<script setup lang="ts">
import { computed, reactive } from "vue"
import PlaygroundCodePanel from "@/components/design-lab/playground/PlaygroundCodePanel.vue"
import PlaygroundInspector from "@/components/design-lab/playground/PlaygroundInspector.vue"
import PlaygroundPreviewCanvas from "@/components/design-lab/playground/PlaygroundPreviewCanvas.vue"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  createDefaultDesignLabPlaygroundState,
  renderPlaygroundOutput,
} from "@/lib/design-lab/playground-definitions"

const state = reactive(createDefaultDesignLabPlaygroundState())
const output = computed(() => renderPlaygroundOutput(state))

function resetState() {
  Object.assign(state, createDefaultDesignLabPlaygroundState())
}
</script>

<template>
  <section id="playground" class="scroll-mt-24">
    <Card class="rounded-2xl border border-border bg-card shadow-sm">
      <CardHeader>
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-1">
            <CardTitle>Playground</CardTitle>
            <CardDescription>
              Live prop controls for Button, Input, Tag, and Card with responsive preview widths and Vue snippet output.
            </CardDescription>
          </div>
          <button
            type="button"
            class="rounded-xl border border-border/70 px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
            @click="resetState"
          >
            Reset playground
          </button>
        </div>
      </CardHeader>
      <CardContent class="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(280px,0.8fr)]">
        <PlaygroundPreviewCanvas :state="state" />
        <div class="grid gap-6">
          <PlaygroundInspector v-model="state" />
          <PlaygroundCodePanel :output="output" />
        </div>
      </CardContent>
    </Card>
  </section>
</template>
