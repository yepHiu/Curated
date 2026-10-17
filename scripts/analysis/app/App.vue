<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from "vue";
import { Download, RefreshCw, Orbit, AlertCircle } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import AtlasExplorer from "./AtlasExplorer.vue";
import type { AtlasSnapshot, ScanStatus } from "./types";

const snapshot = shallowRef<AtlasSnapshot | null>(null);
const status = ref<ScanStatus | null>(null);
const connectionError = ref("");
const starting = ref(false);
const exporting = ref(false);
const busy = computed(() => starting.value || !!status.value?.scanning);
const error = computed(
  () => connectionError.value || status.value?.error || "",
);
const phase = computed(() => {
  if (connectionError.value) return "服务连接中断，正在重试";
  if (!status.value) return "连接本地分析服务";
  const s = status.value;
  return (
    s.message +
    (s.scanning && s.total ? ` · ${s.completed ?? 0} / ${s.total} 个文件` : "")
  );
});
let revision = "";
let timer: ReturnType<typeof setTimeout> | undefined;
let syncing = false;
const controller = new AbortController();

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(url, { ...init, signal: controller.signal });
  if (!response.ok) {
    const text = await response.text();
    try {
      throw new Error(
        (JSON.parse(text) as { error?: string }).error ||
          `HTTP ${response.status}`,
      );
    } catch (error) {
      if (error instanceof SyntaxError)
        throw new Error(`HTTP ${response.status}`);
      throw error;
    }
  }
  return response.json() as Promise<T>;
}
async function sync() {
  if (syncing || controller.signal.aborted) return;
  syncing = true;
  clearTimeout(timer);
  try {
    status.value = await request<ScanStatus>("/api/atlas/status");
    if (
      status.value.hasSnapshot &&
      `${status.value.sessionId}:${status.value.revision}` !== revision
    ) {
      const result = await request<{
        sessionId: string;
        revision: number;
        snapshot: AtlasSnapshot;
      }>("/api/atlas/snapshot");
      snapshot.value = result.snapshot;
      revision = `${result.sessionId}:${result.revision}`;
    }
    connectionError.value = "";
  } catch (error) {
    if (!controller.signal.aborted)
      connectionError.value =
        error instanceof Error ? error.message : "连接失败";
  } finally {
    syncing = false;
    if (!controller.signal.aborted)
      timer = setTimeout(
        () => void sync(),
        status.value?.scanning ? 800 : 4000,
      );
  }
}
async function scan() {
  if (busy.value) return;
  starting.value = true;
  try {
    status.value = await request<ScanStatus>("/api/atlas/scan", {
      method: "POST",
      headers: { "X-Code-Atlas": "1" },
    });
    connectionError.value = "";
    await sync();
  } catch (error) {
    if (!controller.signal.aborted)
      connectionError.value =
        error instanceof Error ? error.message : "无法开始扫描";
  } finally {
    starting.value = false;
  }
}
async function download() {
  exporting.value = true;
  try {
    const response = await fetch("/api/atlas/export", {
      signal: controller.signal,
    });
    if (!response.ok) throw new Error(`导出失败：HTTP ${response.status}`);
    const url = URL.createObjectURL(await response.blob());
    const a = document.createElement("a");
    a.href = url;
    a.download = "curated-code-atlas.html";
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch (error) {
    if (!controller.signal.aborted)
      connectionError.value =
        error instanceof Error ? error.message : "导出失败";
  } finally {
    exporting.value = false;
  }
}
onMounted(() => void sync());
onBeforeUnmount(() => {
  controller.abort();
  clearTimeout(timer);
});
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="flex items-center gap-3">
        <Orbit class="size-7 text-primary" aria-hidden="true" />
        <div>
          <p class="text-lg font-semibold tracking-tight">
            Curated
            <span class="font-normal text-muted-foreground">/ Code Atlas</span>
          </p>
          <p class="text-xs text-muted-foreground">内部分析工具 · 当前工作区</p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <Button
          variant="outline"
          size="lg"
          :disabled="!snapshot || exporting"
          @click="download"
          ><Download data-icon="inline-start" />{{
            exporting ? "正在导出…" : "导出离线 HTML"
          }}</Button
        >
        <Button size="lg" :disabled="busy" @click="scan"
          ><RefreshCw
            data-icon="inline-start"
            :class="{ 'animate-spin motion-reduce:animate-none': busy }"
          />{{ busy ? "扫描中…" : "重新扫描" }}</Button
        >
      </div>
    </header>
    <div class="status-strip" role="status" aria-live="polite">
      <span class="flex items-center gap-2"
        ><span class="size-1.5 rounded-full bg-primary" aria-hidden="true" />{{
          phase
        }}</span
      >
      <span v-if="snapshot" class="text-xs text-muted-foreground">{{
        busy || error ? "继续展示上次成功结果" : "扫描完成后自动更新视图"
      }}</span>
    </div>
    <Alert v-if="error" variant="destructive" class="my-4">
      <AlertCircle aria-hidden="true" /><AlertTitle
        >暂时无法更新分析</AlertTitle
      >
      <AlertDescription
        ><p class="break-words">{{ error }}</p>
        <p>
          请确认本地服务与 Go 环境可用，再点击“重新扫描”。已有结果仍可浏览。
        </p></AlertDescription
      >
    </Alert>
    <div v-if="!snapshot" class="loading-panel" :aria-busy="busy">
      <h1 class="text-2xl font-semibold">
        {{ error ? "尚未生成分析结果" : "正在绘制项目的代码地图" }}
      </h1>
      <p class="mt-3 text-sm text-muted-foreground">
        {{ phase }}。首次扫描完成后会显示目录、函数与依赖。
      </p>
      <template v-if="!error"
        ><div class="mt-7 grid grid-cols-2 gap-4 md:grid-cols-4">
          <Skeleton v-for="n in 4" :key="n" class="h-28" />
        </div>
        <Skeleton class="mt-5 h-80"
      /></template>
    </div>
  </div>
  <AtlasExplorer v-if="snapshot" :snapshot="snapshot" />
</template>
