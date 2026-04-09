<script setup lang="ts">
import { Check, Loader2 } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"

const buttonStates = [
  { label: "Default", class: "" },
  { label: "Hover", class: "bg-primary/90" },
  { label: "Active", class: "bg-primary/95 scale-[0.99]" },
  { label: "Focus", class: "ring-ring/50 ring-[3px]" },
] as const
</script>

<template>
  <section id="components" class="scroll-mt-24">
    <Card class="rounded-2xl border border-border bg-card shadow-sm">
      <CardHeader>
        <CardTitle>Components</CardTitle>
        <CardDescription>
          Canonical component states should be visible here before interactive prop tuning changes their shape.
        </CardDescription>
      </CardHeader>
      <CardContent class="grid gap-6 xl:grid-cols-2">
        <article class="rounded-2xl border border-border/70 p-4">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Button</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Primary actions should read clearly in every state, especially focus-visible and loading.
            </p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div
              v-for="state in buttonStates"
              :key="state.label"
              class="rounded-xl border border-border/60 bg-surface p-3"
            >
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
                {{ state.label }}
              </p>
              <Button :class="state.class">Save changes</Button>
            </div>
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
                Loading
              </p>
              <Button disabled>
                <Loader2 data-icon="inline-start" class="animate-spin" />
                Saving
              </Button>
            </div>
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
                Disabled
              </p>
              <Button disabled>Disabled action</Button>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border border-border/70 p-4">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Input</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Form fields should stay readable across default, focus, invalid, and disabled states.
            </p>
          </div>
          <div class="grid gap-3">
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Default</p>
              <Input default-value="Prototype title" />
            </div>
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Focus</p>
              <Input default-value="Focused field" class="border-ring ring-ring/50 ring-[3px]" />
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-xl border border-border/60 bg-surface p-3">
                <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Invalid</p>
                <Input default-value="Missing metadata" aria-invalid="true" />
              </div>
              <div class="rounded-xl border border-border/60 bg-surface p-3">
                <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Disabled</p>
                <Input default-value="Read only source" disabled />
              </div>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border border-border/70 p-4">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Tag / Badge</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Tags should stay compact, legible, and semantically distinct without looking like tiny buttons.
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <Badge>Featured</Badge>
            <Badge variant="secondary">Queued</Badge>
            <Badge variant="outline">Preview</Badge>
            <Badge variant="destructive">Risk</Badge>
            <Badge class="ring-ring/50 ring-[3px]">
              <Check />
              Focus
            </Badge>
          </div>
        </article>

        <article class="rounded-2xl border border-border/70 p-4">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Card</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Cards should establish surface hierarchy before the playground starts changing padding, radius, and shadow.
            </p>
          </div>
          <div class="grid gap-3">
            <Card class="rounded-2xl border border-border/70 bg-surface shadow-sm">
              <CardHeader class="gap-2">
                <CardTitle class="text-base">Default card</CardTitle>
                <CardDescription>Primary information container for settings and previews.</CardDescription>
              </CardHeader>
              <CardContent class="text-sm leading-6 text-muted-foreground">
                Uses the standard surface, border, and shadow stack.
              </CardContent>
            </Card>
            <Card class="rounded-2xl border border-border/70 bg-surface shadow-sm ring-ring/50 ring-[3px]">
              <CardHeader class="gap-2">
                <CardTitle class="text-base">Focus-visible card</CardTitle>
                <CardDescription>Used for keyboard-targetable surfaces or section highlights.</CardDescription>
              </CardHeader>
            </Card>
          </div>
        </article>

        <article class="rounded-2xl border border-border/70 p-4">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Checkbox And Switch</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Switch uses the shared primitive. Checkbox is shown as a native prototype until a shared checkbox primitive is added.
            </p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Checkbox</p>
              <label class="flex items-center gap-3 text-sm text-foreground">
                <input type="checkbox" checked class="size-4 accent-[var(--primary)]" />
                Include preview assets
              </label>
            </div>
            <div class="rounded-xl border border-border/60 bg-surface p-3">
              <p class="mb-3 text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Switch</p>
              <label class="flex items-center justify-between gap-3 text-sm text-foreground">
                <span>Use dark preview</span>
                <Switch :default-value="true" />
              </label>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border border-border/70 p-4 xl:col-span-2">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-foreground">Skeleton</h3>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">
              Loading placeholders should preserve layout rhythm instead of inventing their own geometry.
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-[240px_minmax(0,1fr)]">
            <Skeleton class="h-48 rounded-2xl" />
            <div class="grid gap-3">
              <Skeleton class="h-5 w-40" />
              <Skeleton class="h-4 w-full" />
              <Skeleton class="h-4 w-5/6" />
              <Skeleton class="h-4 w-2/3" />
            </div>
          </div>
        </article>
      </CardContent>
    </Card>
  </section>
</template>
