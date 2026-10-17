<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { BookMetadata } from "@/api/types"
import { bookCreditRoles } from "@/lib/book-metadata"
defineProps<{ metadata?: BookMetadata; evidence?: boolean }>()
const { t } = useI18n()
</script>

<template>
  <dl v-if="metadata?.cleanTitle" class="flex min-w-0 flex-col gap-2 text-sm">
    <div class="flex flex-wrap gap-x-3"><dt class="text-muted-foreground">{{ t('bookAI.cleanTitle') }}</dt><dd class="break-words">{{ metadata.cleanTitle }}</dd></div>
    <template v-for="role in bookCreditRoles" :key="role">
      <div v-if="metadata[role]?.length" class="flex flex-col gap-1">
        <dt class="text-muted-foreground">{{ t(`bookAI.${role}`) }}</dt>
        <dd v-for="credit in metadata[role]" :key="credit.name" class="break-words">
          {{ credit.name }}<span v-if="evidence" class="ml-2 text-muted-foreground">「{{ credit.evidence }}」</span>
        </dd>
      </div>
    </template>
    <div v-if="metadata.uncertainty" class="flex flex-col gap-1"><dt class="text-muted-foreground">{{ t('bookAI.uncertainty') }}</dt><dd class="break-words">{{ metadata.uncertainty }}</dd></div>
  </dl>
</template>
