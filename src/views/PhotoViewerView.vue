<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useRoute } from "vue-router"
import { useI18n } from "vue-i18n"
import PhotoViewer from "@/components/jav-library/photos/PhotoViewer.vue"
import type { PhotoBook } from "@/domain/photo/types"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const route = useRoute()
const { t } = useI18n()
const photoService = usePhotoLibraryService()

const photoId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)
const pageIndex = computed(() =>
  typeof route.params.pageIndex === "string" ? Number(route.params.pageIndex) : 0,
)

const photo = shallowRef<PhotoBook | undefined>()
const loading = ref(false)
const loadError = ref("")

const initialPageIndex = computed(() => {
  const raw = pageIndex.value
  return Number.isFinite(raw) ? Math.max(0, Math.floor(raw)) : 0
})

watch(
  () => photoId.value,
  async (id) => {
    photo.value = undefined
    loadError.value = ""
    if (!id) return
    loading.value = true
    try {
      await photoService.refreshSettings()
      photo.value = await photoService.loadPhotoDetail(id)
      if (!photo.value) {
        loadError.value = t("photos.detailNotFound")
      }
    } catch (error) {
      loadError.value = error instanceof Error ? error.message : t("photos.detailLoadError")
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
      {{ t("photos.detailLoading") }}
    </div>
    <PhotoViewer
      v-else-if="photo"
      :photo="photo"
      :viewer-defaults="photoService.photoViewer.value"
      :initial-page-index="initialPageIndex"
    />
    <div
      v-else
      class="flex h-full items-center justify-center p-6 text-sm text-muted-foreground"
    >
      {{ loadError || t("photos.detailNotFound") }}
    </div>
  </div>
</template>
