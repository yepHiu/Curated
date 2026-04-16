<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import type { HomepagePortalModel, HomepageTasteEntry } from "@/lib/homepage-portal"
import HomeContinueRow from "@/components/jav-library/HomeContinueRow.vue"
import HomeHeroCarousel from "@/components/jav-library/HomeHeroCarousel.vue"
import HomeSectionRow from "@/components/jav-library/HomeSectionRow.vue"
import { useHomeScrollPreserve } from "@/composables/use-home-scroll-preserve"

const props = defineProps<{
  model: HomepagePortalModel
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  browseTaste: [payload: { kind: HomepageTasteEntry["kind"]; label: string }]
}>()

const { t } = useI18n()
const homeScrollRegionRef = ref<HTMLElement | null>(null)
const { persist } = useHomeScrollPreserve({ scrollElRef: homeScrollRegionRef })

const recommendationMovies = computed(() =>
  props.model.recommendations.map((entry) => entry.movie),
)

const tasteGroups = computed(() => {
  const rows = new Map<"actor" | "tag" | "studio", string[]>()

  for (const item of props.model.tasteRadar) {
    const current = rows.get(item.kind) ?? []
    current.push(item.label)
    rows.set(item.kind, current)
  }

  return [
    { kind: "actor" as const, title: t("home.tasteActorsTitle"), values: rows.get("actor") ?? [] },
    { kind: "tag" as const, title: t("home.tasteTagsTitle"), values: rows.get("tag") ?? [] },
    { kind: "studio" as const, title: t("home.tasteStudiosTitle"), values: rows.get("studio") ?? [] },
  ]
})

function onHomeScroll() {
  persist()
}
</script>

<template>
  <div
    ref="homeScrollRegionRef"
    data-home-scroll-region
    class="h-full min-h-0 overflow-y-auto bg-background text-foreground"
    @scroll.passive="onHomeScroll"
  >
    <HomeHeroCarousel
      :movies="model.heroMovies"
      @open-details="emit('openDetails', $event)"
      @open-player="emit('openPlayer', $event)"
    />

    <div
      class="mx-auto flex w-full max-w-[1680px] flex-col gap-8 px-4 py-6 sm:px-5 lg:gap-10 lg:px-6 lg:py-8 xl:px-8"
    >
      <HomeSectionRow
        :title="t('home.sectionRecentTitle')"
        :subtitle="t('home.sectionRecentBody')"
        :movies="model.recentMovies"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />

      <HomeSectionRow
        :title="t('home.sectionRecommendTitle')"
        :subtitle="t('home.sectionRecommendBody')"
        :movies="recommendationMovies"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />

      <HomeContinueRow
        :entries="model.continueWatching"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />

      <section class="space-y-4">
        <div class="space-y-1">
          <h2 class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
            {{ t("home.sectionTasteTitle") }}
          </h2>
          <p class="text-sm text-muted-foreground">
            {{ t("home.sectionTasteBody") }}
          </p>
        </div>

        <div class="grid gap-3 lg:grid-cols-3">
          <article
            v-for="group in tasteGroups"
            :key="group.kind"
            class="rounded-[1.6rem] border border-border/60 bg-card/65 p-4"
          >
            <h3 class="text-sm font-medium tracking-[0.16em] text-muted-foreground uppercase">
              {{ group.title }}
            </h3>
            <div class="mt-4 flex flex-wrap gap-2">
              <button
                v-for="value in group.values"
                :key="value"
                type="button"
                :data-home-taste-chip-kind="group.kind"
                class="rounded-full border border-border/60 bg-background/80 px-3 py-1.5 text-sm text-foreground transition-colors hover:border-primary/35 hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                @click="emit('browseTaste', { kind: group.kind, label: value })"
              >
                {{ value }}
              </button>
              <span
                v-if="group.values.length === 0"
                class="text-sm text-muted-foreground"
              >
                {{ t("home.tasteEmpty") }}
              </span>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>
