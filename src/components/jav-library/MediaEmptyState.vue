<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Inbox, SearchX } from "lucide-vue-next"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"

withDefaults(defineProps<{
  filtered?: boolean
  title?: string
  description?: string
}>(), { filtered: false })

const { t } = useI18n()
</script>

<template>
  <Empty
    data-media-empty-state
    :data-empty-kind="filtered ? 'filtered' : 'collection'"
    class="min-h-64 w-full flex-none gap-5 rounded-2xl border border-dashed border-border/60 bg-card/40 px-6 py-12 md:py-16"
  >
    <EmptyHeader>
      <EmptyMedia variant="icon" class="size-12 rounded-2xl bg-muted/60 text-muted-foreground">
        <SearchX v-if="filtered" class="size-6" aria-hidden="true" />
        <Inbox v-else class="size-6" aria-hidden="true" />
      </EmptyMedia>
      <EmptyTitle>{{ filtered ? t('mediaEmpty.noResultsTitle') : title ?? t('mediaEmpty.title') }}</EmptyTitle>
      <EmptyDescription>{{ filtered ? t('mediaEmpty.noResultsDescription') : description ?? t('mediaEmpty.description') }}</EmptyDescription>
    </EmptyHeader>
    <EmptyContent v-if="$slots.default">
      <slot />
    </EmptyContent>
  </Empty>
</template>
