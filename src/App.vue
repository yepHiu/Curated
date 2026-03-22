<script setup lang="ts">
import { watch } from "vue"
import { useI18n } from "vue-i18n"
import { RouterView } from "vue-router"
import { syncHtmlLang } from "@/i18n"
import { persistLocale, type SupportedLocale } from "@/lib/locale-storage"

const { locale } = useI18n()

watch(
  locale,
  (l) => {
    persistLocale(l as SupportedLocale)
    syncHtmlLang(l as SupportedLocale)
  },
  { immediate: true },
)
</script>

<template>
  <RouterView />
</template>
