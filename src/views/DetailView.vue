<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { HttpClientError } from "@/api/http-client"
import type { TaskDTO } from "@/api/types"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"
import { buildMovieRouteQuery, getBrowseSourceMode, mergeLibraryQuery } from "@/lib/library-query"
import { loadMovieDetail } from "@/services/adapters/web/web-library-service"
import { useLibraryService } from "@/services/library-service"
import type { Movie } from "@/domain/movie/types"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

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
            detailLoadError.value =
              "无法从服务器加载本片详情，请确认后端在运行（默认 :8080）且 Vite 已将 /api 代理到后端。"
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
    const loaded = await loadMovieDetail(id)
    if (loaded) {
      detailMovie.value = loaded
    }
  } else if (task.status === "failed" || task.status === "partial_failed") {
    metadataRefreshError.value =
      task.errorMessage?.trim() || "元数据刮削失败，详情未更新"
  }
})

const relatedMovies = computed(() =>
  detailMovie.value ? libraryService.getRelatedMovies(detailMovie.value.id) : [],
)

const selectMovie = async (nextMovieId: string) => {
  await router.replace({
    name: "detail",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const openDetails = async (nextMovieId: string) => {
  await router.push({
    name: "detail",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const openPlayer = async (nextMovieId: string) => {
  await router.push({
    name: "player",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const toggleFavorite = async (payload: { movieId: string; nextValue: boolean }) => {
  patchError.value = ""
  try {
    const updated = await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
    if (updated && detailMovie.value?.id === payload.movieId) {
      detailMovie.value = updated
    }
  } catch (err) {
    patchError.value = formatClientError(err, "更新收藏失败，请检查网络与后端日志。")
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
    patchError.value = formatClientError(err, "更新评分失败，请检查网络与后端日志。")
    console.error("[DetailView] update user rating failed", err)
  }
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
          : "删除失败"
    deleteError.value = message
    console.error("[DetailView] delete movie failed", err)
  } finally {
    deleteBusy.value = false
  }
}

const handleRefreshMetadata = async (id: string) => {
  metadataRefreshError.value = ""
  metadataRefreshBusy.value = true
  try {
    const task = await libraryService.refreshMovieMetadata(id)
    if (!task?.taskId) {
      metadataRefreshError.value = USE_WEB_API
        ? "无法启动刷新任务"
        : "本地演示模式不支持刷新元数据，请启用后端 API（VITE_USE_WEB_API）"
      return
    }
    scanTaskTracker.start(task.taskId)
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : "刷新失败"
    metadataRefreshError.value = message
    console.error("[DetailView] refresh metadata failed", err)
  } finally {
    metadataRefreshBusy.value = false
  }
}
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <div
      v-if="detailLoading"
      class="rounded-3xl border border-border/70 bg-card/80 p-8 text-sm text-muted-foreground"
    >
      正在加载详情与预览图…
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
        正在删除影片…
      </p>
      <p
        v-if="deleteError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ deleteError }}
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
        :metadata-refresh-busy="metadataRefreshBusy"
        @select="selectMovie"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
        @update-user-rating="updateUserRating"
        @delete-movie="handleDeleteMovie"
        @refresh-metadata="handleRefreshMetadata"
      />
    </template>
    <NotFoundState
      v-else
      title="未找到影片"
      :description="
        detailLoadError ||
        '当前库中不存在该条目，或详情接口暂时不可用。'
      "
    />
  </div>
</template>
