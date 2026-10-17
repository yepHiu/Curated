<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { BookMetadata } from "@/api/types"
import { Badge } from "@/components/ui/badge"
defineProps<{ metadata?: BookMetadata; evidence?: boolean; browseable?: boolean }>()
const emit = defineEmits<{ browseByTag: [tag: string] }>()
const { t } = useI18n()
</script>

<template>
  <dl v-if="metadata?.cleanTitle" class="flex min-w-0 flex-col gap-2 text-sm">
    <div class="flex flex-wrap gap-x-3"><dt class="text-muted-foreground">{{ t('bookAI.cleanTitle') }}</dt><dd class="break-words">{{ metadata.cleanTitle }}</dd></div>
    <template v-for="role in (['authors', 'models'] as const)" :key="role">
      <div v-if="metadata[role]?.length" class="flex flex-col gap-1">
        <dt class="text-muted-foreground">{{ t(`bookAI.${role}`) }}</dt>
        <dd v-for="credit in metadata[role]" :key="credit.name" class="break-words">
          <Badge v-if="browseable" variant="secondary" as-child class="max-w-full rounded-full border border-border/60 bg-secondary/70">
            <button
              type="button"
              class="cursor-pointer whitespace-normal break-words text-left hover:bg-secondary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :data-book-credit-tag="credit.name"
              :aria-label="t('detailPanel.ariaSearchInLibrary', { tag: credit.name })"
              @click="emit('browseByTag', credit.name)"
            >{{ credit.name }}</button>
          </Badge>
          <template v-else>{{ credit.name }}</template>
          <span v-if="evidence" class="ml-2 text-muted-foreground">「{{ credit.evidence }}」</span>
        </dd>
      </div>
    </template>
    <div v-if="metadata.uncertainty" class="flex flex-col gap-1"><dt class="text-muted-foreground">{{ t('bookAI.uncertainty') }}</dt><dd class="break-words">{{ metadata.uncertainty }}</dd></div>
  </dl>
</template>
