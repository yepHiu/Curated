<script setup lang="ts">
import type { WishlistItem } from "@/domain/wishlist/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardTitle } from "@/components/ui/card"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import { useI18n } from "vue-i18n"
defineProps<{ item: WishlistItem; image?: string }>()
defineEmits<{ open: [id: string] }>()
const { t } = useI18n()
</script>
<template>
  <Card class="group min-w-0 gap-0 overflow-hidden rounded-[1.2rem] border-border/70 bg-card/80 py-0 shadow-md shadow-black/5 transition-[box-shadow,border-color] duration-150 hover:border-primary/25 hover:shadow-lg motion-reduce:transition-none">
    <button type="button" class="flex w-full min-w-0 flex-col rounded-[1.2rem] text-left focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring" @click="$emit('open', item.id)">
      <div class="w-full p-[var(--movie-card-padding)] pb-0">
        <div class="relative aspect-[358/537] w-full overflow-hidden rounded-[0.95rem] border border-border/60 bg-muted/30">
          <MediaStill v-if="image" :src="image" :alt="item.code" />
          <div v-else class="flex size-full items-center justify-center p-4 text-center font-medium text-muted-foreground">{{ item.code }}</div>
          <Badge class="absolute top-0 left-0 m-[var(--movie-card-padding)] max-w-[calc(100%-1.25rem)] truncate rounded-full border-border/40 bg-background/85 px-1.5 text-[10px] text-foreground shadow-sm backdrop-blur-sm">{{ item.code }}</Badge>
        </div>
      </div>
      <CardContent class="flex w-full min-w-0 min-h-[var(--movie-card-body-min-height)] flex-col justify-between gap-[var(--movie-card-body-gap)] p-[var(--movie-card-padding)]">
        <div class="flex min-w-0 flex-col gap-0.5">
          <CardTitle class="truncate text-[13px]">{{ item.metadata.title || t('wishlist.awaiting') }}</CardTitle>
          <CardDescription class="truncate text-[11px]">{{ item.metadata.actors.join(' · ') }}</CardDescription>
        </div>
        <div class="flex min-h-6 min-w-0 items-center gap-1">
          <Badge variant="secondary" class="min-w-0 max-w-full shrink truncate rounded-full px-1.5 text-[10px]">
            {{ t(item.enrichmentState !== 'ready' ? `wishlist.states.${item.enrichmentState}` : `wishlist.filters.${item.status}`) }}
          </Badge>
        </div>
      </CardContent>
    </button>
  </Card>
</template>
