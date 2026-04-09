<script setup lang="ts">
import { DESIGN_LAB_SECTIONS } from "@/lib/design-lab/sections"
</script>

<template>
  <nav
    class="border-r border-border/70 bg-card/70 px-3 py-6"
    aria-label="Design Lab sections"
  >
    <div class="sticky top-0 flex flex-col gap-2">
      <a
        v-for="section in DESIGN_LAB_SECTIONS"
        :key="section.id"
        :href="`#${section.id}`"
        class="rounded-xl border border-transparent px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:border-border/70 hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
      >
        {{ section.label }}
      </a>
    </div>
  </nav>
</template>
