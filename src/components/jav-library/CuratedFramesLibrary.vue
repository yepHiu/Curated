<script setup lang="ts">
import { useFocusWithin } from "@vueuse/core"
import { computed, nextTick, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { Camera, PlayCircle, Plus, Trash2, X } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"
import { getCuratedFrameSearchQuery, mergeCuratedFramesQuery } from "@/lib/library-query"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"
import {
  deleteCuratedFrame,
  listCuratedFramesByCapturedAtDesc,
  updateCuratedFrameTags,
} from "@/lib/curated-frames/db"
import { curatedFramesRevision } from "@/lib/curated-frames/revision"
import {
  buildCuratedFrameTagSuggestionPool,
  filterCuratedFramesByQuery,
} from "@/lib/curated-frames/search"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"
import { curatedFrameImageUrl } from "@/lib/curated-frame-image-url"
import { buildPlayerRouteFromCuratedFrame } from "@/lib/player-route"

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const maxFrameTags = 64
const maxFrameTagRunes = 64

interface RowWithUrl {
  row: CuratedFrameDbRow
  url: string
}

const rawRows = ref<CuratedFrameDbRow[]>([])
const listWithUrls = ref<RowWithUrl[]>([])

function revokeAllUrls() {
  for (const x of listWithUrls.value) {
    if (x.url.startsWith("blob:")) {
      URL.revokeObjectURL(x.url)
    }
  }
  listWithUrls.value = []
}

async function reloadFromDb() {
  rawRows.value = await listCuratedFramesByCapturedAtDesc()
}

watch(
  () => curatedFramesRevision.value,
  () => {
    void reloadFromDb()
  },
  { immediate: true },
)

watch(
  [rawRows, () => getCuratedFrameSearchQuery(route.query)],
  () => {
    revokeAllUrls()
    const q = getCuratedFrameSearchQuery(route.query)
    const filtered = filterCuratedFramesByQuery(rawRows.value, q)
    listWithUrls.value = filtered.map((row) => ({
      row,
      url: row.imageBlob ? URL.createObjectURL(row.imageBlob) : curatedFrameImageUrl(row.id),
    }))
  },
  { immediate: true, deep: true },
)

onUnmounted(() => {
  revokeAllUrls()
})

const isEmpty = computed(() => listWithUrls.value.length === 0)

const noActorLabel = computed(() => t("curated.noActor"))
const noMovieLabel = computed(() => t("curated.noMovie"))

/** 无 movieId 时归入同一组，避免 Map 用空串作 key 歧义 */
const UNKNOWN_MOVIE_KEY = "__curated_no_movie__"

const actorGroups = computed(() => {
  const none = noActorLabel.value
  const m = new Map<string, RowWithUrl[]>()
  for (const item of listWithUrls.value) {
    const acts = item.row.actors.length > 0 ? item.row.actors : [none]
    for (const a of acts) {
      const k = a.trim() || none
      if (!m.has(k)) m.set(k, [])
      m.get(k)!.push(item)
    }
  }
  for (const arr of m.values()) {
    arr.sort((x, y) => y.row.capturedAt.localeCompare(x.row.capturedAt))
  }
  return [...m.entries()].sort(([a], [b]) =>
    a.localeCompare(b, locale.value, { numeric: true }),
  )
})

interface MovieGroupSection {
  movieKey: string
  heading: string
  sub: string
  items: RowWithUrl[]
}

const movieGroups = computed((): MovieGroupSection[] => {
  void locale.value
  const none = noMovieLabel.value
  const m = new Map<string, RowWithUrl[]>()
  for (const item of listWithUrls.value) {
    const mid = item.row.movieId.trim()
    const key = mid || UNKNOWN_MOVIE_KEY
    if (!m.has(key)) {
      m.set(key, [])
    }
    m.get(key)!.push(item)
  }
  for (const arr of m.values()) {
    arr.sort((x, y) => y.row.capturedAt.localeCompare(x.row.capturedAt))
  }
  const entries = [...m.entries()].sort(([, ia], [, ib]) => {
    const ca = ia[0]?.row.capturedAt ?? ""
    const cb = ib[0]?.row.capturedAt ?? ""
    return cb.localeCompare(ca)
  })
  return entries.map(([movieKey, items]) => {
    const r = items[0]!.row
    const code = r.code.trim()
    const title = r.title.trim()
    const isUnknown = movieKey === UNKNOWN_MOVIE_KEY
    if (isUnknown) {
      const line = [code, title].filter(Boolean).join(code && title ? " · " : "")
      return { movieKey, heading: none, sub: line, items }
    }
    if (code) {
      return {
        movieKey,
        heading: code,
        sub: title && title !== code ? title : "",
        items,
      }
    }
    return {
      movieKey,
      heading: title || movieKey,
      sub: "",
      items,
    }
  })
})

const dialogOpen = ref(false)
const selected = ref<CuratedFrameRecord | null>(null)
const selectedImageUrl = ref("")
const dialogTags = ref<string[]>([])

/** 与详情页「我的标签」一致：内联添加 */
const newUserTagDraft = ref("")
const userTagFormError = ref("")
const userTagInputOpen = ref(false)
const newUserTagInputRef = ref<HTMLInputElement | null>(null)
const userTagSuggestRootRef = ref<HTMLElement | null>(null)
const { focused: userTagSuggestRowFocused } = useFocusWithin(userTagSuggestRootRef)

/** 仅萃取帧库内已出现过的标签，与影片库标签无关 */
const userTagSuggestionCandidates = computed(() => buildCuratedFrameTagSuggestionPool(rawRows.value))

const filteredUserTagSuggestions = computed(() =>
  filterUserTagSuggestions(
    userTagSuggestionCandidates.value,
    newUserTagDraft.value,
    new Set(dialogTags.value),
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

function resetTagInputState() {
  newUserTagDraft.value = ""
  userTagFormError.value = ""
  userTagInputOpen.value = false
}

function openDialog(item: RowWithUrl) {
  const { imageBlob, ...meta } = item.row
  void imageBlob
  selected.value = meta
  selectedImageUrl.value = item.url
  dialogTags.value = [...item.row.tags]
  resetTagInputState()
  dialogOpen.value = true
}

async function handleDialogOpenChange(v: boolean) {
  if (!v) {
    const id = selected.value?.id
    if (id) {
      await updateCuratedFrameTags(id, dialogTags.value)
    }
    selected.value = null
    selectedImageUrl.value = ""
    resetTagInputState()
  }
  dialogOpen.value = v
}

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
  const tagText = newUserTagDraft.value.trim()
  if (!tagText) {
    return
  }
  if ([...tagText].length > maxFrameTagRunes) {
    userTagFormError.value = t("curated.tagMaxRunes", { n: maxFrameTagRunes })
    return
  }
  if (dialogTags.value.includes(tagText)) {
    newUserTagDraft.value = ""
    return
  }
  if (dialogTags.value.length >= maxFrameTags) {
    userTagFormError.value = t("curated.tagMaxCount", { n: maxFrameTags })
    return
  }
  dialogTags.value = [...dialogTags.value, tagText]
  newUserTagDraft.value = ""
}

function removeUserTag(tag: string) {
  dialogTags.value = dialogTags.value.filter((x) => x !== tag)
}

function pickUserTagSuggestion(tag: string) {
  newUserTagDraft.value = tag
  userTagFormError.value = ""
  void nextTick(() => newUserTagInputRef.value?.focus())
}

/** 在本页用顶栏同源 cfq 筛选萃取帧，不进入影片库 tag */
async function browseCuratedFramesByTag(tag: string) {
  const t = tag.trim()
  if (!t || !selected.value) {
    return
  }
  await updateCuratedFrameTags(selected.value.id, dialogTags.value)
  dialogOpen.value = false
  selected.value = null
  selectedImageUrl.value = ""
  resetTagInputState()
  await router.push({
    name: "curated-frames",
    query: mergeCuratedFramesQuery(route.query, { cfq: t }),
  })
}

async function playFromFrame() {
  if (!selected.value) return
  await updateCuratedFrameTags(selected.value.id, dialogTags.value)
  const { movieId, positionSec } = selected.value
  selected.value = null
  selectedImageUrl.value = ""
  resetTagInputState()
  dialogOpen.value = false
  await router.push(buildPlayerRouteFromCuratedFrame(movieId, positionSec))
}

/** 删除萃取帧（确认弹窗） */
const deleteConfirmOpen = ref(false)
const deleteTargetId = ref<string | null>(null)
const deleteTargetLabel = ref("")
const deleteFrameBusy = ref(false)
const deleteFrameError = ref("")

function openDeleteConfirmForCard(item: RowWithUrl) {
  deleteFrameError.value = ""
  deleteTargetId.value = item.row.id
  deleteTargetLabel.value =
    item.row.code.trim() || item.row.title.trim().slice(0, 48) || t("curated.deleteLabel")
  deleteConfirmOpen.value = true
}

function openDeleteConfirmFromDialog() {
  if (!selected.value) return
  deleteFrameError.value = ""
  deleteTargetId.value = selected.value.id
  deleteTargetLabel.value =
    selected.value.code.trim() || selected.value.title.trim().slice(0, 48) || t("curated.deleteLabel")
  deleteConfirmOpen.value = true
}

async function executeDeleteCuratedFrame() {
  const id = deleteTargetId.value
  if (!id) return
  deleteFrameError.value = ""
  deleteFrameBusy.value = true
  try {
    await deleteCuratedFrame(id)
    deleteConfirmOpen.value = false
    deleteTargetId.value = null
    deleteTargetLabel.value = ""
    if (selected.value?.id === id) {
      selected.value = null
      selectedImageUrl.value = ""
      resetTagInputState()
      dialogOpen.value = false
    }
  } catch (err) {
    console.error("[curated-frames] delete failed", err)
    deleteFrameError.value = t("curated.deleteFailed")
  } finally {
    deleteFrameBusy.value = false
  }
}

function formatClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00"
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}

function formatCapturedAt(iso: string) {
  try {
    const d = new Date(iso)
    return d.toLocaleString(locale.value, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  } catch {
    return iso
  }
}
</script>

<template>
  <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-3 pb-8 sm:px-6">
    <div class="flex flex-col gap-2 pt-2">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t("curated.title") }}</h1>
      <p class="text-sm text-muted-foreground">
        {{ t("curated.subtitle", { key: t("curated.keyHint") }) }}
      </p>
    </div>

    <div
      v-if="isEmpty"
      class="flex flex-col items-center justify-center gap-3 rounded-3xl border border-dashed border-border/70 bg-muted/20 py-16 text-center"
    >
      <Camera class="size-12 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">{{ t("curated.empty") }}</p>
    </div>

    <Tabs v-else default-value="timeline" class="w-full gap-4">
      <TabsList class="h-auto w-fit flex-wrap rounded-2xl bg-muted/60 p-1">
        <TabsTrigger value="timeline" class="rounded-xl px-4 py-2">{{ t("curated.tabTimeline") }}</TabsTrigger>
        <TabsTrigger value="actors" class="rounded-xl px-4 py-2">{{ t("curated.tabActors") }}</TabsTrigger>
        <TabsTrigger value="movies" class="rounded-xl px-4 py-2">{{ t("curated.tabMovies") }}</TabsTrigger>
      </TabsList>

      <TabsContent value="timeline" class="mt-0 outline-none">
        <div
          class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
        >
          <div
            v-for="item in listWithUrls"
            :key="item.row.id"
            class="group relative min-w-0 overflow-hidden rounded-2xl border border-border/70 bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
          >
            <button
              type="button"
              class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              @click="openDialog(item)"
            >
              <div class="relative aspect-video w-full bg-black/80">
                <img
                  :src="item.url"
                  :alt="item.row.code"
                  class="h-full w-full object-contain"
                  loading="lazy"
                />
              </div>
              <div class="space-y-1 p-3">
                <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                </p>
              </div>
            </button>
            <Button
              type="button"
              variant="secondary"
              size="icon"
              class="absolute top-2 right-2 z-10 size-9 rounded-xl border border-border/60 bg-background/90 text-destructive opacity-100 shadow-md backdrop-blur-sm transition-opacity hover:bg-destructive/10 sm:opacity-0 sm:group-hover:opacity-100 sm:focus-visible:opacity-100"
              :title="t('curated.deleteCard')"
              :aria-label="t('curated.deleteCardAria')"
              @click.stop="openDeleteConfirmForCard(item)"
            >
              <Trash2 class="size-4" />
            </Button>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="actors" class="mt-0 outline-none">
        <div class="flex flex-col gap-8">
          <section v-for="[actor, items] in actorGroups" :key="actor">
            <h2 class="mb-3 text-lg font-semibold">{{ actor }}</h2>
            <div
              class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
            >
              <div
                v-for="item in items"
                :key="`${actor}-${item.row.id}`"
                class="group relative min-w-0 overflow-hidden rounded-2xl border border-border/70 bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
              >
                <button
                  type="button"
                  class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  @click="openDialog(item)"
                >
                  <div class="relative aspect-video w-full bg-black/80">
                    <img
                      :src="item.url"
                      :alt="item.row.code"
                      class="h-full w-full object-contain"
                      loading="lazy"
                    />
                  </div>
                  <div class="space-y-1 p-3">
                    <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                    <p class="text-xs text-muted-foreground">
                      {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                    </p>
                  </div>
                </button>
                <Button
                  type="button"
                  variant="secondary"
                  size="icon"
                  class="absolute top-2 right-2 z-10 size-9 rounded-xl border border-border/60 bg-background/90 text-destructive opacity-100 shadow-md backdrop-blur-sm transition-opacity hover:bg-destructive/10 sm:opacity-0 sm:group-hover:opacity-100 sm:focus-visible:opacity-100"
                  :title="t('curated.deleteCard')"
                  :aria-label="t('curated.deleteCardAria')"
                  @click.stop="openDeleteConfirmForCard(item)"
                >
                  <Trash2 class="size-4" />
                </Button>
              </div>
            </div>
          </section>
        </div>
      </TabsContent>

      <TabsContent value="movies" class="mt-0 outline-none">
        <div class="flex flex-col gap-8">
          <section v-for="g in movieGroups" :key="g.movieKey">
            <div class="mb-3">
              <h2 class="text-lg font-semibold">{{ g.heading }}</h2>
              <p
                v-if="g.sub"
                class="mt-0.5 line-clamp-2 text-sm text-muted-foreground"
              >
                {{ g.sub }}
              </p>
            </div>
            <div
              class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4"
            >
              <div
                v-for="item in g.items"
                :key="`${g.movieKey}-${item.row.id}`"
                class="group relative min-w-0 overflow-hidden rounded-2xl border border-border/70 bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
              >
                <button
                  type="button"
                  class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  @click="openDialog(item)"
                >
                  <div class="relative aspect-video w-full bg-black/80">
                    <img
                      :src="item.url"
                      :alt="item.row.code"
                      class="h-full w-full object-contain"
                      loading="lazy"
                    />
                  </div>
                  <div class="space-y-1 p-3">
                    <p class="line-clamp-2 text-sm font-medium">{{ item.row.title }}</p>
                    <p class="text-xs text-muted-foreground">
                      {{ item.row.code }} · {{ formatClock(item.row.positionSec) }}
                    </p>
                  </div>
                </button>
                <Button
                  type="button"
                  variant="secondary"
                  size="icon"
                  class="absolute top-2 right-2 z-10 size-9 rounded-xl border border-border/60 bg-background/90 text-destructive opacity-100 shadow-md backdrop-blur-sm transition-opacity hover:bg-destructive/10 sm:opacity-0 sm:group-hover:opacity-100 sm:focus-visible:opacity-100"
                  :title="t('curated.deleteCard')"
                  :aria-label="t('curated.deleteCardAria')"
                  @click.stop="openDeleteConfirmForCard(item)"
                >
                  <Trash2 class="size-4" />
                </Button>
              </div>
            </div>
          </section>
        </div>
      </TabsContent>
    </Tabs>

    <Dialog :open="dialogOpen" @update:open="handleDialogOpenChange">
      <!-- 覆盖 DialogContent 默认 sm:max-w-lg，否则整窗约 512px 宽，左侧预览会被压成一条 -->
      <DialogContent
        class="h-[min(94vh,960px)] max-h-[min(94vh,960px)] w-[min(98vw,92rem)] max-w-[min(98vw,92rem)] gap-0 overflow-hidden border-border/70 p-0 sm:max-w-[min(98vw,92rem)]"
      >
        <div
          class="grid h-full min-h-0 w-full grid-cols-1 grid-rows-1 md:grid-cols-[minmax(0,2.4fr)_minmax(16rem,22rem)] lg:grid-cols-[minmax(0,2.75fr)_minmax(17rem,24rem)]"
        >
          <div
            class="relative flex h-[min(52vh,560px)] w-full min-w-0 items-center justify-center bg-black md:h-full md:min-h-0"
          >
            <img
              v-if="selectedImageUrl"
              :src="selectedImageUrl"
              alt=""
              class="box-border h-full w-full object-contain p-2 sm:p-4"
            />
          </div>
          <div
            class="flex min-h-0 flex-col gap-5 overflow-y-auto border-t border-border/70 p-5 sm:p-6 md:max-h-full md:border-t-0 md:border-l"
          >
            <DialogHeader class="space-y-1.5 text-left">
              <DialogTitle class="line-clamp-3 text-lg font-semibold leading-snug sm:text-xl">
                {{ selected?.title }}
              </DialogTitle>
              <DialogDescription class="text-xs sm:text-sm">
                {{ t("curated.detailDialogDesc") }}
              </DialogDescription>
            </DialogHeader>

            <dl class="space-y-3 text-sm">
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldCode") }}</dt>
                <dd class="font-medium">{{ selected?.code }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldActors") }}</dt>
                <dd>{{ selected?.actors?.length ? selected.actors.join("、") : "—" }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldPosition") }}</dt>
                <dd>{{ selected ? formatClock(selected.positionSec) : "—" }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">{{ t("curated.fieldCapturedAt") }}</dt>
                <dd>{{ selected ? formatCapturedAt(selected.capturedAt) : "—" }}</dd>
              </div>
            </dl>

            <div class="flex flex-col gap-3">
              <p class="text-sm font-medium">{{ t("curated.tagsSectionTitle") }}</p>
              <p class="text-xs text-muted-foreground">
                {{ t("curated.tagsSectionHint") }}
              </p>
              <div class="flex flex-wrap items-center gap-2">
                <Badge
                  v-for="tag in dialogTags"
                  :key="`frame-${tag}`"
                  variant="outline"
                  as-child
                  class="group rounded-full border-primary/35 bg-primary/5 pl-2 pr-1 text-foreground"
                >
                  <span class="inline-flex max-w-full items-center gap-0.5 rounded-[inherit] py-0.5 pl-1">
                    <button
                      type="button"
                      class="min-w-0 max-w-[12rem] cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                      :aria-label="t('curated.ariaFilterInLibrary', { tag })"
                      @click="browseCuratedFramesByTag(tag)"
                    >
                      {{ tag }}
                    </button>
                    <button
                      type="button"
                      class="inline-flex size-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                      :aria-label="t('curated.ariaRemoveTag', { tag })"
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
                    {{ t("common.add") }}
                  </Button>
                  <div
                    v-if="userTagInputOpen"
                    ref="userTagSuggestRootRef"
                    class="relative max-w-full min-w-[min(100%,12rem)] flex-1 sm:flex-initial"
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
                        :placeholder="t('curated.newTagPlaceholder')"
                        class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
                        @keydown.enter.prevent="addUserTag"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                        :aria-label="t('curated.ariaCancelTagInput')"
                        @click="cancelUserTagInput"
                      >
                        <X class="size-4" />
                      </Button>
                    </div>
                    <ul
                      v-if="showUserTagSuggestions"
                      class="absolute top-full left-0 z-50 mt-1 max-h-60 w-full min-w-[min(100%,12rem)] overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
                      role="listbox"
                      :aria-label="t('curated.tagSuggestAria')"
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

            <div class="mt-auto flex flex-col gap-2 sm:flex-row sm:flex-wrap">
              <Button class="flex-1 rounded-2xl sm:min-w-[12rem]" @click="playFromFrame">
                <PlayCircle data-icon="inline-start" />
                {{ t("curated.playFromTime") }}
              </Button>
              <Button
                type="button"
                variant="outline"
                class="rounded-2xl border-destructive/50 text-destructive hover:bg-destructive/10 sm:flex-1"
                @click="openDeleteConfirmFromDialog"
              >
                <Trash2 data-icon="inline-start" class="size-4" />
                {{ t("curated.deleteThisFrame") }}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="deleteConfirmOpen">
      <DialogContent class="rounded-3xl border-border/70 sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t("curated.deleteCard") }}</DialogTitle>
          <DialogDescription class="text-pretty">
            {{ t("curated.deleteConfirm", { label: deleteTargetLabel }) }}
          </DialogDescription>
        </DialogHeader>
        <p v-if="deleteFrameError" class="text-sm text-destructive" role="alert">
          {{ deleteFrameError }}
        </p>
        <DialogFooter class="gap-3">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="rounded-2xl" :disabled="deleteFrameBusy">
              {{ t("curated.cancel") }}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            class="rounded-2xl"
            :disabled="deleteFrameBusy"
            @click="executeDeleteCuratedFrame"
          >
            {{ deleteFrameBusy ? t("curated.deleteWorking") : t("curated.confirmDelete") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
