<script setup lang="ts">
import type { Component } from "vue"
import {
  Clock3,
  Heart,
  LibraryBig,
  Settings2,
  Sparkles,
  Tags,
} from "lucide-vue-next"
import { RouterLink, useRoute } from "vue-router"
import type { AppPage, LibraryMode } from "@/domain/library/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { buildBrowseRouteTarget } from "@/lib/library-query"

interface NavigationItem {
  label: string
  page: AppPage
  icon: Component
  hint?: string
}

const route = useRoute()

const browseItems: NavigationItem[] = [
  { label: "All Movies", page: "library", icon: LibraryBig, hint: "2.1k" },
  { label: "Favorites", page: "favorites", icon: Heart, hint: "246" },
  { label: "Recently Added", page: "recent", icon: Clock3, hint: "48" },
  { label: "Tags", page: "tags", icon: Tags, hint: "132" },
]

const isActive = (page: AppPage) => route.name === page

const getNavigationTarget = (page: AppPage) =>
  page === "settings"
    ? { name: page }
    : buildBrowseRouteTarget(page as LibraryMode, route.query)
</script>

<template>
  <aside
    class="flex h-full w-full min-w-0 flex-col rounded-3xl border border-border/70 bg-sidebar/95 p-4 text-sidebar-foreground shadow-2xl shadow-black/20 backdrop-blur"
  >
    <div class="flex items-center justify-between gap-3 px-2 py-3">
      <div class="flex min-w-0 flex-col gap-1">
        <span class="text-xs font-medium uppercase tracking-[0.28em] text-muted-foreground">
          Media Shelf
        </span>
        <h1 class="truncate text-xl font-semibold">JAV Library</h1>
      </div>
      <Badge class="rounded-full bg-primary/15 text-primary hover:bg-primary/15">
        <Sparkles class="text-primary" />
        Curated
      </Badge>
    </div>

    <Separator class="my-3 bg-sidebar-border/80" />

    <ScrollArea class="min-h-0 flex-1">
      <div class="flex flex-col gap-6 pr-3">
        <section class="flex flex-col gap-2">
          <span class="px-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">
            Browse
          </span>
          <Button
            v-for="item in browseItems"
            :key="item.page"
            as-child
            :variant="isActive(item.page) ? 'secondary' : 'ghost'"
            class="w-full justify-between rounded-2xl px-3"
          >
            <RouterLink :to="getNavigationTarget(item.page)">
              <span class="flex min-w-0 items-center gap-2 truncate">
                <component :is="item.icon" data-icon="inline-start" />
                <span class="truncate">{{ item.label }}</span>
              </span>
              <Badge
                v-if="item.hint"
                variant="secondary"
                class="rounded-full border border-border/60 bg-background/60"
              >
                {{ item.hint }}
              </Badge>
            </RouterLink>
          </Button>
        </section>

      </div>
    </ScrollArea>

    <Separator class="my-3 bg-sidebar-border/80" />

    <Button
      as-child
      :variant="isActive('settings') ? 'secondary' : 'ghost'"
      class="w-full justify-start rounded-2xl px-3"
    >
      <RouterLink :to="getNavigationTarget('settings')">
        <Settings2 data-icon="inline-start" />
        Settings
      </RouterLink>
    </Button>
  </aside>
</template>
