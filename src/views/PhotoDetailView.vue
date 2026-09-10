<script setup lang="ts">
import { Skeleton } from "@/components/ui/skeleton"
import { computed, ref, shallowRef, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import PhotoDetailPanel from "@/components/jav-library/photos/PhotoDetailPanel.vue"
import PhotoPagePreviewGrid from "@/components/jav-library/photos/PhotoPagePreviewGrid.vue"
import type { PhotoBook } from "@/domain/photo/types"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const photoService = usePhotoLibraryService()

const photoId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const detailPhoto = shallowRef<PhotoBook | undefined>()
const detailLoading = ref(false)
const errorText = ref("")

watch(
  () => photoId.value,
  async (id, _previous, onCleanup) => {
    let stale = false
    onCleanup(() => { stale = true })
    detailPhoto.value = undefined
    errorText.value = ""
    if (!id) return
    detailLoading.value = true
    try {
      await photoService.refreshSettings()
      const loaded = await photoService.loadPhotoDetail(id)
      if (stale) return
      detailPhoto.value = loaded
      if (!detailPhoto.value) {
        errorText.value = t("photos.detailNotFound")
      }
    } catch (error) {
      if (stale) return
      errorText.value = error instanceof Error ? error.message : t("photos.detailLoadError")
    } finally {
      if (!stale) detailLoading.value = false
    }
  },
  { immediate: true },
)

function openViewer(pageIndex: number) {
  const id = detailPhoto.value?.id
  if (!id) return
  void router.push({
    name: "photo-viewer",
    params: { id, pageIndex: String(Math.max(0, Math.floor(pageIndex))) },
    query: { returnTo: route.fullPath },
  })
}

function browseByTag(payload: { tag: string }) {
  const q = payload.tag.trim()
  if (!q) return
  void router.push({
    name: "photos",
    query: { q },
  })
}

async function addTag(tag: string, done: (error?: unknown) => void) {
  const previous = detailPhoto.value
  if (!previous) { done(new Error(t("photos.detailNotFound"))); return }
  try {
    const updated = await photoService.replacePhotoTags(previous.id, [...new Set([...previous.tags, tag])])
    if (detailPhoto.value === previous) detailPhoto.value = updated
    done()
  } catch (error) {
    done(error)
  }
}
</script>

<template>
  <div class="h-full min-w-0 w-full overflow-y-auto pr-2">
    <h1 class="sr-only">{{ detailPhoto?.title || t('nav.photos') }}</h1>
    <div v-if="detailLoading" role="status" :aria-label="t('photos.detailLoading')" class="grid gap-6 rounded-3xl border border-border/70 p-6 sm:grid-cols-[14rem_1fr]">
      <Skeleton class="aspect-[2/3] rounded-2xl" /><div class="flex flex-col gap-4"><Skeleton class="h-10 w-3/4" /><Skeleton class="h-40 w-full" /><Skeleton class="h-11 w-36 rounded-full" /></div>
    </div>

    <template v-else-if="detailPhoto">
      <div class="flex min-w-0 flex-col gap-6">
        <p
          v-if="errorText"
          role="alert"
          class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        >
          {{ errorText }}
        </p>

        <PhotoDetailPanel
          :key="detailPhoto.id"
          :photo="detailPhoto"
          @add-tag="addTag"
          @start-browsing="openViewer"
          @browse-by-tag="browseByTag"
        />

        <PhotoPagePreviewGrid
          :photo="detailPhoto"
          @open-viewer="openViewer"
        />
      </div>
    </template>

    <div
      v-else
      class="rounded-xl border border-dashed border-border/70 bg-muted/20 p-8 text-center text-sm text-muted-foreground"
    >
      {{ errorText || t("photos.detailNotFound") }}
    </div>
  </div>
</template>
