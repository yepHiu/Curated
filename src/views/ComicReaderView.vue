<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"
import ComicReader from "@/components/jav-library/comics/ComicReader.vue"
import type { ComicBook } from "@/domain/comic/types"
import { clampComicPageIndex } from "@/lib/comic-reader-controls"
import { useComicLibraryService } from "@/services/comic-library-service"

const { t } = useI18n()
const route = useRoute()
const comicService = useComicLibraryService()

const comicId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)
const initialPageIndex = computed(() => {
  const raw = typeof route.params.pageIndex === "string" ? Number(route.params.pageIndex) : 0
  return Number.isFinite(raw) ? Math.max(0, Math.floor(raw)) : 0
})
const comic = shallowRef<ComicBook | undefined>()
const loading = ref(false)
const loadError = ref("")

watch(
  () => comicId.value,
  async (id) => {
    comic.value = undefined
    loadError.value = ""
    if (!id) return
    loading.value = true
    try {
      comic.value = await comicService.loadComicDetail(id)
      if (!comic.value) {
        loadError.value = t("comics.detailNotFound")
      }
    } catch (error) {
      loadError.value = error instanceof Error ? error.message : t("comics.detailLoadError")
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="h-full min-h-0 min-w-0">
    <div
      v-if="loading"
      class="flex h-full items-center justify-center text-sm text-muted-foreground"
    >
      {{ t("comics.detailLoading") }}
    </div>
    <ComicReader
      v-else-if="comic"
      :comic="comic"
      :reader-defaults="comicService.comicReader.value"
      :initial-page-index="clampComicPageIndex(initialPageIndex, comic.pageCount)"
      :load-preferences="comicService.getComicPreferences"
      :save-preferences="comicService.saveComicPreferences"
      :save-progress="comicService.saveComicProgress"
    />
    <div
      v-else
      class="flex h-full items-center justify-center p-6 text-sm text-muted-foreground"
    >
      {{ loadError || t("comics.detailNotFound") }}
    </div>
  </div>
</template>
