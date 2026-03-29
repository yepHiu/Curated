<script setup lang="ts">
import { useFocusWithin, useResizeObserver, onClickOutside } from "@vueuse/core"
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"
import { Plus, X } from "lucide-vue-next"
import type { ActorListItemDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { useUserTagSuggestKeyboard } from "@/composables/use-user-tag-suggest-keyboard"
import { mergeLibraryQuery } from "@/lib/library-query"
import { filterUserTagSuggestions } from "@/lib/user-tag-suggestions"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    actor: ActorListItemDTO
    /** 与详情页「我的标签」同源：影片 userTags + 当前列表演员标签去重 */
    userTagSuggestions?: readonly string[]
  }>(),
  {
    userTagSuggestions: () => [],
  },
)

const emit = defineEmits<{
  "tags-updated": [payload: ActorListItemDTO]
  "filter-by-actor-tag": [payload: { tag: string }]
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
const patching = ref(false)

const userTags = computed(() => props.actor.userTags ?? [])

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

/** 联想列表挂到 body + fixed，避免卡片/overflow-y-auto 裁切 */
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

watch(
  () => props.actor.name,
  () => {
    newUserTagDraft.value = ""
    userTagFormError.value = ""
    userTagInputOpen.value = false
  },
)

const initials = computed(() => {
  const n = props.actor.name.trim()
  if (!n) return "?"
  const parts = n.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0]![0]! + parts[1]![0]!).toUpperCase()
  }
  return n.slice(0, 2).toUpperCase()
})

function goFilmography() {
  void router.push({
    name: "library",
    query: mergeLibraryQuery(
      {},
      {
        actor: props.actor.name,
        q: undefined,
        tag: undefined,
        studio: undefined,
        tab: "all",
        selected: undefined,
      },
    ),
  })
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
  const x = newUserTagDraft.value.trim()
  if (!x) {
    return
  }
  void addUserTag()
}

async function patchTags(next: string[]) {
  patching.value = true
  userTagFormError.value = ""
  try {
    const updated = await libraryService.patchActorUserTags(props.actor.name, next)
    emit("tags-updated", updated)
  } catch (e) {
    userTagFormError.value = e instanceof Error ? e.message : t("actors.tagsSaveError")
  } finally {
    patching.value = false
  }
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
  await patchTags([...userTags.value, tagText])
  newUserTagDraft.value = ""
}

async function addUserTag() {
  await addUserTagWithValue(newUserTagDraft.value)
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
  await patchTags(userTags.value.filter((x) => x !== tag))
}

function onTagLabelClick(tag: string) {
  const x = tag.trim()
  if (!x) {
    return
  }
  emit("filter-by-actor-tag", { tag: x })
}

function pickUserTagSuggestion(s: string) {
  newUserTagDraft.value = s
  userTagFormError.value = ""
  void nextTick(() => newUserTagInputRef.value?.focus())
}
</script>

<template>
  <Card
    class="group flex h-full min-w-0 w-full max-w-full flex-col gap-0 overflow-visible rounded-2xl border-border/70 bg-card/90 py-0 shadow-sm transition-[box-shadow,border-color] hover:border-primary/20 hover:shadow-md"
    :class="showUserTagSuggestions ? 'relative z-30' : ''"
  >
    <!-- 仅头部裁圆角；根节点勿 overflow-hidden，否则会裁掉标签联想下拉 -->
    <button
      type="button"
      class="flex flex-col overflow-hidden rounded-t-2xl text-left focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:outline-none"
      @click="goFilmography"
    >
      <CardHeader class="gap-2 px-4 pt-3.5 pb-2.5">
        <div class="flex items-start gap-3">
          <Avatar class="size-12 shrink-0 rounded-xl border border-border/60">
            <AvatarImage
              v-if="actor.avatarUrl"
              :src="actor.avatarUrl"
              :alt="actor.name"
              class="object-cover"
            />
            <AvatarFallback
              class="rounded-xl bg-muted text-xs font-medium text-muted-foreground"
            >
              {{ initials }}
            </AvatarFallback>
          </Avatar>
          <div class="min-w-0 flex-1">
            <CardTitle class="line-clamp-2 text-base font-semibold leading-snug">
              {{ actor.name }}
            </CardTitle>
            <CardDescription class="text-xs leading-tight">
              {{ t("actors.movieCount", { n: actor.movieCount }) }}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
    </button>

    <CardContent
      class="flex flex-col gap-2 overflow-visible rounded-b-2xl border-t border-border/50 px-4 pt-2.5 pb-3.5"
      @click.stop
    >
      <!-- 与 DetailPanel「我的标签」同款字号与间距 -->
      <div class="flex flex-wrap items-center gap-2">
        <Badge
          v-for="tag in userTags"
          :key="`actor-tag-${tag}`"
          variant="outline"
          as-child
          class="group h-[29px] max-h-[29px] min-h-[29px] rounded-full border-primary/35 bg-primary/5 px-0 py-0 pl-2 pr-1 text-foreground"
        >
          <span class="inline-flex h-full max-w-full items-center gap-0.5 rounded-[inherit] pl-1">
            <button
              type="button"
              class="flex h-full min-h-0 max-w-[12rem] cursor-pointer items-center truncate rounded-md px-1.5 text-left text-xs font-medium transition hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-40"
              :aria-label="t('actors.ariaFilterActorTag', { tag })"
              :disabled="patching"
              @click="onTagLabelClick(tag)"
            >
              {{ tag }}
            </button>
            <button
              type="button"
              class="inline-flex size-[22px] shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive disabled:pointer-events-none disabled:opacity-40"
              :aria-label="t('detailPanel.ariaRemoveMyTag', { tag })"
              :disabled="patching"
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
            class="h-[29px] min-h-[29px] max-h-[29px] shrink-0 rounded-2xl px-2.5 py-0 text-xs has-[>svg]:px-2 disabled:pointer-events-none disabled:opacity-40 [&_svg:not([class*='size-'])]:size-3.5"
            :disabled="patching"
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
              class="flex h-[29px] max-h-[29px] min-h-[29px] w-full items-center gap-0.5 rounded-2xl border border-border/80 bg-background/80 pl-3 pr-0.5 shadow-sm"
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
                class="placeholder:text-muted-foreground h-full min-h-0 min-w-0 flex-1 border-0 bg-transparent px-0 text-xs shadow-none outline-none focus-visible:ring-0 disabled:opacity-50"
                :disabled="patching"
                @keydown="onTagSuggestKeydown"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="size-[25px] shrink-0 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground [&_svg]:size-3.5"
                :aria-label="t('detailPanel.ariaCancelTagInput')"
                :disabled="patching"
                @click="cancelUserTagInput"
              >
                <X class="size-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
      <p v-if="userTagFormError" class="text-sm text-destructive">{{ userTagFormError }}</p>
    </CardContent>
  </Card>

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
