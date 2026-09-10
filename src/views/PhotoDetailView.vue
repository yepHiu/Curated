<script setup lang="ts">
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
  async (id) => {
    detailPhoto.value = undefined
    errorText.value = ""
    if (!id) return
    detailLoading.value = true
    try {
      await photoService.refreshSettings()
      detailPhoto.value = await photoService.loadPhotoDetail(id)
      if (!detailPhoto.value) {
        errorText.value = t("photos.detailNotFound")
      }
    } catch (error) {
      errorText.value = error instanceof Error ? error.message : t("photos.detailLoadError")
    } finally {
      detailLoading.value = false
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
</script>

<template>
  <div class="h-full min-w-0 w-full overflow-y-auto px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]">
    <div
      v-if="detailLoading"
      class="rounded-xl border border-border/70 bg-card/80 p-6 text-sm text-muted-foreground"
    >
      {{ t("photos.detailLoading") }}
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
          :photo="detailPhoto"
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
