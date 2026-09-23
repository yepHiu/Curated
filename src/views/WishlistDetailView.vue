<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { useLibraryService } from "@/services/library-service"
import type { Movie } from "@/domain/movie/types"
import type { WishlistItem } from "@/domain/wishlist/types"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import WishlistPlaybackCard from "@/components/jav-library/wishlist/WishlistPlaybackCard.vue"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const service = useLibraryService().wishlist
const item = ref<WishlistItem>()
const loading = ref(true)
const error = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
let alive = true
let generation = 0

// Adapt wishlist metadata for the existing detail component without invoking movie APIs.
const movie = computed<Movie | undefined>(() => {
  const value = item.value
  if (!value) return undefined
  const metadata = value.metadata
  const cover = value.assets.find((asset) => asset.role === "cover") ?? value.assets[0]
  return {
    id: value.id,
    code: value.code,
    title: metadata.title || value.code,
    summary: metadata.summary,
    actors: metadata.actors,
    tags: metadata.tags,
    studio: metadata.studio,
    releaseDate: metadata.releaseDate,
    year: Number(metadata.releaseDate.slice(0, 4)) || 0,
    runtimeMinutes: metadata.runtimeMinutes,
    metadataProvider: metadata.provider,
    coverUrl: cover ? service.assetUrl(cover.url) : "",
    thumbUrl: cover ? service.assetUrl(cover.thumbnailUrl) : "",
    previewImages: value.assets.filter((asset) => asset.role === "preview_image").map((asset) => service.assetUrl(asset.url)),
    userTags: [],
    rating: 0,
    isFavorite: false,
    addedAt: value.createdAt,
    location: "",
    resolution: "",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
  }
})

async function load(initial = false) {
  const current = ++generation
  if (initial) loading.value = true
  try {
    const result = await service.get(String(route.params.id))
    if (!alive || current !== generation) return
    item.value = result
    error.value = false
  } catch {
    if (alive && current === generation) error.value = true
  } finally {
    if (alive && current === generation) loading.value = false
  }
}

async function poll() {
  if (!document.hidden && item.value && ["queued", "running"].includes(item.value.enrichmentState)) await load()
  if (alive) timer = setTimeout(poll, 6000)
}

function browseByTag({ tag }: { tag: string }) {
  void router.push({ name: "library", query: { tag } })
}
function browseByActor({ actor }: { actor: string }) {
  void router.push({ name: "actor-detail", params: { actorName: actor } })
}
function browseByStudio({ studio }: { studio: string }) {
  void router.push({ name: "library", query: { studio } })
}

watch(() => route.params.id, () => {
  item.value = undefined
  error.value = false
  void load(true)
}, { immediate: true })
timer = setTimeout(poll, 6000)
onBeforeUnmount(() => {
  alive = false
  generation++
  clearTimeout(timer)
})
</script>

<template>
  <div class="h-full min-w-0 w-full overflow-y-auto pr-2">
    <div v-if="loading" class="rounded-3xl border border-border/70 bg-card/80 p-8 text-sm text-muted-foreground">
      {{ t('detail.loadingDetail') }}
    </div>
    <DetailPage
      v-else-if="movie"
      :movie="movie"
      :source-url="item?.sourceUrl"
      :related-movies="[]"
      read-only
      @browse-by-tag="browseByTag"
      @browse-by-actor="browseByActor"
      @browse-by-studio="browseByStudio"
    >
      <template #after-detail-panel>
        <WishlistPlaybackCard :key="`${item!.id}:${item!.code}`" :item-id="item!.id" :code="item!.code" />
      </template>
    </DetailPage>
    <NotFoundState
      v-else
      :title="t('detail.notFoundTitle')"
      :description="t(error ? 'detail.loadError' : 'detail.notFoundDescFallback')"
    />
  </div>
</template>
