<script setup lang="ts">
import { onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import HomeSectionRow from "@/components/jav-library/HomeSectionRow.vue"
import { useLibraryService } from "@/services/library-service"
import { useTagOrganization, organizationProgressText } from "@/composables/use-tag-organization"
import type { HomepageTopicGroup } from "@/services/contracts/topic-service"

const emit = defineEmits<{ openDetails: [id: string]; openPlayer: [id: string]; loaded: [count: number] }>()
const { t } = useI18n()
const router = useRouter()
const library = useLibraryService()
const state = useTagOrganization()
const groups = ref<HomepageTopicGroup[]>([])
const failed = ref(false)
const loading = ref(true)
const updated = ref(false)

/** 首次加载或用户主动刷新时更新海报，不在浏览中突然重排。 */
async function refresh() {
  loading.value = true
  try { groups.value = await library.getHomepageTopics(); emit("loaded", groups.value.length); failed.value = false; updated.value = false }
  catch { failed.value = true }
  finally { loading.value = false }
}
/** 隐藏只改变首页呈现，影片标签保持原样。 */
async function hide(id: string) {
  try { await library.setTopicHidden(id, true); groups.value = groups.value.filter((g) => { /* 即时移除当前组。 */ return g.topic.id !== id }); emit("loaded", groups.value.length) }
  catch { failed.value = true }
}
/** 打开稳定主题路由，影片属性筛选与源标签隔离。 */
function browse(id: string) { void router.push({ name: "library", query: { topicId: id } }) }
onMounted(() => { /* 浏览只读已有结果。 */ void refresh() })
watch(state.revision, () => { /* 通知可用更新，不抢滚动位置。 */ if (groups.value.length) updated.value = true; else void refresh() })
</script>

<template>
  <section class="flex flex-col gap-4" data-home-topics>
    <div class="flex min-h-11 flex-wrap items-center justify-between gap-x-3 gap-y-2 sm:min-h-8">
      <h2 class="text-lg font-semibold tracking-tight sm:text-xl">{{ t("topics.heading") }}</h2>
      <div class="flex flex-wrap items-center gap-2">
        <span v-if="!state.connected.value" class="text-xs text-muted-foreground">{{ t("topics.disconnected") }}</span>
        <Button v-if="state.active.value" variant="ghost" size="sm" @click="state.dialogOpen.value = true">
          {{ organizationProgressText(state.active.value) }}
        </Button>
        <Button v-if="updated" variant="outline" size="sm" @click="refresh">{{ t("topics.updated") }}</Button>
        <Button variant="ghost" size="sm" @click="state.dialogOpen.value = true">{{ t("topics.organize") }}</Button>
      </div>
    </div>
    <p v-if="loading" role="status" class="text-sm text-muted-foreground">{{ t("topics.loading") }}</p>
    <div v-else-if="failed" class="flex items-center gap-3 text-sm"><span>{{ t("topics.loadFailed") }}</span><Button variant="outline" size="sm" @click="refresh">{{ t("topics.retry") }}</Button></div>
    <p v-else-if="groups.length === 0" class="text-sm text-muted-foreground">{{ t("topics.empty") }}</p>
    <div v-if="groups.length > 0" class="flex flex-col gap-8 lg:gap-10">
      <HomeSectionRow v-for="group in groups" :key="group.topic.id" :title="group.topic.name" :movies="group.movies" @open-details="emit('openDetails', $event)" @open-player="emit('openPlayer', $event)">
        <template #action><Button variant="ghost" size="sm" class="min-h-11 sm:min-h-8" @click="hide(group.topic.id)">{{ t("topics.hide") }}</Button><Button variant="ghost" size="sm" class="min-h-11 sm:min-h-8" @click="browse(group.topic.id)">{{ t("topics.viewAll", { count: group.topic.movieCount }) }} →</Button></template>
      </HomeSectionRow>
    </div>
  </section>
</template>
