<script setup lang="ts">
import { Skeleton } from "@/components/ui/skeleton"
import { computed, ref, shallowRef, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import PhotoDetailPanel from "@/components/jav-library/photos/PhotoDetailPanel.vue"
import PhotoPagePreviewGrid from "@/components/jav-library/photos/PhotoPagePreviewGrid.vue"
import BookCommentSection from "@/components/jav-library/books/BookCommentSection.vue"
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
const patchBusy = ref(false)
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

/** 详情标签走精确 tag=，避免把标签字面量当成墙面子串搜索。 */
function browseByTag(payload: { tag: string }) {
  const tag = payload.tag.trim()
  if (!tag) return
  void router.push({
    name: "photos",
    query: { tag },
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

/** 把详情评分卡的选择写入当前写真。 */
async function updateRating(value: number | null) {
  const previous = detailPhoto.value
  if (!previous) return
  patchBusy.value = true
  errorText.value = ""
  try {
    const updated = await photoService.patchPhoto(previous.id, { rating: value })
    if (detailPhoto.value?.id === updated.id) {
      detailPhoto.value = updated
    }
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("photos.detailSaveError")
  } finally {
    patchBusy.value = false
  }
}

/** 把媒体信息弹窗的展示标题写入当前写真。 */
async function savePhotoTitle(title: string, done: (err?: unknown) => void) {
  const previous = detailPhoto.value
  if (!previous) {
    done(new Error(t("photos.detailNotFound")))
    return
  }
  patchBusy.value = true
  errorText.value = ""
  try {
    const updated = await photoService.patchPhoto(previous.id, { title })
    if (detailPhoto.value?.id === updated.id) {
      detailPhoto.value = updated
    }
    done()
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("photos.detailSaveError")
    done(error)
  } finally {
    patchBusy.value = false
  }
}

/** 翻译确认后重新读取当前写真，避免草稿覆盖后续写入。 */
async function reloadPhoto() {
  const id = detailPhoto.value?.id
  if (!id) return
  try {
    const loaded = await photoService.loadPhotoDetail(id)
    if (loaded && detailPhoto.value?.id === loaded.id) {
      detailPhoto.value = loaded
    }
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t("photos.detailLoadError")
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
          :busy="patchBusy"
          @add-tag="addTag"
          @start-browsing="openViewer"
          @browse-by-tag="browseByTag"
          @update-rating="updateRating"
          @save-title="savePhotoTitle"
          @reload="reloadPhoto"
        />

        <PhotoPagePreviewGrid
          :photo="detailPhoto"
          @open-viewer="openViewer"
        />

        <BookCommentSection
          kind="photos"
          :entity-id="detailPhoto.id"
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
