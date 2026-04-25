<script setup lang="ts">
import { Skeleton } from "@/components/ui/skeleton"

const posterSkeletons = Array.from({ length: 6 }, (_, index) => `poster-${index}`)
const continueSkeletons = Array.from({ length: 4 }, (_, index) => `continue-${index}`)
const tasteSkeletons = Array.from({ length: 3 }, (_, index) => `taste-${index}`)
</script>

<template>
  <div
    data-homepage-portal-skeleton
    class="h-full min-h-0 overflow-y-auto bg-background text-foreground"
  >
    <section
      data-homepage-skeleton-hero
      class="relative isolate bg-[radial-gradient(circle_at_top,_hsl(var(--primary)/0.24),_transparent_52%),linear-gradient(180deg,hsl(var(--background)),hsl(var(--background)))]"
    >
      <div class="flex w-full justify-center px-0 py-4 lg:py-5">
        <div class="w-full">
          <div class="w-full overflow-visible">
            <div
              class="relative h-[clamp(22rem,44vw,40rem)] w-full overflow-hidden bg-[radial-gradient(circle_at_top,_hsl(var(--primary)/0.24),_transparent_68%),linear-gradient(180deg,rgba(8,10,14,0.92),rgba(8,10,14,0.98))] sm:h-[clamp(25rem,46vw,44rem)]"
            >
              <div class="flex h-full items-center justify-center px-4 sm:px-6 lg:px-8">
                <Skeleton class="h-[82%] w-[82%] rounded-[1.6rem] sm:w-[76%] lg:w-[68%] xl:w-[64%]" />
              </div>
            </div>
          </div>

          <div
            class="relative mx-auto mt-3 w-[calc(100%-2rem)] max-w-[54rem] rounded-[1.4rem] bg-background/72 px-5 py-4 backdrop-blur-md sm:w-[calc(100%-2.5rem)] sm:px-7 lg:w-[calc(100%-3rem)] lg:px-10 xl:w-[calc(100%-4rem)] xl:px-14"
          >
            <div class="grid grid-cols-8 gap-2">
              <Skeleton
                v-for="bar in 8"
                :key="bar"
                class="h-1.5 rounded-full"
              />
            </div>
          </div>
        </div>
      </div>
    </section>

    <div
      class="mx-auto flex w-full max-w-[1680px] flex-col gap-8 px-4 py-6 sm:px-5 lg:gap-10 lg:px-6 lg:py-8 xl:px-8"
    >
      <section
        v-for="sectionId in ['recent', 'recommend']"
        :key="sectionId"
        :data-homepage-skeleton-section="sectionId"
        class="space-y-4"
      >
        <div class="space-y-2">
          <Skeleton class="h-6 w-44" />
          <Skeleton class="h-4 w-full max-w-xl" />
        </div>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          <article
            v-for="posterId in posterSkeletons"
            :key="`${sectionId}-${posterId}`"
            class="space-y-3"
          >
            <Skeleton class="aspect-[4/5.6] w-full rounded-xl" />
            <div class="space-y-2">
              <Skeleton class="h-4 w-2/3" />
              <Skeleton class="h-3 w-1/2" />
            </div>
          </article>
        </div>
      </section>

      <section
        data-homepage-skeleton-section="continue"
        class="space-y-4"
      >
        <div class="space-y-2">
          <Skeleton class="h-6 w-48" />
          <Skeleton class="h-4 w-full max-w-xl" />
        </div>

        <div class="grid gap-3 lg:grid-cols-2">
          <article
            v-for="itemId in continueSkeletons"
            :key="itemId"
            class="grid min-h-32 grid-cols-[5rem_1fr] gap-4 rounded-[1.25rem] border border-border/50 bg-card/60 p-4"
          >
            <Skeleton class="h-full w-full rounded-lg" />
            <div class="space-y-3">
              <Skeleton class="h-5 w-2/3" />
              <Skeleton class="h-4 w-1/2" />
              <Skeleton class="h-2.5 w-full rounded-full" />
            </div>
          </article>
        </div>
      </section>

      <section
        data-homepage-skeleton-section="taste"
        class="space-y-4"
      >
        <div class="space-y-2">
          <Skeleton class="h-6 w-40" />
          <Skeleton class="h-4 w-full max-w-xl" />
        </div>

        <div class="grid gap-3 lg:grid-cols-3">
          <article
            v-for="itemId in tasteSkeletons"
            :key="itemId"
            class="rounded-[1.6rem] border border-border/60 bg-card/65 p-4"
          >
            <Skeleton class="h-4 w-24" />
            <div class="mt-4 flex flex-wrap gap-2">
              <Skeleton
                v-for="chip in 4"
                :key="`${itemId}-${chip}`"
                class="h-8 w-20 rounded-full"
              />
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>
