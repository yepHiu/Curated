<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  AlertTriangle,
  FolderOpen,
  Languages,
  Loader2,
  MoreVertical,
  Pencil,
  PlayCircle,
  RefreshCw,
  Star,
  Trash2,
  X,
} from "lucide-vue-next"
import type { PatchMovieBody } from "@/api/types"
import type { Movie } from "@/domain/movie/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Separator } from "@/components/ui/separator"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import MovieDeleteConfirmDialog from "@/components/jav-library/MovieDeleteConfirmDialog.vue"
import MovieEditDialog from "@/components/jav-library/MovieEditDialog.vue"
import MovieMetadataRefreshConfirmDialog from "@/components/jav-library/MovieMetadataRefreshConfirmDialog.vue"
import MovieRatingStars from "@/components/jav-library/MovieRatingStars.vue"
import ExpandableText from "@/components/jav-library/ExpandableText.vue"
import { formatMovieSummaryForDisplay } from "@/lib/format-movie-summary"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import { getMovieImageVersion } from "@/lib/image-version"
import { sourcePageLink } from "@/lib/source-page-link"
import DetailTagAddControl from "./DetailTagAddControl.vue"

const { t, locale } = useI18n()

const props = withDefaults(
  defineProps<{
    movie: Movie
    /** 添加用户标签时的联想候选 */
    userTagSuggestions?: readonly string[]
    compact?: boolean
    showActions?: boolean
    metadataRefreshBusy?: boolean
    readOnly?: boolean
    sourceUrl?: string
  }>(),
  {
    userTagSuggestions: () => [],
    compact: false,
    showActions: true,
    metadataRefreshBusy: false,
    readOnly: false,
  },
)

const deleteConfirmOpen = ref(false)
const permanentDeleteConfirmOpen = ref(false)
const metadataRefreshConfirmOpen = ref(false)

const isTrashed = computed(() => Boolean(props.movie.trashedAt?.trim()))

const summaryDisplay = computed(() =>
  formatMovieSummaryForDisplay(props.movie.summary ?? ""),
)
const { writeEnabled: agentEnabled } = useExperimentalAgent()
const aiService = useAIService()
const titleAction = useAIActionRequest(aiService)
const summaryAction = useAIActionRequest(aiService)
const translatedTitle = ref("")
const translatedSummary = ref("")
const titleTranslateError = ref("")
const summaryTranslateError = ref("")
const canTranslate = computed(() => agentEnabled.value && !props.readOnly && !isTrashed.value)
const displayedTitle = computed(() => translatedTitle.value || props.movie.title)
const displayedSummary = computed(() =>
  translatedSummary.value ? formatMovieSummaryForDisplay(translatedSummary.value) : summaryDisplay.value,
)

watch([() => props.movie.id, () => props.movie.title, () => props.movie.summary, locale, canTranslate], () => {
  titleAction.cancel()
  summaryAction.cancel()
  translatedTitle.value = ""
  translatedSummary.value = ""
  titleTranslateError.value = ""
  summaryTranslateError.value = ""
})

async function translateField(field: "title" | "summary") {
  if (!canTranslate.value) return
  const translation = field === "title" ? translatedTitle : translatedSummary
  const error = field === "title" ? titleTranslateError : summaryTranslateError
  const action = field === "title" ? titleAction : summaryAction
  if (action.pending.value) return
  if (translation.value) {
    translation.value = ""
    return
  }
  const movieId = props.movie.id
  const source = field === "title" ? props.movie.title.trim() : props.movie.summary?.trim() ?? ""
  if (!source) return
  error.value = ""
  try {
    const result = await action.run(field === "title" ? "translate_title" : "translate_summary", {
      movieId,
      body: source,
      locale: locale.value,
    })
    if (movieId !== props.movie.id || source !== (field === "title" ? props.movie.title.trim() : props.movie.summary?.trim() ?? "") || !canTranslate.value) return
    if (result.noop || !result.proposedText?.trim()) {
      error.value = t(field === "title" ? "detailPanel.movieAiTranslateNoop" : "detailPanel.movieAiTranslateSummaryNoop")
      return
    }
    translation.value = result.proposedText.trim()
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (movieId !== props.movie.id || !canTranslate.value) return
    error.value = err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE"
      ? t("detailPanel.movieAiUnconfigured")
      : err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieAiError")
  }
}

const useWebApi = import.meta.env.VITE_USE_WEB_API === "true"
const canRevealInFileManager = computed(
  () => useWebApi && Boolean(props.movie.location?.trim()) && !isTrashed.value,
)

const metadataProvider = computed(() => props.movie.metadataProvider?.trim() ?? "")
const sourceLink = computed(() => sourcePageLink(props.sourceUrl))

const javdbSearchUrl = computed(
  () => `https://javdb.com/search?q=${encodeURIComponent(props.movie.code.trim())}&f=all`,
)

const movieEditOpen = ref(false)
const emit = defineEmits<{
  openPlayer: [movieId: string]
  /** 用户评分：null 表示清除本地评分，恢复为站点评分 */
  updateUserRating: [payload: { movieId: string; value: number | null }]
  /** 整表替换用户标签（与元数据 tags 独立） */
  updateUserTags: [payload: { movieId: string; tags: string[] }]
  /** 在资料库中按标签筛选（与列表页搜索框同源） */
  browseByTag: [payload: { tag: string }]
  /** 在资料库中按演员精确筛选（URL `actor`，与列表页搜索框展示同源） */
  browseByActor: [payload: { actor: string }]
  /** 在资料库中按厂商精确筛选（URL `studio`） */
  browseByStudio: [payload: { studio: string }]
  /** 整表替换元数据/NFO 标签（删除单个时传过滤后的数组） */
  updateMetadataTags: [payload: { movieId: string; tags: string[] }]
  deleteMovie: [movieId: string]
  restoreMovie: [movieId: string]
  deleteMoviePermanently: [movieId: string]
  refreshMetadata: [movieId: string]
  revealInFileManager: [movieId: string]
  patchMovieDisplay: [body: PatchMovieBody, done: (err?: unknown) => void]
  /** 在详情页用全屏查看器打开海报（与样本图查看器一致） */
  openPosterViewer: []
}>()

watch(
  () => props.movie.id,
  () => {
    movieEditOpen.value = false
  },
)

function patchMovieDisplayFromEdit(body: PatchMovieBody, done: (err?: unknown) => void) {
  emit("patchMovieDisplay", body, done)
}

function actorAvatarSrc(name: string): string {
  return props.movie.actorAvatarUrls?.[name]?.trim() ?? ""
}

const actorInitials = (name: string) =>
  name
    .split(" ")
    .slice(0, 2)
    .map((part) => part.charAt(0))
    .join("")
    .toUpperCase()

/** 星标组件展示用：有用户分时用用户分，否则用站点/综合 */
const starDisplayValue = computed(() => {
  const m = props.movie
  if (typeof m.userRating === "number") {
    return m.userRating
  }
  return m.metadataRating ?? m.rating
})

const hasUserRatingOverride = computed(
  () => typeof props.movie.userRating === "number",
)

const siteRatingLabel = computed(() => {
  const m = props.movie.metadataRating
  if (m === undefined || m === null || m <= 0) return null
  return m.toFixed(1)
})

function commitUserRatingFromStars(value: number) {
  emit("updateUserRating", { movieId: props.movie.id, value })
}

function clearUserRating() {
  emit("updateUserRating", { movieId: props.movie.id, value: null })
}

/** 详情页优先展示封面，其次缩略图 */
const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl || "")

/** 图片版本号 - 用于强制刷新重新搜刮后的海报 */
const imageVersion = computed(() => getMovieImageVersion(props.movie.id))

function addUserTag(tag: string, done: (error?: unknown) => void) {
  emit("updateUserTags", { movieId: props.movie.id, tags: [...props.movie.userTags, tag] })
  done()
}

function removeUserTag(tag: string) {
  emit("updateUserTags", {
    movieId: props.movie.id,
    tags: props.movie.userTags.filter((x) => x !== tag),
  })
}

function browseByTagLabel(tag: string) {
  const t = tag.trim()
  if (!t) {
    return
  }
  emit("browseByTag", { tag: t })
}

function browseByActorName(actor: string) {
  const a = actor.trim()
  if (!a) {
    return
  }
  emit("browseByActor", { actor: a })
}

function browseByStudioName(studio: string) {
  const s = studio.trim()
  if (!s) {
    return
  }
  emit("browseByStudio", { studio: s })
}

function removeMetadataTag(tag: string) {
  emit("updateMetadataTags", {
    movieId: props.movie.id,
    tags: props.movie.tags.filter((x) => x !== tag),
  })
}
</script>

<template>
  <Card class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 py-0 shadow-xl shadow-black/10">
    <CardContent
      class="grid w-full min-w-0 gap-6 overflow-x-hidden p-5 sm:p-6"
      :class="
        props.compact
          ? 'lg:grid-cols-[minmax(0,12.5rem)_minmax(0,1fr)]'
          : 'lg:grid-cols-[minmax(0,30rem)_minmax(0,1fr)] xl:grid-cols-[minmax(0,34rem)_minmax(0,1fr)]'
      "
    >
      <div
        data-detail-media-column
        class="w-full min-w-0 max-w-full overflow-hidden"
        :class="
          props.compact
            ? 'mx-auto max-w-[12.5rem]'
            : 'lg:mx-auto lg:max-w-[min(100%,30rem)] xl:max-w-[min(100%,34rem)]'
        "
      >
        <!-- 不锁死竖版比例：横版整碟封套 / 竖版封面都由图片 intrinsic 高度决定，避免上下黑边 -->
        <div
          class="relative isolate w-full overflow-hidden rounded-[1.5rem] border border-border/60"
          :class="
            posterSrc
              ? 'bg-zinc-950/90'
              : `aspect-[358/537] min-h-[14rem] bg-gradient-to-br p-4 ${movie.tone}`
          "
        >
          <MediaStill
            v-if="posterSrc"
            :src="posterSrc"
            :alt="t('detailPanel.coverAlt', { code: movie.code })"
            layout="intrinsic"
            :version="imageVersion"
            class="relative z-0"
            loading="eager"
            fetch-priority="high"
          />
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/55 via-transparent to-black/30"
            aria-hidden="true"
          />

          <button
            v-if="posterSrc"
            type="button"
            class="absolute inset-0 z-[2] cursor-pointer rounded-[1.5rem] border-0 bg-transparent p-0 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-0"
            :aria-label="t('detailPanel.coverViewerAria', { code: movie.code })"
            @click="emit('openPosterViewer')"
          />

          <div class="pointer-events-none absolute inset-x-0 top-0 z-[3] flex justify-start p-4">
            <Badge
              as="a"
              variant="outline"
              :href="javdbSearchUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="pointer-events-auto w-fit select-none rounded-full border-border/40 bg-background/90 shadow-sm backdrop-blur-sm focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              {{ movie.code }}
            </Badge>
          </div>
        </div>
      </div>

      <div
        data-detail-info-column
        class="flex min-w-0 max-w-full flex-col gap-5"
      >
        <div
          v-if="isTrashed"
          class="flex gap-3 rounded-2xl border border-border/70 border-l-[3px] border-l-amber-600/45 bg-muted/25 px-4 py-3 text-sm font-medium leading-relaxed text-foreground dark:border-l-amber-500/40 dark:bg-muted/20"
          role="status"
        >
          <AlertTriangle
            class="mt-0.5 size-5 shrink-0 text-amber-700/90 dark:text-amber-500/80"
            aria-hidden="true"
          />
          <p class="min-w-0 flex-1 text-pretty text-foreground/95">
            {{ t("detailPanel.inTrashBanner") }}
          </p>
        </div>
        <div class="flex min-w-0 max-w-full flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
          <div class="min-w-0 max-w-full flex-1">
            <CardTitle
              :class="[
                props.compact ? 'text-2xl' : 'text-2xl sm:text-3xl',
                'break-words',
              ]"
            >
              <span data-detail-title>{{ displayedTitle }}</span><button
                v-if="canTranslate && movie.title.trim()"
                data-translate-title
                type="button"
                class="relative -bottom-1 ml-1 inline-flex size-7 cursor-pointer items-center justify-center rounded-md align-baseline text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-wait disabled:opacity-60"
                :disabled="titleAction.pending.value"
                :aria-label="t(translatedTitle ? 'detailPanel.showOriginalTitle' : 'detailPanel.translateTitleInline')"
                :title="t(translatedTitle ? 'detailPanel.showOriginalTitle' : 'detailPanel.translateTitleInline')"
                @click="translateField('title')"
              >
                <Loader2 v-if="titleAction.pending.value" class="size-3.5 animate-spin" aria-hidden="true" />
                <Languages v-else class="size-3.5" aria-hidden="true" />
              </button>
            </CardTitle>
            <p v-if="titleTranslateError" class="mt-1 text-xs text-destructive" role="status">{{ titleTranslateError }}</p>
            <CardDescription class="text-sm text-muted-foreground sm:text-base">
              <template v-if="movie.studio.trim()">
                <button
                  type="button"
                  class="inline cursor-pointer border-0 bg-transparent p-0 align-baseline text-primary underline-offset-2 hover:underline"
                  :aria-label="t('detailPanel.ariaFilterStudio', { studio: movie.studio })"
                  @click="browseByStudioName(movie.studio)"
                >
                  {{ movie.studio }}
                </button>
              </template>
              <template v-else>
                <span>—</span>
              </template>
              <span v-if="movie.year" aria-hidden="true"> · {{ movie.year }}</span>
              <span v-if="movie.resolution" aria-hidden="true"> · {{ movie.resolution }}</span>
              <span
                v-if="metadataProvider"
                data-metadata-provider
                :aria-label="t('detailPanel.ariaMetadataProvider', { provider: metadataProvider })"
              > · {{ metadataProvider }}</span>
              <span v-if="sourceLink" class="ml-3 inline-block">
                <a
                  data-source-page-link
                  :href="sourceLink.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="rounded-sm text-primary underline underline-offset-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >{{ sourceLink.label }}</a>
              </span>
            </CardDescription>
          </div>

          <DropdownMenu v-if="!props.readOnly && !isTrashed">
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="shrink-0 rounded-xl"
                :aria-label="t('detailPanel.moreActions')"
              >
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[11rem]">
              <DropdownMenuGroup>
                <DropdownMenuItem @click="movieEditOpen = true">
                  <Pencil
                    class="size-4 shrink-0"
                    aria-hidden="true"
                  />
                  {{ t("detailPanel.editMovie") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  :disabled="props.metadataRefreshBusy"
                  @click="metadataRefreshConfirmOpen = true"
                >
                  <RefreshCw
                    class="size-4 shrink-0"
                    :class="props.metadataRefreshBusy ? 'animate-spin' : ''"
                    aria-hidden="true"
                  />
                  {{
                    props.metadataRefreshBusy
                      ? t("detailPanel.scrapeSubmitting")
                      : t("detailPanel.refreshMetadata")
                  }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  :disabled="!canRevealInFileManager"
                  :title="
                    !useWebApi
                      ? t('detailPanel.revealInFileManagerMockHint')
                      : !movie.location?.trim()
                        ? t('detailPanel.revealInFileManagerNoPath')
                        : undefined
                  "
                  @click="emit('revealInFileManager', movie.id)"
                >
                  <FolderOpen
                    class="size-4 shrink-0"
                    aria-hidden="true"
                  />
                  {{ t("detailPanel.revealInFileManager") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  @click="deleteConfirmOpen = true"
                >
                  <Trash2
                    class="size-4 shrink-0"
                    aria-hidden="true"
                  />
                  {{ t("detailPanel.moveToTrash") }}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu v-else-if="!props.readOnly">
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="shrink-0 rounded-xl"
                :aria-label="t('detailPanel.moreActions')"
              >
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[11rem]">
              <DropdownMenuGroup>
                <DropdownMenuItem @click="emit('restoreMovie', movie.id)">
                  {{ t("detailPanel.restoreMovie") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  @click="permanentDeleteConfirmOpen = true"
                >
                  {{ t("detailPanel.deleteMoviePermanently") }}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <MovieEditDialog
            v-if="!props.readOnly"
            v-model:open="movieEditOpen"
            :movie="movie"
            :patch-movie-display="patchMovieDisplayFromEdit"
          />

          <MovieDeleteConfirmDialog
            v-if="!props.readOnly"
            v-model:open="deleteConfirmOpen"
            variant="trash"
            @confirm="emit('deleteMovie', movie.id)"
          />

          <MovieDeleteConfirmDialog
            v-if="!props.readOnly"
            v-model:open="permanentDeleteConfirmOpen"
            variant="permanent"
            @confirm="emit('deleteMoviePermanently', movie.id)"
          />

          <MovieMetadataRefreshConfirmDialog
            v-if="!props.readOnly"
            v-model:open="metadataRefreshConfirmOpen"
            :movie-title="movie.title"
            @confirm="emit('refreshMetadata', movie.id)"
          />
        </div>

        <div v-if="displayedSummary" class="min-w-0">
          <div class="flex min-w-0 items-start gap-1">
            <div class="min-w-0 flex-1">
              <ExpandableText
                :text="displayedSummary"
                :collapsed-lines="5"
                :expand-label="t('detailPanel.expandSummary')"
                :collapse-label="t('detailPanel.collapseSummary')"
              />
            </div>
            <button
              v-if="canTranslate && movie.summary?.trim()"
              data-translate-summary
              type="button"
              class="inline-flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-wait disabled:opacity-60"
              :disabled="summaryAction.pending.value"
              :aria-label="t(translatedSummary ? 'detailPanel.showOriginalSummary' : 'detailPanel.translateSummaryInline')"
              :title="t(translatedSummary ? 'detailPanel.showOriginalSummary' : 'detailPanel.translateSummaryInline')"
              @click="translateField('summary')"
            >
              <Loader2 v-if="summaryAction.pending.value" class="size-3.5 animate-spin" aria-hidden="true" />
              <Languages v-else class="size-3.5" aria-hidden="true" />
            </button>
          </div>
          <p v-if="summaryTranslateError" class="mt-1 text-xs text-destructive" role="status">{{ summaryTranslateError }}</p>
        </div>

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("detailPanel.metadataTags") }}</p>
          <p v-if="movie.tags.length === 0" class="text-sm text-muted-foreground">
            {{ t("detailPanel.noMetadataTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="tag in movie.tags"
              :key="`meta-${tag}`"
              variant="secondary"
              as-child
              class="h-[29px] max-h-[29px] min-h-[29px] rounded-full border border-border/60 bg-secondary/70 py-0 pl-2 pr-1"
            >
              <span class="inline-flex h-full max-w-full items-center gap-0.5 rounded-[inherit] py-0 pl-1">
                <button
                  type="button"
                  class="flex h-full min-w-0 max-w-[12rem] cursor-pointer items-center truncate rounded-md px-1.5 text-left text-xs font-medium transition hover:bg-secondary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                  :aria-label="t('detailPanel.ariaSearchInLibrary', { tag })"
                  @click="browseByTagLabel(tag)"
                >
                  {{ tag }}
                </button>
                <button
                  v-if="!props.readOnly && !isTrashed"
                  type="button"
                  class="inline-flex size-[1.375rem] shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                  :aria-label="t('detailPanel.ariaRemoveNfoTag', { tag })"
                  @click.stop="removeMetadataTag(tag)"
                >
                  <X class="size-3" />
                </button>
              </span>
            </Badge>
          </div>
        </div>

        <div v-if="!props.readOnly" class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("detailPanel.myTags") }}</p>
          <div class="flex flex-wrap items-center gap-2">
            <Badge
              v-for="tag in movie.userTags"
              :key="`user-${tag}`"
              variant="outline"
              as-child
              class="group h-[29px] max-h-[29px] min-h-[29px] rounded-full border-primary/35 bg-primary/5 py-0 pl-2 pr-1 text-foreground"
            >
              <span class="inline-flex h-full max-w-full items-center gap-0.5 rounded-[inherit] py-0 pl-1">
                <button
                  type="button"
                  class="flex h-full min-w-0 max-w-[12rem] cursor-pointer items-center truncate rounded-md px-1.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                  :aria-label="t('detailPanel.ariaSearchInLibrary', { tag })"
                  @click="browseByTagLabel(tag)"
                >
                  {{ tag }}
                </button>
                <button
                  v-if="!isTrashed"
                  type="button"
                  class="inline-flex size-[1.375rem] shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                  :aria-label="t('detailPanel.ariaRemoveMyTag', { tag })"
                  @click.stop="removeUserTag(tag)"
                >
                  <X class="size-3" />
                </button>
              </span>
            </Badge>

            <DetailTagAddControl
              v-if="!isTrashed"
              :key="movie.id"
              :tags="movie.userTags"
              :suggestions="userTagSuggestions"
              :save-error-message="t('detail.errUserTags')"
              @add="addUserTag"
            />
          </div>
        </div>

        <div
          v-if="!props.readOnly || movie.rating > 0"
          data-detail-rating-card
          class="w-[250px] max-w-full rounded-2xl border border-border/70 bg-background/50 p-3"
        >
          <p class="text-xs text-muted-foreground">{{ t("detailPanel.rating") }}</p>
          <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-semibold">
            <Star class="size-4 shrink-0 text-primary" aria-hidden="true" />
            <span>{{ t("detailPanel.combined", { n: movie.rating.toFixed(1) }) }}</span>
            <span
              v-if="siteRatingLabel !== null"
              class="text-xs font-normal text-muted-foreground"
            >
              {{ t("detailPanel.siteDot", { n: siteRatingLabel }) }}
            </span>
            <span
              v-if="hasUserRatingOverride"
              class="text-[0.65rem] font-normal text-primary/90"
            >
              {{ t("detailPanel.usingLocalRating") }}
            </span>
          </p>
          <template v-if="!props.readOnly && !isTrashed">
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <span class="text-xs text-muted-foreground">{{ t("detailPanel.myRating") }}</span>
              <MovieRatingStars
                :model-value="starDisplayValue"
                @commit="commitUserRatingFromStars"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="h-7 rounded-lg px-2 text-xs text-muted-foreground hover:text-foreground"
                :disabled="!hasUserRatingOverride"
                @click="clearUserRating"
              >
                {{ t("detailPanel.clearLocalRating") }}
              </Button>
            </div>
          </template>
          <p v-else-if="isTrashed" class="mt-2 text-xs text-muted-foreground">
            {{ t("detailPanel.ratingLockedInTrash") }}
          </p>
        </div>

        <Separator />

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("detailPanel.cast") }}</p>
          <div class="flex flex-wrap gap-3">
            <div
              v-for="actor in movie.actors"
              :key="actor"
              class="flex w-[15rem] max-w-full rounded-2xl border border-border/70 bg-background/50 p-3"
            >
              <button
                type="button"
                class="flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl py-0.5 text-left transition-colors hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                :aria-label="t('detailPanel.ariaFilterActor', { actor })"
                @click="browseByActorName(actor)"
              >
                <Avatar class="size-10 shrink-0 border border-border/70">
                  <AvatarImage
                    v-if="actorAvatarSrc(actor)"
                    :src="actorAvatarSrc(actor)"
                    :alt="actor"
                    class="object-cover"
                  />
                  <AvatarFallback class="bg-primary/15 text-primary">
                    {{ actorInitials(actor) }}
                  </AvatarFallback>
                </Avatar>
                <span class="min-w-0 truncate text-sm font-medium">{{ actor }}</span>
              </button>
            </div>
          </div>
        </div>

        <div v-if="props.showActions && !props.readOnly" class="flex flex-wrap items-center gap-3">
          <Button class="rounded-full px-8" @click="emit('openPlayer', movie.id)">
            <PlayCircle data-icon="inline-start" />
            {{ t("detailPanel.play") }}
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
