<script setup lang="ts">
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { getContrastRatio } from "@/lib/design-lab/contrast"

const contrastExamples = [
  {
    name: "Primary",
    foreground: "#1D0910",
    background: "#FE628E",
  },
  {
    name: "Danger",
    foreground: "#25070F",
    background: "#E14B6D",
  },
  {
    name: "Info",
    foreground: "#0B1332",
    background: "#5B6FD4",
  },
].map((example) => ({
  ...example,
  ratio: getContrastRatio(example.foreground, example.background),
}))
</script>

<template>
  <section id="accessibility" class="scroll-mt-24">
    <Card class="rounded-2xl border border-border bg-card shadow-sm">
      <CardHeader>
        <CardTitle>Accessibility</CardTitle>
        <CardDescription>
          Focus visibility, keyboard affordances, and contrast checks should be visible next to the same tokens used in production UI.
        </CardDescription>
      </CardHeader>
      <CardContent class="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <div class="grid gap-4">
          <article class="rounded-2xl border border-border/70 p-4">
            <h3 class="text-sm font-semibold text-foreground">Focus-visible Examples</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Keyboard users should see an unambiguous ring on actions and text inputs.
            </p>
            <div class="mt-4 flex flex-wrap gap-3">
              <Button class="ring-ring/50 ring-[3px]">Primary focus</Button>
              <Button variant="outline" class="ring-ring/50 ring-[3px]">Outline focus</Button>
            </div>
            <div class="mt-3 max-w-sm">
              <Input default-value="Keyboard focus field" class="border-ring ring-ring/50 ring-[3px]" />
            </div>
          </article>

          <article class="rounded-2xl border border-border/70 p-4">
            <h3 class="text-sm font-semibold text-foreground">Keyboard Navigation Notes</h3>
            <ul class="mt-3 space-y-2 text-sm leading-6 text-muted-foreground">
              <li>Use semantic buttons and inputs instead of clickable `div` wrappers.</li>
              <li>Preserve visible focus styling on every interactive component state.</li>
              <li>Do not rely on opacity alone to communicate disabled or unavailable actions.</li>
            </ul>
          </article>
        </div>

        <article class="rounded-2xl border border-border/70 p-4">
          <h3 class="text-sm font-semibold text-foreground">Contrast Summary</h3>
          <div class="mt-4 grid gap-3">
            <div
              v-for="example in contrastExamples"
              :key="example.name"
              class="rounded-2xl border border-border/70 p-3"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium text-foreground">{{ example.name }}</span>
                <span class="font-mono text-xs text-foreground">{{ example.ratio }}:1</span>
              </div>
              <div
                class="mt-3 flex h-16 items-center justify-center rounded-xl text-sm font-medium"
                :style="{ backgroundColor: example.background, color: example.foreground }"
              >
                Accessible sample text
              </div>
              <p class="mt-3 text-xs leading-5 text-muted-foreground">
                {{ example.ratio >= 4.5 ? "Passes AA for normal text." : "Below AA, adjust before shipping." }}
              </p>
            </div>
          </div>
        </article>
      </CardContent>
    </Card>
  </section>
</template>
