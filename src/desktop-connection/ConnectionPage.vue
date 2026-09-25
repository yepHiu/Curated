<script setup lang="ts">
import { onMounted, ref } from "vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface Connection { url: string; name: string; serverId: string }
interface ConnectionAPI {
  list(): Promise<{ connections: Connection[]; lastUrl?: string; activeUrl?: string; desktopVersion: string }>
  connect(url: string): Promise<{ ok: boolean; error?: string }>
  cancel(): Promise<void>
  forget(url: string): Promise<void>
}
const api = (window as unknown as { curatedConnection: ConnectionAPI }).curatedConnection
const address = ref("")
const connections = ref<Connection[]>([])
const busy = ref(false)
const error = ref("")
const version = ref("")
async function refresh() {
  const state = await api.list()
  connections.value = state.connections
  version.value = state.desktopVersion
  return state
}
async function connect(url = address.value) {
  busy.value = true; error.value = ""
  try {
    const result = await api.connect(url)
    if (!result.ok) error.value = result.error ?? "连接失败"
    else { address.value = url; await refresh() }
  } catch (reason) { error.value = String(reason) }
  finally { busy.value = false }
}
async function forget(url: string) {
  try { await api.forget(url); await refresh() } catch (reason) { error.value = String(reason) }
}
onMounted(async () => {
  try {
    const state = await refresh()
    address.value = state.lastUrl ?? ""
    if (state.lastUrl && !state.activeUrl) await connect(state.lastUrl)
  } catch (reason) { error.value = String(reason) }
})
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-xl flex-col gap-8 p-8 text-foreground">
    <header class="space-y-2 pt-6">
      <p class="font-curated text-2xl font-semibold text-primary">Curated Desktop</p>
      <h1 class="text-xl font-semibold">连接服务器</h1>
    </header>
    <form class="flex flex-col gap-4 rounded-xl border border-border bg-card p-6" @submit.prevent="connect()">
      <label for="server-address" class="text-sm font-medium">服务器地址</label>
      <Input id="server-address" v-model="address" placeholder="http://192.168.1.20:8081" autocomplete="url" :disabled="busy" required />
      <p class="text-sm text-muted-foreground">输入 IP、域名与端口，或完整 HTTP / HTTPS 地址。</p>
      <p v-if="error" role="alert" class="break-words text-sm text-destructive">{{ error }}</p>
      <div class="flex gap-3">
        <Button type="submit" :disabled="busy || !address.trim()">{{ busy ? '正在连接…' : '连接' }}</Button>
        <Button v-if="busy" type="button" variant="outline" @click="api.cancel()">取消</Button>
      </div>
    </form>
    <section v-if="connections.length" aria-labelledby="recent-title" class="space-y-3">
      <h2 id="recent-title" class="text-sm font-semibold">最近连接</h2>
      <ul class="divide-y divide-border rounded-xl border border-border">
        <li v-for="item in connections" :key="item.url" class="flex items-center gap-3 p-4">
          <div class="min-w-0 flex-1"><p class="truncate font-medium">{{ item.name }}</p><p class="truncate text-sm text-muted-foreground">{{ item.url }}</p></div>
          <Button variant="outline" :disabled="busy" @click="connect(item.url)">连接</Button>
          <Button variant="ghost" :disabled="busy" :aria-label="`忘记 ${item.name}`" @click="forget(item.url)">忘记</Button>
        </li>
      </ul>
    </section>
    <p class="mt-auto text-xs text-muted-foreground">Desktop {{ version }}</p>
  </main>
</template>
