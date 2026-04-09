<script setup lang="ts">
import { computed } from "vue"
import { Moon, RotateCcw, Sun } from "lucide-vue-next"
import { useTheme } from "@/composables/use-theme"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const { resolvedMode, setThemePreference } = useTheme()

const themeToggleLabel = computed(() =>
  resolvedMode.value === "dark" ? "Switch to light" : "Switch to dark",
)

function toggleTheme() {
  setThemePreference(resolvedMode.value === "dark" ? "light" : "dark")
}
</script>

<template>
  <header
    class="border-b border-border/70 bg-background/92 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/70 lg:px-8"
  >
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="min-w-0 space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="secondary" class="rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em]">
            Dev Only
          </Badge>
          <Badge variant="outline" class="rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em]">
            /design-lab
          </Badge>
        </div>
        <div class="space-y-1">
          <h1 class="text-2xl font-semibold tracking-tight text-foreground lg:text-3xl">
            Curated Design Lab
          </h1>
          <p class="max-w-3xl text-sm leading-6 text-muted-foreground">
            Internal playground for tokens, component states, and interaction prototyping.
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          class="h-10 rounded-xl px-4"
          :aria-label="themeToggleLabel"
          @click="toggleTheme"
        >
          <Sun v-if="resolvedMode === 'dark'" data-icon="inline-start" />
          <Moon v-else data-icon="inline-start" />
          {{ themeToggleLabel }}
        </Button>
        <Button variant="secondary" class="h-10 rounded-xl px-4" disabled>
          Desktop
        </Button>
        <Button variant="ghost" class="h-10 rounded-xl px-4" disabled>
          <RotateCcw data-icon="inline-start" />
          Reset
        </Button>
      </div>
    </div>
  </header>
</template>
