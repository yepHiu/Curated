<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { HttpClientError } from "@/api/http-client"
import type { PatchMovieBody, TaskDTO } from "@/api/types"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import {
  getBrowseSourceMode,
  getDetailBrowseTargetMode,
  mergeLibraryQuery,
} from "@/lib/library-query"
import { buildDetailRouteFromBrowse, buildPlayerRouteFromBrowseIntent } from "@/lib/navigation-intent"
import { buildUserTagSuggestionPool } from "@/lib/user-tag-suggestions"
import { loadMovieDetail } from "@/services/adapters/web/web-library-service"
import { useLibraryService } from "@/services/library-service"
import { bumpMovieImageVersion } from "@/lib/image-version"
import type { Movie } from "@/domain/movie/types"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()
const scanTaskTracker = useScanTaskTracker()

function isTerminalTaskStatus(status: TaskDTO["status"]): boolean {
  return (
    status === "completed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "partial_failed"
  )
}

function formatClientError(err: unknown, fallback: string) {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || err.message || fallback
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const detailMovie = shallowRef<Movie | undefined>()
const detailLoading = ref(false)
const detailLoadError = ref("")
const patchError = ref("")
const deleteBusy = ref(false)
const deleteError = ref("")
const restoreBusy = ref(false)
const restoreError = ref("")
const permanentDeleteBusy = ref(false)
const permanentDeleteError = ref("")
const metadataRefreshBusy = ref(false)
const metadataRefreshError = ref("")

watch(
  () => movieId.value,
  async (id) => {
    detailMovie.value = undefined
    detailLoadError.value = ""
    patchError.value = ""
    if (!id) {
      return
    }
    if (USE_WEB_API) {
      detailLoading.value = true
      try {
        const loaded = await loadMovieDetail(id)
        if (loaded) {
          detailMovie.value = loaded
        } else {
          detailMovie.value = libraryService.getMovieById(id)
          if (!detailMovie.value) {
            detailLoadError.value = t("detail.loadError")
          }
        }
      } finally {
        detailLoading.value = false
      }
    } else {
      detailMovie.value = libraryService.getMovieById(id)
    }
  },
  { immediate: true },
)

watch(scanTaskTracker.activeTask, async (task) => {
  const id = movieId.value
  if (!USE_WEB_API || !id || !task) return
  if (!isTerminalTaskStatus(task.status)) return
  if (task.type !== "scrape.movie") return
  const mid =
    task.metadata && typeof task.metadata.movieId === "string" ? task.metadata.movieId : undefined
  if (mid !== id) return

  if (task.status === "completed") {
    metadataRefreshError.value = ""
    // 递增图片版本号，强制刷新海报/缩略图缓存
    bumpMovieImageVersion(id)
    const loaded = await loadMovieDetail(id)
    if (loaded) {
      detailMovie.value = loaded
    }
  } else if (task.status === "failed" || task.status === "partial_failed") {
    metadataRefreshError.value = task.errorMessage?.trim() || t("detail.scrapeFailed")
  }
})

const relatedMovies = computed(() =>
  detailMovie.value ? libraryService.getRelatedMovies(detailMovie.value.id) : [],
)

/** 全库 userTags ∪ 当前片元数据 tags，供「我的标签」输入联想 */
const userTagSuggestionPool = computed(() =>
  buildUserTagSuggestionPool(libraryService.movies.value, detailMovie.value?.tags ?? []),
)

const selectMovie = async (nextMovieId: string) => {
  await router.replace(
    buildDetailRouteFromBrowse(nextMovieId, route.query, getBrowseSourceMode(route.query)),
  )
}

const openDetails = async (nextMovieId: string) => {
  await router.push(
    buildDetailRouteFromBrowse(nextMovieId, route.query, getBrowseSourceMode(route.query)),
  )
}

const openPlayer = async (nextMovieId: string) => {
  await router.push(
    buildPlayerRouteFromBrowseIntent(
      nextMovieId,
      route.query,
      getBrowseSourceMode(route.query),
      "detail",
    ),
  )
}

const toggleFavorite = async (payload: { movieId: string; nextValue: boolean }) => {
  patchError.value = ""
  try {
    const updated = await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
    if (updated && detailMovie.value?.id === payload.movieId) {
      detailMovie.value = updated
    }
  } catch (err) {
    patchError.value = formatClientError(err, t("detail.errFavorite"))
    console.error("[DetailView] toggle favorite failed", err)
  }
}

const updateUserRating = async (payload: { movieId: string; value: number | null }) => {
  patchError.value = ""
  try {
    const updated = await libraryService.patchMovie(payload.movieId, {
      rating: payload.value,
    })
    if (updated && detailMovie.value?.id === payload.movieId) {
      detailMovie.value = updated
    }
  } catch (err) {
    patchError.value = formatClientError(err, t("detail.errRating"))
    console.error("[DetailView] update user rating failed", err)
  }
}

const updateUserTags = async (payload: { movieId: string; tags: string[] }) => {
  patchError.value = ""
  try {
    const updated = await libraryService.patchMovie(payload.movieId, {
      userTags: payload.tags,
    })
    if (updated && detailMovie.value?.id === payload.movieId) {
      detailMovie.value = updated
    }
  } catch (err) {
    patchError.value = formatClientError(err, t("detail.errUserTags"))
    console.error("[DetailView] update user tags failed", err)
  }
}

const updateMetadataTags = async (payload: { movieId: string; tags: string[] }) => {
  patchError.value = ""
  try {
    const updated = await libraryService.patchMovie(payload.movieId, {
      metadataTags: payload.tags,
    })
    if (updated && detailMovie.value?.id === payload.movieId) {
      detailMovie.value = updated
    }
  } catch (err) {
    patchError.value = formatClientError(err, t("detail.errMetadataTags"))
    console.error("[DetailView] update metadata tags failed", err)
  }
}

const patchMovieDisplay = async (body: PatchMovieBody, done: (err?: unknown) => void) => {
  const id = detailMovie.value?.id
  if (!id) {
    done(new Error("no movie"))
    return
  }
  patchError.value = ""
  try {
    const updated = await libraryService.patchMovie(id, body)
    if (updated && detailMovie.value?.id === id) {
      detailMovie.value = updated
    }
    done()
  } catch (err) {
    patchError.value = formatClientError(err, t("detail.errMovieEdit"))
    console.error("[DetailView] patch movie display failed", err)
    done(err)
  }
}

const browseByTag = async (payload: { tag: string }) => {
  const tag = payload.tag.trim()
  if (!tag) {
    return
  }
  const sourceMode = getBrowseSourceMode(route.query)
  await router.push({
    name: getDetailBrowseTargetMode(sourceMode, "tag"),
    query: mergeLibraryQuery(route.query, {
      from: undefined,
      tag,
      q: undefined,
      actor: undefined,
      studio: undefined,
      tab: "all",
      selected: undefined,
    }),
  })
}

const browseByActor = async (payload: { actor: string }) => {
  const actor = payload.actor.trim()
  if (!actor) {
    return
  }
  const sourceMode = getBrowseSourceMode(route.query)
  await router.push({
    name: getDetailBrowseTargetMode(sourceMode, "actor"),
    query: mergeLibraryQuery(route.query, {
      from: undefined,
      actor,
      q: undefined,
      tag: undefined,
      studio: undefined,
      tab: "all",
      selected: undefined,
    }),
  })
}

const browseByStudio = async (payload: { studio: string }) => {
  const studio = payload.studio.trim()
  if (!studio) {
    return
  }
  const sourceMode = getBrowseSourceMode(route.query)
  await router.push({
    name: getDetailBrowseTargetMode(sourceMode, "studio"),
    query: mergeLibraryQuery(route.query, {
      from: undefined,
      studio,
      q: undefined,
      tag: undefined,
      actor: undefined,
      tab: "all",
      selected: undefined,
    }),
  })
}

const handleDeleteMovie = async (id: string) => {
  deleteError.value = ""
  deleteBusy.value = true
  try {
    await libraryService.deleteMovie(id)
    await router.replace({
      name: getBrowseSourceMode(route.query),
      query: mergeLibraryQuery(route.query, { selected: undefined }),
    })
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : t("detail.deleteFailGeneric")
    deleteError.value = message
    console.error("[DetailView] move to trash failed", err)
  } finally {
    deleteBusy.value = false
  }
}

const handleRestoreMovie = async (id: string) => {
  restoreError.value = ""
  restoreBusy.value = true
  try {
    await libraryService.restoreMovie(id)
    if (USE_WEB_API) {
      const loaded = await loadMovieDetail(id)
      if (loaded) {
        detailMovie.value = loaded
      } else {
        detailMovie.value = libraryService.getMovieById(id)
      }
    } else {
      detailMovie.value = libraryService.getMovieById(id)
    }
  } catch (err) {
    restoreError.value = formatClientError(err, t("detail.restoreFailGeneric"))
    console.error("[DetailView] restore movie failed", err)
  } finally {
    restoreBusy.value = false
  }
}

const handleDeleteMoviePermanently = async (id: string) => {
  permanentDeleteError.value = ""
  permanentDeleteBusy.value = true
  try {
    await libraryService.deleteMoviePermanently(id)
    await router.replace({
      name: "trash",
      query: mergeLibraryQuery(route.query, { selected: undefined }),
    })
  } catch (err) {
    permanentDeleteError.value = formatClientError(err, t("detail.permanentDeleteFailGeneric"))
    console.error("[DetailView] permanent delete failed", err)
  } finally {
    permanentDeleteBusy.value = false
  }
}

const handleRefreshMetadata = async (id: string) => {
  metadataRefreshError.value = ""
  metadataRefreshBusy.value = true
  try {
    const task = await libraryService.refreshMovieMetadata(id)
    if (!task?.taskId) {
      metadataRefreshError.value = USE_WEB_API
        ? t("detail.refreshTaskFail")
        : t("detail.refreshMockMode")
      return
    }
    scanTaskTracker.start(task.taskId)
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : t("detail.refreshFailGeneric")
    metadataRefreshError.value = message
    console.error("[DetailView] refresh metadata failed", err)
  } finally {
    metadataRefreshBusy.value = false
  }
}

const handleRevealInFileManager = async (id: string) => {
  try {
    await libraryService.revealMovieInFileManager(id)
    pushAppToast(t("detail.revealSuccess"), { variant: "success", durationMs: 3200 })
  } catch (err) {
    if (err instanceof Error && err.message === "MOCK_REVEAL_NOT_SUPPORTED") {
      pushAppToast(t("detail.revealMockMode"), { variant: "warning" })
      return
    }
    const message = formatClientError(err, t("detail.revealFailGeneric"))
    pushAppToast(message, { variant: "destructive" })
    console.error("[DetailView] reveal in file manager failed", err)
  }
}
</script>

<template>
  <div class="h-full min-w-0 w-full overflow-y-auto pr-2">
    <div
      v-if="detailLoading"
      class="rounded-3xl border border-border/70 bg-card/80 p-8 text-sm text-muted-foreground"
    >
      {{ t("detail.loadingDetail") }}
    </div>
    <template v-else-if="detailMovie">
      <p
        v-if="patchError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ patchError }}
      </p>
      <p
        v-if="deleteBusy"
        class="mb-3 rounded-2xl border border-border/70 bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
      >
        {{ t("detail.movingToTrash") }}
      </p>
      <p
        v-if="deleteError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ deleteError }}
      </p>
      <p
        v-if="restoreBusy"
        class="mb-3 rounded-2xl border border-border/70 bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
      >
        {{ t("detail.restoringMovie") }}
      </p>
      <p
        v-if="restoreError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ restoreError }}
      </p>
      <p
        v-if="permanentDeleteBusy"
        class="mb-3 rounded-2xl border border-border/70 bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
      >
        {{ t("detail.permanentDeletingMovie") }}
      </p>
      <p
        v-if="permanentDeleteError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ permanentDeleteError }}
      </p>
      <p
        v-if="metadataRefreshError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ metadataRefreshError }}
      </p>
      <DetailPage
        :movie="detailMovie"
        :related-movies="relatedMovies"
        :user-tag-suggestions="userTagSuggestionPool"
        :metadata-refresh-busy="metadataRefreshBusy"
        @select="selectMovie"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
        @update-user-rating="updateUserRating"
        @update-user-tags="updateUserTags"
        @browse-by-tag="browseByTag"
        @browse-by-actor="browseByActor"
        @browse-by-studio="browseByStudio"
        @update-metadata-tags="updateMetadataTags"
        @delete-movie="handleDeleteMovie"
        @restore-movie="handleRestoreMovie"
        @delete-movie-permanently="handleDeleteMoviePermanently"
        @refresh-metadata="handleRefreshMetadata"
        @reveal-in-file-manager="handleRevealInFileManager"
        @patch-movie-display="patchMovieDisplay"
      />
    </template>
    <NotFoundState
      v-else
      :title="t('detail.notFoundTitle')"
      :description="detailLoadError || t('detail.notFoundDescFallback')"
    />
  </div>
</template>
