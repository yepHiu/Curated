<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { LayoutDashboard } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

type DashboardStat = {
  labelKey: string
  value: string | number
  detailKey?: string
}

defineProps<{
  dashboardStats: readonly DashboardStat[]
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="space-y-3 pb-2">
          <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <LayoutDashboard class="size-[1.15rem]" />
            </span>
            {{ t("settings.navOverview") }}
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-2">
          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <Card
              v-for="stat in dashboardStats"
              :key="stat.labelKey"
              class="rounded-lg border border-border/50 bg-muted/5 shadow-none"
            >
              <CardHeader class="gap-3">
                <CardDescription>{{ t(stat.labelKey) }}</CardDescription>
                <CardTitle class="text-2xl">{{ stat.value }}</CardTitle>
              </CardHeader>
              <CardContent v-if="stat.detailKey">
                <p class="text-sm text-muted-foreground">{{ t(stat.detailKey) }}</p>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
