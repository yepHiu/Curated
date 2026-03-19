<script setup lang="ts">
import { computed } from "vue"
import { LayoutDashboard, Search } from "lucide-vue-next"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import AppSidebar from "@/components/jav-library/AppSidebar.vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { getMovieById } from "@/lib/jav-library"

const route = useRoute()
const router = useRouter()

const currentMovieId = computed(() => {
  const routeMovieId = typeof route.params.id === "string" ? route.params.id : undefined
  const selectedMovieId = typeof route.query.selected === "string" ? route.query.selected : undefined

  return getMovieById(routeMovieId ?? selectedMovieId).id
})

const isLibraryRoute = computed(() =>
  route.name === "library" ||
  route.name === "favorites" ||
  route.name === "recent" ||
  route.name === "tags",
)

const isPlayerRoute = computed(() => route.name === "player")

const showHeaderBack = computed(() => !isLibraryRoute.value)

const headerBackTarget = computed(() =>
  isPlayerRoute.value
    ? {
        name: "detail",
        params: { id: currentMovieId.value },
      }
    : { name: "library" },
)

const headerBackLabel = computed(() =>
  isPlayerRoute.value ? "Back to details" : "Back to library",
)

const headerSearchValue = computed({
  get: () => (typeof route.query.q === "string" ? route.query.q : ""),
  set: (value: string) => {
    if (!isLibraryRoute.value) {
      return
    }

    void router.replace({
      name: route.name ?? "library",
      query: {
        ...route.query,
        q: value || undefined,
      },
    })
  },
})
</script>

<template>
  <div class="h-screen overflow-hidden bg-background text-foreground">
    <div class="h-full w-full px-3 py-3 lg:px-4 lg:py-4">
      <div class="grid h-full min-h-0 gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
        <AppSidebar :current-movie-id="currentMovieId" />

        <section
          class="flex min-h-0 flex-col overflow-hidden rounded-[2rem] border border-border/70 bg-background/85 shadow-2xl shadow-black/10 backdrop-blur"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/70 px-4 py-4 lg:px-5">
            <div class="flex flex-wrap items-center gap-2">
              <Button v-if="showHeaderBack" as-child variant="secondary" class="rounded-2xl">
                <RouterLink :to="headerBackTarget">
                  <LayoutDashboard data-icon="inline-start" />
                  {{ headerBackLabel }}
                </RouterLink>
              </Button>
            </div>

            <div class="flex flex-1 justify-center px-0 lg:px-3">
              <div
                v-if="isLibraryRoute"
                class="relative w-full max-w-lg"
              >
                <Search class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="headerSearchValue"
                  class="h-10 rounded-2xl border-border/70 bg-background/70 pl-10 transition-[border-color,background-color,box-shadow] hover:border-primary/60 hover:bg-background/85 hover:ring-1 hover:ring-primary/30"
                  placeholder="Search by code, title, actor, or tag"
                />
              </div>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden p-4 lg:p-5 xl:p-6">
            <RouterView />
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
