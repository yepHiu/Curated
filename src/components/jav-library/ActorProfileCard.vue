<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { HttpClientError } from "@/api/http-client"
import type { ActorProfileDTO, TaskDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
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
import {
  isValidActorExternalLink,
  normalizeActorExternalLinkDraft,
} from "@/lib/actor-external-links"
import { ACTORS_SEARCH_QUERY_KEY } from "@/lib/actors-route-query"
import { useLibraryService } from "@/services/library-service"

const useWeb = import.meta.env.VITE_USE_WEB_API === "true"

const props = withDefaults(
  defineProps<{
    actorName: string
    /** Kept for parent compatibility while actor tag UI is hidden. */
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
const libraryService = useLibraryService()
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

const externalLinks = computed(() => profile.value?.externalLinks ?? [])
const primaryExternalLink = computed(() => externalLinks.value[0]?.trim() ?? "")

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
    const task = await libraryService.getTaskStatus(taskId)
    if (isTerminalStatus(task.status)) {
      return task
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  return null
}

async function fetchProfileForSeq(seq: number, name: string): Promise<void> {
  const data = await libraryService.getActorProfile(name)
  if (seq !== loadSeq) {
    return
  }
  profile.value = data
}

function needsAutoScrape(p: ActorProfileDTO): boolean {
  return !p.avatarUrl?.trim() && !p.summary?.trim()
}

function actorNotificationSource(name: string) {
  return { route: `/actors?${ACTORS_SEARCH_QUERY_KEY}=${encodeURIComponent(name)}` }
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
    const started = await libraryService.scrapeActorProfile(name)
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
          notification: {
            type: "scrape",
            title: t("notificationCenter.titles.scrapeFailed"),
            source: actorNotificationSource(name),
          },
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
          notification: {
            type: "scrape",
            title: t("notificationCenter.titles.scrapeDone"),
            source: actorNotificationSource(name),
          },
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
        notification: {
          type: "scrape",
          title: t("notificationCenter.titles.scrapeFailed"),
          source: actorNotificationSource(name),
        },
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
    newExternalLinkDraft.value = ""
    externalLinkFormError.value = ""
    actorEditDialogOpen.value = false
    void load()
  },
)

async function patchActorExternalLinks(next: string[]) {
  const name = profile.value?.name?.trim() || props.actorName.trim()
  if (!name) {
    return false
  }
  externalLinksSaving.value = true
  externalLinkFormError.value = ""
  try {
    profile.value = await libraryService.patchActorExternalLinks(name, next)
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
</template>
