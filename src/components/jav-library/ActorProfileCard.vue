<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { ActorProfileDTO, TaskDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { pushAppToast } from "@/composables/use-app-toast"

const useWeb = import.meta.env.VITE_USE_WEB_API === "true"

const props = defineProps<{
  actorName: string
}>()

const emit = defineEmits<{
  clearFilter: []
}>()

const { t } = useI18n()

const profile = ref<ActorProfileDTO | null>(null)
const initialLoading = ref(false)
const loadError = ref<string | null>(null)
const notFound = ref(false)
const scraping = ref(false)
const scrapeError = ref<string | null>(null)

let disposed = false
/** 递增以丢弃过期的并发 load（快速换演员、路由抖动时避免错写 profile） */
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

/** 头像与简介皆空时自动补刮（含曾写入时间戳但仍无展示内容的情况） */
function needsAutoScrape(p: ActorProfileDTO): boolean {
  return !p.avatarUrl?.trim() && !p.summary?.trim()
}

/** 发起刮削并轮询任务；force 时跳过 needsAutoScrape（手动刷新） */
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
    void load()
  },
)

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
    class="rounded-3xl border-border/70 bg-card/85 shadow-lg shadow-black/5"
  >
    <CardHeader class="gap-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 flex-1 space-y-1">
          <CardTitle>{{ t("library.actorCardTitle") }}</CardTitle>
          <CardDescription class="text-pretty">
            {{ t("library.actorCardDesc") }}
          </CardDescription>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2">
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
    <CardContent class="flex flex-col gap-4 sm:flex-row sm:items-start">
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
        <div class="min-w-0 flex-1 space-y-3">
          <div>
            <p class="text-lg font-semibold tracking-tight">
              {{ profile.name }}
            </p>
            <p
              v-if="profile.summary"
              class="mt-2 text-sm leading-relaxed text-muted-foreground text-pretty whitespace-pre-wrap"
            >
              {{ profile.summary }}
            </p>
          </div>
          <dl
            v-if="profile.birthday || (profile.height && profile.height > 0) || profile.homepage"
            class="grid gap-2 text-sm sm:grid-cols-2"
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
</template>
