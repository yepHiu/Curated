<script setup lang="ts">
import { useFocusWithin, useResizeObserver, onClickOutside } from "@vueuse/core"
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, ref, useId, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"
import { Plus, X } from "lucide-vue-next"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { ActorProfileDTO, TaskDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { pushAppToast } from "@/composables/use-app-toast"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import {
  isValidActorExternalLink,
  normalizeActorExternalLinkDraft,
} from "@/lib/actor-external-links"
import { mergeActorsQuery } from "@/lib/actors-route-query"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"
import { useLibraryService } from "@/services/library-service"

const useWeb = import.meta.env.VITE_USE_WEB_API === "true"

const props = withDefaults(
  defineProps<{
    actorName: string
    /** 与演员库卡、详情页「我的标签」同源联想池 */
    userTagSuggestions?: readonly string[]
  }>(),
  {
    userTagSuggestions: () => [],
  },
)

const emit = defineEmits<{
  clearFilter: []
}>()

const { t } = useI18n()
const router = useRouter()
const libraryService = useLibraryService()

const maxUserTags = 64
const maxUserTagRunes = 64

const newUserTagDraft = ref("")
const userTagFormError = ref("")
const userTagInputOpen = ref(false)
const newUserTagInputRef = ref<HTMLInputElement | null>(null)
const userTagInlineZoneRef = ref<HTMLElement | null>(null)
const userTagSuggestRootRef = ref<HTMLElement | null>(null)
const userTagSuggestListRef = ref<HTMLElement | null>(null)
const tagSuggestDomId = useId()
const { focused: userTagSuggestRowFocused } = useFocusWithin(userTagSuggestRootRef)
const tagPatching = ref(false)
const externalLinksSaving = ref(false)
const actorEditDialogOpen = ref(false)
const newExternalLinkDraft = ref("")
const externalLinkFormError = ref("")
const externalLinkInputRef = ref<HTMLInputElement | null>(null)

const profile = ref<ActorProfileDTO | null>(null)
const initialLoading = ref(false)
const loadError = ref<string | null>(null)
const notFound = ref(false)
const scraping = ref(false)
const scrapeError = ref<string | null>(null)

const userTags = computed(() => profile.value?.userTags ?? [])
const externalLinks = computed(() => profile.value?.externalLinks ?? [])
const primaryExternalLink = computed(() => externalLinks.value[0]?.trim() ?? "")

const filteredUserTagSuggestions = computed(() =>
  filterUserTagSuggestions(
    props.userTagSuggestions,
    newUserTagDraft.value,
    new Set(userTags.value),
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

const tagSuggestPanelStyle = ref<Record<string, string>>({})

function measureTagSuggestPanel() {
  const el = userTagSuggestRootRef.value
  if (!el || !showUserTagSuggestions.value) {
    return
  }
  const r = el.getBoundingClientRect()
  tagSuggestPanelStyle.value = {
    position: "fixed",
    top: `${Math.round(r.bottom + 4)}px`,
    left: `${Math.round(r.left)}px`,
    width: `${Math.max(120, Math.round(r.width))}px`,
    zIndex: "200",
    boxSizing: "border-box",
  }
}

const suggestPanelScrollCleanups: (() => void)[] = []

function detachSuggestPanelScrollListeners() {
  while (suggestPanelScrollCleanups.length) {
    suggestPanelScrollCleanups.pop()?.()
  }
}

function attachSuggestPanelScrollListeners() {
  detachSuggestPanelScrollListeners()
  const el = userTagSuggestRootRef.value
  if (!el) {
    return
  }
  const handler = () => measureTagSuggestPanel()
  let p: HTMLElement | null = el.parentElement
  while (p) {
    const oy = getComputedStyle(p).overflowY
    if (/(auto|scroll|overlay)/.test(oy)) {
      p.addEventListener("scroll", handler, { passive: true })
      suggestPanelScrollCleanups.push(() => p!.removeEventListener("scroll", handler))
    }
    p = p.parentElement
  }
  window.addEventListener("resize", handler, { passive: true })
  suggestPanelScrollCleanups.push(() => window.removeEventListener("resize", handler))
}

watch(showUserTagSuggestions, async (open) => {
  if (!open) {
    detachSuggestPanelScrollListeners()
    tagSuggestPanelStyle.value = {}
    return
  }
  await nextTick()
  measureTagSuggestPanel()
  attachSuggestPanelScrollListeners()
})

watch(
  () => [newUserTagDraft.value, filteredUserTagSuggestions.value.length] as const,
  () => {
    if (showUserTagSuggestions.value) {
      void nextTick(() => measureTagSuggestPanel())
    }
  },
)

useResizeObserver(userTagSuggestRootRef, () => {
  if (showUserTagSuggestions.value) {
    measureTagSuggestPanel()
  }
})

onBeforeUnmount(() => {
  detachSuggestPanelScrollListeners()
})

function goFilterByActorTag(tag: string) {
  const x = tag.trim()
  if (!x) return
  void router.push({ name: "actors", query: mergeActorsQuery({}, { actorTag: x }) })
}

let disposed = false
let loadSeq = 0

function isTerminalStatus(s: TaskDTO["status"]): boolean {
  return (
    s === "completed" ||
    s === "failed" ||
    s === "cancelled" ||
    s === "partial_failed"
  )
}

async function pollTaskToEnd(taskId: string): Promise<TaskDTO | null> {
  while (!disposed) {
    const task = await api.getTaskStatus(taskId)
    if (isTerminalStatus(task.status)) {
      return task
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  return null
}

async function fetchProfileForSeq(seq: number, name: string): Promise<void> {
  const data = await api.getActorProfile(name)
  if (seq !== loadSeq) {
    return
  }
  profile.value = data
}

function needsAutoScrape(p: ActorProfileDTO): boolean {
  return !p.avatarUrl?.trim() && !p.summary?.trim()
}

async function runScrapePipeline(seq: number, name: string, force: boolean): Promise<void> {
  if (seq !== loadSeq || disposed) {
    return
  }
  const isAuto = !force
  if (isAuto) {
    const p = profile.value
    if (!p || !needsAutoScrape(p)) {
      return
    }
  }
  if (isAuto) {
    pushAppToast(t("library.actorAutoScrapeToastStart", { name }), {
      durationMs: 5000,
    })
  }
  scraping.value = true
  scrapeError.value = null
  try {
    const started = await api.scrapeActorProfile(name)
    if (seq !== loadSeq) {
      return
    }
    const finalTask = await pollTaskToEnd(started.taskId)
    if (seq !== loadSeq || disposed || !finalTask) {
      return
    }
    if (finalTask.status !== "completed") {
      const msg =
        finalTask.errorMessage?.trim() ||
        finalTask.message?.trim() ||
        t("library.actorScrapeFailedGeneric")
      scrapeError.value = msg
      if (isAuto) {
        pushAppToast(t("library.actorAutoScrapeToastFail", { name, msg }), {
          variant: "destructive",
          durationMs: 6000,
        })
      }
      return
    }
    await fetchProfileForSeq(seq, name)
    if (isAuto && seq === loadSeq && !disposed) {
      const refreshed = profile.value
      if (refreshed && !needsAutoScrape(refreshed)) {
        pushAppToast(t("library.actorAutoScrapeToastDone", { name }), {
          variant: "success",
          durationMs: 4000,
        })
      }
    }
  } catch (err) {
    if (seq !== loadSeq) {
      return
    }
    const msg =
      err instanceof Error ? err.message : t("library.actorScrapeFailedGeneric")
    scrapeError.value = msg
    if (isAuto) {
      pushAppToast(t("library.actorAutoScrapeToastFail", { name, msg }), {
        variant: "destructive",
        durationMs: 6000,
      })
    }
  } finally {
    if (seq === loadSeq) {
      scraping.value = false
    }
  }
}

function manualRefreshProfile() {
  const name = props.actorName.trim()
  if (!name || !useWeb) {
    return
  }
  const seq = loadSeq
  void runScrapePipeline(seq, name, true)
}

async function load(): Promise<void> {
  if (!useWeb) {
    return
  }
  const name = props.actorName.trim()
  if (!name) {
    return
  }
  const seq = ++loadSeq
  loadError.value = null
  notFound.value = false
  scrapeError.value = null
  initialLoading.value = true
  profile.value = null
  try {
    await fetchProfileForSeq(seq, name)
  } catch (err) {
    if (seq !== loadSeq) {
      return
    }
    if (err instanceof HttpClientError && err.status === 404) {
      notFound.value = true
    } else {
      loadError.value =
        err instanceof Error ? err.message : t("library.actorProfileError")
    }
    return
  } finally {
    if (seq === loadSeq) {
      initialLoading.value = false
    }
  }
  if (seq !== loadSeq) {
    return
  }
  await runScrapePipeline(seq, name, false)
}

watch(
  () => props.actorName,
  () => {
    newUserTagDraft.value = ""
    userTagFormError.value = ""
    userTagInputOpen.value = false
    newExternalLinkDraft.value = ""
    externalLinkFormError.value = ""
    actorEditDialogOpen.value = false
    void load()
  },
)

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
  const x = newUserTagDraft.value.trim()
  if (!x) {
    return
  }
  void addUserTag()
}

async function patchActorTags(next: string[]) {
  const name = props.actorName.trim()
  if (!name) {
    return
  }
  tagPatching.value = true
  userTagFormError.value = ""
  try {
    const updated = await libraryService.patchActorUserTags(name, next)
    const cur = profile.value
    if (cur && cur.name === updated.name) {
      profile.value = { ...cur, userTags: updated.userTags }
    }
  } catch (e) {
    userTagFormError.value = e instanceof Error ? e.message : t("actors.tagsSaveError")
  } finally {
    tagPatching.value = false
  }
}

async function patchActorExternalLinks(next: string[]) {
  const name = profile.value?.name?.trim() || props.actorName.trim()
  if (!name) {
    return false
  }
  externalLinksSaving.value = true
  externalLinkFormError.value = ""
  try {
    profile.value = await api.patchActorExternalLinks(name, next)
    return true
  } catch (e) {
    if (e instanceof HttpClientError && e.status === 404) {
      const apiMessage = e.apiError?.message?.trim().toLowerCase() ?? ""
      const apiCode = e.apiError?.code?.trim() ?? ""
      if (apiCode === "COMMON_NOT_FOUND" && apiMessage === "actor not found") {
        externalLinkFormError.value = t("library.actorProfileNotFound")
      } else {
        externalLinkFormError.value = t("library.actorExternalLinksUnsupported")
      }
    } else {
      externalLinkFormError.value =
        e instanceof Error ? e.message : t("library.actorExternalLinksSaveError")
    }
  } finally {
    externalLinksSaving.value = false
  }
  return false
}

async function addUserTagWithValue(raw: string) {
  userTagFormError.value = ""
  const tagText = raw.trim()
  if (!tagText) {
    return
  }
  if ([...tagText].length > maxUserTagRunes) {
    userTagFormError.value = t("curated.tagMaxRunes", { n: maxUserTagRunes })
    return
  }
  if (userTags.value.includes(tagText)) {
    newUserTagDraft.value = ""
    return
  }
  if (userTags.value.length >= maxUserTags) {
    userTagFormError.value = t("curated.tagMaxCount", { n: maxUserTags })
    return
  }
  await patchActorTags([...userTags.value, tagText])
  newUserTagDraft.value = ""
}

async function addUserTag() {
  await addUserTagWithValue(newUserTagDraft.value)
}

async function saveActorExternalLinks() {
  externalLinkFormError.value = ""
  const next = normalizeActorExternalLinkDraft(newExternalLinkDraft.value)
  if (!next && !primaryExternalLink.value) {
    actorEditDialogOpen.value = false
    return
  }
  if (next && !isValidActorExternalLink(next)) {
    externalLinkFormError.value = t("library.actorExternalLinksInvalid")
    return
  }
  const nextLinks = next ? [next] : []
  const unchanged =
    nextLinks.length === externalLinks.value.length &&
    nextLinks.every((link, idx) => link === externalLinks.value[idx])

  if (unchanged) {
    actorEditDialogOpen.value = false
    return
  }
  const ok = await patchActorExternalLinks(nextLinks)
  if (ok) {
    actorEditDialogOpen.value = false
  }
}

function cancelActorEditDialog() {
  actorEditDialogOpen.value = false
  newExternalLinkDraft.value = ""
  externalLinkFormError.value = ""
}

async function openActorEditDialog() {
  externalLinkFormError.value = ""
  newExternalLinkDraft.value = primaryExternalLink.value
  actorEditDialogOpen.value = true
  await nextTick()
  externalLinkInputRef.value?.focus()
}

const { highlightIndex, onTagSuggestKeydown } = useUserTagSuggestKeyboard({
  showSuggestions: showUserTagSuggestions,
  suggestions: filteredUserTagSuggestions,
  listRootRef: userTagSuggestListRef,
  commitTag: (tag) => void addUserTagWithValue(tag),
  commitDraft: () => void addUserTag(),
})

onClickOutside(
  userTagInlineZoneRef,
  () => {
    if (!userTagInputOpen.value) {
      return
    }
    cancelUserTagInput()
  },
  { ignore: [userTagSuggestListRef] },
)

async function removeUserTag(tag: string) {
  await patchActorTags(userTags.value.filter((x) => x !== tag))
}

function pickUserTagSuggestion(s: string) {
  newUserTagDraft.value = s
  userTagFormError.value = ""
  void nextTick(() => newUserTagInputRef.value?.focus())
}

onMounted(() => {
  void load()
})

onUnmounted(() => {
  disposed = true
  loadSeq++
})
</script>

<template>
  <Card
    v-if="useWeb"
    class="gap-3 py-4 sm:py-5 rounded-3xl border-border/70 bg-card/85 shadow-lg shadow-black/5"
    :class="showUserTagSuggestions ? 'relative z-30' : ''"
  >
    <CardHeader class="gap-2">
      <div class="flex flex-wrap items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <CardTitle>{{ t("library.actorCardTitle") }}</CardTitle>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <Button
            v-if="profile && !initialLoading && !notFound && !loadError"
            type="button"
            variant="outline"
            size="sm"
            class="rounded-xl"
            data-actor-edit-open
            @click="openActorEditDialog"
          >
            {{ t("library.editActorInfo") }}
          </Button>
          <Button
            v-if="profile && !initialLoading && !notFound && !loadError"
            type="button"
            variant="secondary"
            size="sm"
            class="rounded-xl"
            :disabled="scraping"
            @click="manualRefreshProfile"
          >
            {{ scraping ? t("library.actorRefreshing") : t("library.actorRefreshProfile") }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="rounded-xl"
            @click="emit('clearFilter')"
          >
            {{ t("library.clearActorFilter") }}
          </Button>
        </div>
      </div>
    </CardHeader>
    <CardContent class="flex flex-col gap-3 overflow-visible sm:flex-row sm:items-start">
      <div
        v-if="initialLoading"
        class="text-sm text-muted-foreground"
      >
        {{ t("library.actorProfileLoading") }}
      </div>
      <template v-else-if="notFound">
        <p class="text-sm text-muted-foreground">
          {{ t("library.actorProfileNotFound") }}
        </p>
      </template>
      <template v-else-if="loadError">
        <p class="text-sm text-destructive">
          {{ loadError }}
        </p>
      </template>
      <template v-else-if="profile">
        <Avatar class="size-24 shrink-0 rounded-2xl border border-border/60">
          <AvatarImage
            v-if="profile.avatarUrl"
            :src="profile.avatarUrl"
            :alt="profile.name"
            class="object-cover"
          />
          <AvatarFallback class="rounded-2xl text-lg">
            {{ profile.name.slice(0, 1) }}
          </AvatarFallback>
        </Avatar>
        <div class="min-w-0 flex-1 space-y-2">
          <div>
            <p class="text-lg font-semibold tracking-tight">
              {{ profile.name }}
            </p>
            <p
              v-if="profile.summary"
              class="mt-1.5 text-sm leading-relaxed text-muted-foreground text-pretty whitespace-pre-wrap"
            >
              {{ profile.summary }}
            </p>
          </div>
          <section class="space-y-1.5">
            <div class="flex flex-wrap items-center gap-2">
              <Badge
                v-for="tag in userTags"
                :key="`profile-actor-tag-${tag}`"
                variant="outline"
                as-child
                class="group rounded-full border-primary/35 bg-primary/5 pl-2 pr-1 text-foreground"
              >
                <span class="inline-flex max-w-full items-center gap-0.5 rounded-[inherit] py-0.5 pl-1">
                  <button
                    type="button"
                    class="min-w-0 max-w-[12rem] cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-40"
                    :aria-label="t('actors.ariaFilterActorTag', { tag })"
                    :disabled="tagPatching"
                    @click="goFilterByActorTag(tag)"
                  >
                    {{ tag }}
                  </button>
                  <button
                    type="button"
                    class="inline-flex size-6 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive disabled:pointer-events-none disabled:opacity-40"
                    :aria-label="t('detailPanel.ariaRemoveMyTag', { tag })"
                    :disabled="tagPatching"
                    @click.stop="removeUserTag(tag)"
                  >
                    <X class="size-3.5" />
                  </button>
                </span>
              </Badge>
              <div ref="userTagInlineZoneRef" class="flex max-w-full flex-wrap items-center gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  class="h-[29px] min-h-[29px] max-h-[29px] shrink-0 rounded-2xl py-0 leading-none disabled:pointer-events-none disabled:opacity-40"
                  :disabled="tagPatching"
                  @click="onUserTagAddButtonClick"
                >
                  <Plus data-icon="inline-start" />
                  {{ t("common.add") }}
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
                      role="combobox"
                      :aria-expanded="showUserTagSuggestions"
                      :aria-activedescendant="
                        highlightIndex >= 0 ? `${tagSuggestDomId}-opt-${highlightIndex}` : undefined
                      "
                      aria-autocomplete="list"
                      :aria-controls="showUserTagSuggestions ? `${tagSuggestDomId}-list` : undefined"
                      :placeholder="t('detailPanel.newTagPlaceholder')"
                      class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0 disabled:opacity-50"
                      :disabled="tagPatching"
                      @keydown="onTagSuggestKeydown"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                      :aria-label="t('detailPanel.ariaCancelTagInput')"
                      :disabled="tagPatching"
                      @click="cancelUserTagInput"
                    >
                      <X class="size-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>
            <p v-if="userTagFormError" class="text-sm text-destructive">{{ userTagFormError }}</p>
          </section>
          <dl
            v-if="profile.birthday || (profile.height && profile.height > 0) || profile.homepage"
            class="grid gap-1.5 text-sm sm:grid-cols-2"
          >
            <div v-if="profile.birthday">
              <dt class="text-muted-foreground">
                {{ t("library.actorBirthday") }}
              </dt>
              <dd>{{ profile.birthday }}</dd>
            </div>
            <div v-if="profile.height && profile.height > 0">
              <dt class="text-muted-foreground">
                {{ t("library.actorHeight") }}
              </dt>
              <dd>{{ profile.height }} cm</dd>
            </div>
            <div
              v-if="profile.homepage"
              class="sm:col-span-2"
            >
              <dt class="text-muted-foreground">
                {{ t("library.actorHomepage") }}
              </dt>
              <dd class="truncate">
                <a
                  :href="profile.homepage"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-primary underline-offset-4 hover:underline"
                >
                  {{ profile.homepage }}
                </a>
              </dd>
            </div>
          </dl>
          <section v-if="primaryExternalLink" class="space-y-1">
            <p class="text-sm text-muted-foreground">
              {{ t("library.actorExternalLinks") }}
            </p>
            <p
              class="truncate text-sm"
            >
              <a
                :href="primaryExternalLink"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary underline-offset-4 hover:underline"
              >
                {{ primaryExternalLink }}
              </a>
            </p>
          </section>
          <p
            v-if="scraping"
            class="text-sm text-muted-foreground"
          >
            {{ t("library.actorScraping") }}
          </p>
          <p
            v-if="scrapeError"
            class="text-sm text-destructive"
          >
            {{ t("library.actorScrapeFailed", { msg: scrapeError }) }}
          </p>
        </div>
      </template>
    </CardContent>
  </Card>

  <Dialog v-model:open="actorEditDialogOpen">
    <DialogContent
      v-if="actorEditDialogOpen"
      data-actor-edit-dialog
      class="max-h-[min(90vh,32rem)] overflow-y-auto rounded-3xl border-border/70 sm:max-w-lg"
    >
      <DialogHeader>
        <DialogTitle>{{ t("library.editActorInfoTitle") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("library.editActorInfoDesc") }}
        </DialogDescription>
      </DialogHeader>
      <div class="flex flex-col gap-4 py-2">
        <div class="grid gap-2">
          <label class="text-sm font-medium" for="actor-edit-external-link">
            {{ t("library.actorExternalLinks") }}
          </label>
          <input
            id="actor-edit-external-link"
            ref="externalLinkInputRef"
            data-actor-edit-external-link-input
            v-model="newExternalLinkDraft"
            type="url"
            inputmode="url"
            autocomplete="off"
            :disabled="externalLinksSaving"
            :placeholder="t('library.actorExternalLinksPlaceholder')"
            class="text-foreground placeholder:text-muted-foreground flex h-10 w-full rounded-xl border border-input bg-background px-3 py-2 text-sm shadow-sm transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
            @keydown.enter.prevent="saveActorExternalLinks"
          />
          <p class="text-xs text-muted-foreground">
            {{ t("library.editActorInfoExternalLinkHint") }}
          </p>
        </div>
        <p
          v-if="externalLinkFormError"
          class="rounded-xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ externalLinkFormError }}
        </p>
      </div>
      <DialogFooter class="gap-3">
        <Button
          data-actor-edit-cancel
          type="button"
          variant="outline"
          class="rounded-2xl"
          :disabled="externalLinksSaving"
          @click="cancelActorEditDialog"
        >
          {{ t("common.cancel") }}
        </Button>
        <Button
          data-actor-edit-save
          type="button"
          class="rounded-2xl"
          :disabled="externalLinksSaving"
          @click="saveActorExternalLinks"
        >
          {{ externalLinksSaving ? t("common.saving") : t("common.save") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <Teleport to="body">
    <ul
      v-if="showUserTagSuggestions"
      :id="`${tagSuggestDomId}-list`"
      ref="userTagSuggestListRef"
      :style="tagSuggestPanelStyle"
      class="max-h-60 overflow-y-auto rounded-2xl border border-border/80 bg-popover/98 py-1 text-popover-foreground shadow-lg backdrop-blur-sm"
      role="listbox"
      :aria-label="t('detailPanel.tagSuggestAria')"
    >
      <li v-for="(s, si) in filteredUserTagSuggestions" :key="s">
        <button
          :id="`${tagSuggestDomId}-opt-${si}`"
          type="button"
          role="option"
          :data-tag-suggest-idx="si"
          class="w-full truncate px-3 py-2 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
          :class="highlightIndex === si ? 'bg-muted' : ''"
          :aria-selected="highlightIndex === si"
          @mousedown.prevent="pickUserTagSuggestion(s)"
        >
          {{ s }}
        </button>
      </li>
    </ul>
  </Teleport>
</template>
