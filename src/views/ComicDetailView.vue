<script setup lang="ts">
import { Skeleton } from "@/components/ui/skeleton"
import { computed, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import ComicDetailPanel from "@/components/jav-library/comics/ComicDetailPanel.vue"
import ComicPagePreviewGrid from "@/components/jav-library/comics/ComicPagePreviewGrid.vue"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import { buildComicReaderRouteFromSource } from "@/lib/navigation-intent"
import { useComicLibraryService } from "@/services/comic-library-service"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const comicService = useComicLibraryService()

const comicId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)
const detailComic = shallowRef<ComicBook | undefined>()
const detailLoading = ref(false)
const patchBusy = ref(false)
const errorText = ref("")

watch(
  () => comicId.value,
  async (id, _previous, onCleanup) => {
    let stale = false
    onCleanup(() => { stale = true })
    detailComic.value = undefined
    errorText.value = ""
    if (!id) return
    detailLoading.value = true
    try {
      const loaded = await comicService.loadComicDetail(id)
      if (stale) return
      detailComic.value = loaded
      if (!detailComic.value) {
        errorText.value = t("comics.detailNotFound")
      }
    } catch (error) {
      if (stale) return
      errorText.value = error instanceof Error ? error.message : t("comics.detailLoadError")
    } finally {
      if (!stale) detailLoading.value = false
    }
  },
  { immediate: true },
)

async function patchComic(patch: ComicPatch, done?: (err?: unknown) => void) {
  const id = detailComic.value?.id
  if (!id) return
  patchBusy.value = true
  errorText.value = ""
  try {
    const updated = await comicService.patchComic(id, patch)
    if (updated) {
      detailComic.value = updated
    }
    done?.()
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("comics.detailSaveError")
    done?.(error)
  } finally {
    patchBusy.value = false
  }
}

async function deleteComic(comicId: string) {
  const id = comicId.trim() || detailComic.value?.id
  if (!id) return
  patchBusy.value = true
  errorText.value = ""
  try {
    await comicService.deleteComic(id)
    await router.replace({ name: "comics" })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("comics.deleteComicError")
  } finally {
    patchBusy.value = false
  }
}

async function revealSource(comicId: string) {
  const id = comicId.trim() || detailComic.value?.id
  if (!id) return
  errorText.value = ""
  try {
    await comicService.revealComicSource(id)
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("comics.revealComicError")
  }
}

function openReader(pageIndex: number) {
  const id = detailComic.value?.id
  if (!id) return
  void router.push(buildComicReaderRouteFromSource(id, pageIndex, route.fullPath))
}

function browseByTag(payload: { tag: string }) {
  const q = payload.tag.trim()
  if (!q) return
  void router.push({
    name: "comics",
    query: { q },
  })
}
</script>

<template>
  <div class="h-full min-w-0 w-full overflow-y-auto px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]">
    <h1 class="sr-only">{{ detailComic?.title || t('nav.comics') }}</h1>
    <div v-if="detailLoading" role="status" :aria-label="t('comics.detailLoading')" class="grid gap-6 rounded-3xl border border-border/70 p-6 sm:grid-cols-[14rem_1fr]">
      <Skeleton class="aspect-[2/3] rounded-2xl" /><div class="flex flex-col gap-4"><Skeleton class="h-10 w-3/4" /><Skeleton class="h-40 w-full" /><Skeleton class="h-11 w-36 rounded-full" /></div>
    </div>

    <template v-else-if="detailComic">
      <div class="flex min-w-0 flex-col gap-6">
        <p
          v-if="errorText"
          role="alert"
          class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        >
          {{ errorText }}
        </p>

        <ComicDetailPanel
          :comic="detailComic"
          :busy="patchBusy"
          @patch="patchComic"
          @start-reading="openReader"
          @delete-comic="deleteComic"
          @reveal-source="revealSource"
          @browse-by-tag="browseByTag"
        />

        <ComicPagePreviewGrid
          :comic="detailComic"
          @open-reader="openReader"
        />
      </div>
    </template>

    <div
      v-else
      class="rounded-xl border border-dashed border-border/70 bg-muted/20 p-8 text-center text-sm text-muted-foreground"
    >
      {{ errorText || t("comics.detailNotFound") }}
    </div>
  </div>
</template>
