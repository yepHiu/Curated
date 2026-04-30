<script setup lang="ts">
import { onErrorCaptured, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { RouterView } from "vue-router"
import { useTheme } from "@/composables/use-theme"
import { syncHtmlLang } from "@/i18n"
import { persistLocale, type SupportedLocale } from "@/lib/locale-storage"

useTheme()

const { locale, t } = useI18n()
const appFault = ref(false)

watch(
  locale,
  (l) => {
    persistLocale(l as SupportedLocale)
    syncHtmlLang(l as SupportedLocale)
  },
  { immediate: true },
)

onErrorCaptured((err, _instance, info) => {
  console.error("[App] captured routed subtree error", err, info)
  appFault.value = true
  return false
})

function reloadApp() {
  window.location.reload()
}
</script>

<template>
  <main
    v-if="appFault"
    data-app-fault
    class="flex min-h-screen items-center justify-center bg-background px-6 py-10 text-foreground"
  >
    <section
      role="alert"
      class="w-full max-w-md rounded-lg border border-border bg-card p-6 text-card-foreground shadow-sm"
    >
      <p class="text-sm font-medium text-muted-foreground">Curated</p>
      <h1 class="mt-2 text-xl font-semibold leading-tight">
        {{ t("app.faultTitle") }}
      </h1>
      <p class="mt-3 text-sm leading-6 text-muted-foreground">
        {{ t("app.faultDescription") }}
      </p>
      <button
        type="button"
        class="mt-6 inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        @click="reloadApp"
      >
        {{ t("app.reload") }}
      </button>
    </section>
  </main>
  <RouterView v-else />
</template>
