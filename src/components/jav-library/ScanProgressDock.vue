<script setup lang="ts">
import { computed } from "vue"
import { X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"

const { activeTask, pollError, dismiss } = useScanTaskTracker()

const visible = computed(() => activeTask.value != null || pollError.value != null)

function metaNum(m: Record<string, unknown> | undefined, key: string): string {
  if (!m) return "—"
  const v = m[key]
  if (typeof v === "number" && Number.isFinite(v)) {
    return String(Math.trunc(v))
  }
  if (typeof v === "string" && v !== "") {
    const n = Number(v)
    return Number.isFinite(n) ? String(Math.trunc(n)) : "—"
  }
  return "—"
}

const dockTitle = computed(() => {
  if (pollError.value && !activeTask.value) return "扫描状态"
  const t = activeTask.value
  if (!t) return ""
  if (t.status === "completed") return "扫描完成"
  if (t.status === "failed" || t.status === "partial_failed") return "扫描结束"
  return "正在扫描库"
})

const progressModel = computed(() => {
  const t = activeTask.value
  if (!t) return 0
  return Math.min(100, Math.max(0, t.progress))
})

const detailLine = computed(() => {
  const t = activeTask.value
  if (!t?.message) return ""
  return t.message
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="animate-in fade-in-0 slide-in-from-bottom-4 fixed right-4 bottom-4 z-50 w-[min(100%-2rem,22rem)] duration-300"
    >
      <Card class="border-border/80 bg-card/95 shadow-xl backdrop-blur-sm">
        <CardHeader class="flex flex-row items-start justify-between gap-2 space-y-0 pb-2">
          <CardTitle class="text-base font-semibold">{{ dockTitle }}</CardTitle>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="shrink-0 rounded-xl"
            aria-label="关闭"
            @click="dismiss"
          >
            <X class="size-4" />
          </Button>
        </CardHeader>
        <CardContent class="flex flex-col gap-3">
          <template v-if="activeTask">
            <Progress :model-value="progressModel" />
            <p v-if="detailLine" class="text-xs text-muted-foreground leading-snug">
              {{ detailLine }}
            </p>
            <div class="grid grid-cols-2 gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span>已处理</span>
              <span class="text-right font-mono text-foreground">
                {{ metaNum(activeTask.metadata, "scanProcessed") }} /
                {{ metaNum(activeTask.metadata, "scanTotal") }}
              </span>
              <span>新入库</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(activeTask.metadata, "scanImported")
              }}</span>
              <span>更新</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(activeTask.metadata, "scanUpdated")
              }}</span>
              <span>跳过</span>
              <span class="text-right font-mono text-foreground">{{
                metaNum(activeTask.metadata, "scanSkipped")
              }}</span>
            </div>
            <p
              v-if="activeTask.status === 'failed' && activeTask.errorMessage"
              class="text-xs text-destructive"
            >
              {{ activeTask.errorMessage }}
            </p>
          </template>
          <p v-else-if="pollError" class="text-sm text-destructive">
            {{ pollError }}
          </p>
        </CardContent>
      </Card>
    </div>
  </Teleport>
</template>
