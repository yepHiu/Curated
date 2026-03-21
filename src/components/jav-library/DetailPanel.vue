<script setup lang="ts">
import { useFocusWithin } from "@vueuse/core"
import { computed, nextTick, ref, watch } from "vue"
import { MoreVertical, PlayCircle, Plus, RefreshCw, Star, X } from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Separator } from "@/components/ui/separator"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import MovieRatingStars from "@/components/jav-library/MovieRatingStars.vue"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"

const props = withDefaults(
  defineProps<{
    movie: Movie
    /** 添加用户标签时的联想候选 */
    userTagSuggestions?: readonly string[]
    compact?: boolean
    showActions?: boolean
    metadataRefreshBusy?: boolean
  }>(),
  {
    userTagSuggestions: () => [],
    compact: false,
    showActions: true,
    metadataRefreshBusy: false,
  },
)

const deleteConfirmOpen = ref(false)
const newUserTagDraft = ref("")
const userTagFormError = ref("")
/** 是否展开「添加标签」内联输入（与标签同一行） */
const userTagInputOpen = ref(false)
const newUserTagInputRef = ref<HTMLInputElement | null>(null)
const userTagSuggestRootRef = ref<HTMLElement | null>(null)
const { focused: userTagSuggestRowFocused } = useFocusWithin(userTagSuggestRootRef)

const filteredUserTagSuggestions = computed(() =>
  filterUserTagSuggestions(
    props.userTagSuggestions,
    newUserTagDraft.value,
    new Set(props.movie.userTags),
    { limit: 10 },
  ),
)

const showUserTagSuggestions = computed(
  () =>
    userTagInputOpen.value &&
    userTagSuggestRowFocused.value &&
    newUserTagDraft.value.trim() !== "" &&
    filteredUserTagSuggestions.value.length > 0,
)

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
  /** 整表替换元数据/NFO 标签（删除单个时传过滤后的数组） */
  updateMetadataTags: [payload: { movieId: string; tags: string[] }]
  deleteMovie: [movieId: string]
  refreshMetadata: [movieId: string]
}>()

watch(
  () => props.movie.id,
  () => {
    newUserTagDraft.value = ""
    userTagFormError.value = ""
    userTagInputOpen.value = false
  },
)

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

const confirmDeleteMovie = () => {
  deleteConfirmOpen.value = false
  emit("deleteMovie", props.movie.id)
}

/** 详情页优先展示封面，其次缩略图 */
const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl || "")

const maxUserTags = 64
const maxUserTagRunes = 64

function cancelUserTagInput() {
  userTagInputOpen.value = false
  newUserTagDraft.value = ""
  userTagFormError.value = ""
}

async function onUserTagAddButtonClick() {
  userTagFormError.value = ""
  if (!userTagInputOpen.value) {
    userTagInputOpen.value = true
    await nextTick()
    newUserTagInputRef.value?.focus()
    return
  }
  const t = newUserTagDraft.value.trim()
  if (!t) {
    return
  }
  addUserTag()
}

function addUserTag() {
  userTagFormError.value = ""
  const t = newUserTagDraft.value.trim()
  if (!t) {
    return
  }
  if ([...t].length > maxUserTagRunes) {
    userTagFormError.value = `单个标签最多 ${maxUserTagRunes} 个字符`
    return
  }
  if (props.movie.userTags.includes(t)) {
    newUserTagDraft.value = ""
    return
  }
  if (props.movie.userTags.length >= maxUserTags) {
    userTagFormError.value = `最多 ${maxUserTags} 个用户标签`
    return
  }
  emit("updateUserTags", {
    movieId: props.movie.id,
    tags: [...props.movie.userTags, t],
  })
  newUserTagDraft.value = ""
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

function removeMetadataTag(tag: string) {
  emit("updateMetadataTags", {
    movieId: props.movie.id,
    tags: props.movie.tags.filter((x) => x !== tag),
  })
}

function pickUserTagSuggestion(tag: string) {
  newUserTagDraft.value = tag
  userTagFormError.value = ""
  void nextTick(() => newUserTagInputRef.value?.focus())
}
</script>

<template>
  <Card class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
    <CardContent
      class="grid w-full min-w-0 gap-6 overflow-x-hidden p-5 sm:p-6"
      :class="
        props.compact
          ? 'lg:grid-cols-[minmax(0,12.5rem)_minmax(0,1fr)]'
          : 'lg:grid-cols-[minmax(0,30rem)_minmax(0,1fr)] xl:grid-cols-[minmax(0,34rem)_minmax(0,1fr)]'
      "
    >
      <div
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
            :alt="`${movie.code} cover`"
            layout="intrinsic"
            class="relative z-0"
          />
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/55 via-transparent to-black/30"
            aria-hidden="true"
          />

          <div class="pointer-events-none absolute inset-x-0 top-0 z-[2] flex justify-start p-4">
            <Badge
              class="pointer-events-auto w-fit rounded-full border border-border/40 bg-background/90 text-foreground shadow-sm backdrop-blur-sm hover:bg-background/90"
            >
              {{ movie.code }}
            </Badge>
          </div>
        </div>

        <div class="mt-3 rounded-2xl border border-border/70 bg-background/50 p-3">
          <p class="text-xs text-muted-foreground">评分</p>
          <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-semibold">
            <Star class="size-4 shrink-0 text-primary" aria-hidden="true" />
            <span>综合 {{ movie.rating.toFixed(1) }}/5</span>
            <span
              v-if="siteRatingLabel !== null"
              class="text-xs font-normal text-muted-foreground"
            >
              · 站点 {{ siteRatingLabel }}/5
            </span>
            <span
              v-if="hasUserRatingOverride"
              class="text-[0.65rem] font-normal text-primary/90"
            >
              已用本地分
            </span>
          </p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <span class="text-xs text-muted-foreground">我的</span>
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
              清除
            </Button>
          </div>
        </div>
      </div>

      <div class="flex min-w-0 max-w-full flex-col gap-5">
        <div class="flex min-w-0 max-w-full flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
          <div class="min-w-0 max-w-full flex-1">
            <CardTitle
              :class="[
                props.compact ? 'text-2xl' : 'text-2xl sm:text-3xl',
                'break-words',
              ]"
            >
              {{ movie.title }}
            </CardTitle>
            <CardDescription class="text-sm text-muted-foreground sm:text-base">
              {{ movie.studio }} · {{ movie.year }} · {{ movie.resolution }}
            </CardDescription>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="shrink-0 rounded-xl"
                aria-label="更多操作"
              >
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[11rem]">
              <DropdownMenuGroup>
                <DropdownMenuItem
                  :disabled="props.metadataRefreshBusy"
                  @click="emit('refreshMetadata', movie.id)"
                >
                  <RefreshCw
                    class="size-4 shrink-0"
                    :class="props.metadataRefreshBusy ? 'animate-spin' : ''"
                    aria-hidden="true"
                  />
                  {{ props.metadataRefreshBusy ? "正在提交刮削…" : "刷新元数据" }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  @click="deleteConfirmOpen = true"
                >
                  删除影片
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <Dialog v-model:open="deleteConfirmOpen">
            <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
              <DialogHeader>
                <DialogTitle>删除影片</DialogTitle>
                <DialogDescription class="text-pretty">
                  将从资料库中移除该片的记录（演员、标签、媒体资源等关联数据）。系统会尝试删除本地主视频、已下载的刮削文件，以及同目录下的
                  <span class="font-mono text-xs">movie.nfo</span>。若文件正被占用或路径异常，可能需稍后手动清理磁盘。
                </DialogDescription>
              </DialogHeader>
              <DialogFooter class="gap-2 sm:gap-0">
                <DialogClose as-child>
                  <Button type="button" variant="outline" class="rounded-2xl">
                    取消
                  </Button>
                </DialogClose>
                <Button
                  type="button"
                  variant="destructive"
                  class="rounded-2xl"
                  @click="confirmDeleteMovie"
                >
                  确认删除
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        <p class="text-sm leading-6 text-muted-foreground">{{ movie.summary }}</p>

        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-1">
            <p class="text-sm font-medium">元数据标签</p>
            <p class="text-xs text-muted-foreground">
              点击文字可筛选同名影片；「×」仅从本地库移除该 NFO 标签（不影响「我的标签」）。重新刮削元数据可能再次写入站方标签。
            </p>
          </div>
          <p v-if="movie.tags.length === 0" class="text-sm text-muted-foreground">暂无刮削标签</p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="tag in movie.tags"
              :key="`meta-${tag}`"
              variant="secondary"
              as-child
              class="rounded-full border border-border/60 bg-secondary/70 pl-2 pr-1"
            >
              <span class="inline-flex max-w-full items-center gap-0.5 rounded-[inherit] py-0.5 pl-1">
                <button
                  type="button"
                  class="min-w-0 max-w-[12rem] cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium transition hover:bg-secondary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                  :aria-label="`在资料库中搜索：${tag}`"
                  @click="browseByTagLabel(tag)"
                >
                  {{ tag }}
                </button>
                <button
                  type="button"
                  class="inline-flex size-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                  :aria-label="`从库中移除 NFO 标签 ${tag}`"
                  @click.stop="removeMetadataTag(tag)"
                >
                  <X class="size-3.5" />
                </button>
              </span>
            </Badge>
          </div>
        </div>

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">我的标签</p>
          <p class="text-xs text-muted-foreground">
            仅保存在本地库中，重新刮削元数据不会覆盖；与上方元数据标签可同名共存。点击标签文字按<strong>同名</strong>在资料库中筛选。
          </p>
          <div class="flex flex-wrap items-center gap-2">
            <Badge
              v-for="tag in movie.userTags"
              :key="`user-${tag}`"
              variant="outline"
              as-child
              class="group rounded-full border-primary/35 bg-primary/5 pl-2 pr-1 text-foreground"
            >
              <span class="inline-flex max-w-full items-center gap-0.5 rounded-[inherit] py-0.5 pl-1">
                <button
                  type="button"
                  class="min-w-0 max-w-[12rem] cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                  :aria-label="`在资料库中搜索：${tag}`"
                  @click="browseByTagLabel(tag)"
                >
                  {{ tag }}
                </button>
                <button
                  type="button"
                  class="inline-flex size-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                  :aria-label="`移除标签 ${tag}`"
                  @click.stop="removeUserTag(tag)"
                >
                  <X class="size-3.5" />
                </button>
              </span>
            </Badge>

            <div class="flex max-w-full flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="secondary"
                class="shrink-0 rounded-2xl"
                @click="onUserTagAddButtonClick"
              >
                <Plus data-icon="inline-start" />
                添加
              </Button>
              <div
                v-if="userTagInputOpen"
                ref="userTagSuggestRootRef"
                class="relative max-w-full min-w-[min(100%,12rem)]"
              >
                <div
                  class="flex h-9 w-full items-center gap-0.5 rounded-2xl border border-border/80 bg-background/80 pl-3 pr-0.5 shadow-sm"
                >
                  <input
                    ref="newUserTagInputRef"
                    v-model="newUserTagDraft"
                    type="text"
                    maxlength="64"
                    autocomplete="off"
                    placeholder="新标签…"
                    class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
                    @keydown.enter.prevent="addUserTag"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                    aria-label="取消添加标签"
                    @click="cancelUserTagInput"
                  >
                    <X class="size-4" />
                  </Button>
                </div>
                <ul
                  v-if="showUserTagSuggestions"
                  class="absolute top-full left-0 z-50 mt-1 max-h-60 w-full min-w-[min(100%,12rem)] overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
                  role="listbox"
                  aria-label="标签建议"
                >
                  <li v-for="s in filteredUserTagSuggestions" :key="s">
                    <button
                      type="button"
                      role="option"
                      class="w-full truncate px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                      @mousedown.prevent="pickUserTagSuggestion(s)"
                    >
                      {{ s }}
                    </button>
                  </li>
                </ul>
              </div>
            </div>
          </div>
          <p v-if="userTagFormError" class="text-sm text-destructive">{{ userTagFormError }}</p>
        </div>

        <Separator />

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">Cast</p>
          <div class="flex flex-wrap gap-3">
            <div
              v-for="actor in movie.actors"
              :key="actor"
              class="flex w-[15rem] max-w-full rounded-2xl border border-border/70 bg-background/50 p-3"
            >
              <button
                type="button"
                class="flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl py-0.5 text-left transition-colors hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                :aria-label="`在资料库中筛选演员：${actor}`"
                @click="browseByActorName(actor)"
              >
                <Avatar class="size-10 shrink-0 border border-border/70">
                  <AvatarFallback class="bg-primary/15 text-primary">
                    {{ actorInitials(actor) }}
                  </AvatarFallback>
                </Avatar>
                <span class="min-w-0 truncate text-sm font-medium">{{ actor }}</span>
              </button>
            </div>
          </div>
        </div>

        <div v-if="props.showActions" class="flex flex-wrap items-center gap-3">
          <Button class="rounded-2xl" @click="emit('openPlayer', movie.id)">
            <PlayCircle data-icon="inline-start" />
            播放
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
