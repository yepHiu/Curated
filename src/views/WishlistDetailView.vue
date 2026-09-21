<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { useLibraryService } from "@/services/library-service"
import type { WishlistItem, WishlistPatch } from "@/domain/wishlist/types"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
const { t } = useI18n(), route = useRoute(), router = useRouter(), service = useLibraryService().wishlist
const item = ref<WishlistItem>(), error = ref(false), busy = ref(false), note = ref(""), code = ref(""), removeOpen = ref(false)
const source = computed(() => { /* 仅允许安全的刮削来源链接。 */ const value = item.value?.metadata.homepage ?? ""; return /^https?:\/\//i.test(value) ? value : "" })
const back = computed(() => { /* 返回目标限定在愿望单内部。 */ const value = String(route.query.back ?? ""); return /^\/wishlist(?:\?|$)/.test(value) ? value : "/wishlist" })
let timer: ReturnType<typeof setTimeout> | undefined, alive = true, generation = 0
/** 按 ID 获取独立详情，仅首次加载时填充可编辑草稿。 */
async function load(initial = false) { const current = ++generation; try { const result = await service.get(String(route.params.id)); if (!alive || current !== generation) return; item.value = result; error.value = false; if (initial) { note.value = result.note; code.value = result.code } } catch { if (alive && current === generation) error.value = true } }
/** 只在前台更新正在处理的详情。 */
async function poll() { if (!document.hidden && item.value && ["queued", "running"].includes(item.value.enrichmentState)) await load(); if (alive) timer = setTimeout(poll, 6000) }
/** 按版本保存用户操作，冲突后重新读取当前服务器状态。 */
async function patch(change: Omit<WishlistPatch, "version">) { if (!item.value || busy.value) return; busy.value = true; try { item.value = await service.patch(item.value.id, { version: item.value.version, ...change }); error.value = false } catch { await load(); error.value = true } finally { busy.value = false } }
/** 重试后台补全但保留现有图片。 */
async function retry() { if (!item.value) { await load(true); return } busy.value = true; try { await service.refresh(item.value.id); await load() } catch { error.value = true } finally { busy.value = false } }
/** 确认后删除愿望并回到原列表。 */
async function remove() { if (!item.value) return; busy.value = true; try { await service.remove(item.value.id); await router.replace(back.value) } catch { error.value = true } finally { busy.value = false } }
watch(() => route.params.id, () => { /* 路由切换丢弃旧详情。 */ item.value = undefined; void load(true) }, { immediate: true })
timer = setTimeout(poll, 6000)
onBeforeUnmount(() => { /* 停止轮询并使迟到请求无效。 */ alive = false; generation++; clearTimeout(timer) })
</script>
<template>
  <section class="min-h-0 flex-1 overflow-y-auto">
    <div class="mb-5 flex flex-wrap items-center gap-2"><Button variant="ghost" @click="router.push(back)">{{ t('wishlist.back') }}</Button><Badge v-if="item" variant="secondary">{{ t(`wishlist.filters.${item.status}`) }}</Badge></div>
    <div v-if="error" role="alert" class="mb-4 flex items-center gap-3"><span>{{ t('wishlist.operationFailed') }}</span><Button variant="outline" :disabled="busy" @click="load(true)">{{ t('wishlist.reload') }}</Button></div>
    <div v-if="item" class="grid gap-6 lg:grid-cols-[minmax(220px,340px)_1fr]">
      <div class="relative aspect-[2/3] overflow-hidden rounded-lg bg-muted"><MediaStill v-if="item.assets.length" :src="service.assetUrl(item.assets.find(a => a.role === 'cover')?.url ?? item.assets[0]!.url)" :alt="item.code" fit="contain" loading="eager" /><div v-else class="flex size-full items-center justify-center text-muted-foreground">{{ item.code }}</div></div>
      <div class="flex min-w-0 flex-col gap-4">
        <div><p class="mb-2 text-sm text-muted-foreground">{{ item.code }}</p><h1 class="font-curated text-2xl font-semibold">{{ item.metadata.title || item.code }}</h1></div>
        <p v-if="item.enrichmentState !== 'ready'" role="status" class="text-sm text-muted-foreground">{{ t(`wishlist.states.${item.enrichmentState}`) }}<span v-if="item.error"> · {{ t(`wishlist.errors.${item.error}`) }}</span></p>
        <div class="flex flex-wrap gap-2"><Button v-for="movieId in item.movieIds" :key="movieId" @click="router.push({ name: 'detail', params: { id: movieId } })">{{ t('wishlist.openMovie') }}</Button><Button variant="outline" :disabled="busy" @click="patch({ completed: !item.completed })">{{ t(item.completed ? 'wishlist.restore' : 'wishlist.complete') }}</Button><Button variant="outline" :disabled="busy || ['queued', 'running'].includes(item.enrichmentState)" @click="retry">{{ t('wishlist.retry') }}</Button></div>
        <p v-if="item.metadata.actors.length">{{ item.metadata.actors.join(' / ') }}</p>
        <p class="text-sm text-muted-foreground">{{ [item.metadata.studio, item.metadata.releaseDate, item.metadata.runtimeMinutes ? `${item.metadata.runtimeMinutes} min` : ''].filter(Boolean).join(' · ') }}</p>
        <p v-if="item.metadata.summary" class="whitespace-pre-wrap text-sm leading-relaxed">{{ item.metadata.summary }}</p>
        <div v-if="item.metadata.tags.length" class="flex flex-wrap gap-2"><Badge v-for="tag in item.metadata.tags" :key="tag" variant="secondary">{{ tag }}</Badge></div>
        <a v-if="source" :href="source" target="_blank" rel="noopener noreferrer" class="text-sm text-primary underline">{{ t('wishlist.source') }} · {{ item.metadata.provider }}</a>
        <FieldGroup><Field><FieldLabel for="wishlist-code">{{ t('wishlist.code') }}</FieldLabel><div class="flex gap-2"><Input id="wishlist-code" v-model="code" maxlength="80" /><Button variant="outline" :disabled="busy || code === item.code" @click="patch({ code })">{{ t('wishlist.save') }}</Button></div></Field><Field><FieldLabel for="wishlist-note">{{ t('wishlist.note') }}</FieldLabel><div class="flex gap-2"><Input id="wishlist-note" v-model="note" maxlength="10000" /><Button variant="outline" :disabled="busy || note === item.note" @click="patch({ note })">{{ t('wishlist.save') }}</Button></div></Field></FieldGroup>
        <Button variant="ghost" class="self-start" :disabled="busy" @click="removeOpen = true">{{ t('wishlist.remove') }}</Button>
      </div>
    </div>
    <div v-if="item" class="mt-6 grid grid-cols-2 gap-3 md:grid-cols-3"><a v-for="asset in item.assets.filter(a => a.role === 'preview_image')" :key="asset.id" :href="service.assetUrl(asset.url)" target="_blank" rel="noopener noreferrer" class="relative aspect-video overflow-hidden rounded-lg bg-muted"><MediaStill :src="service.assetUrl(asset.thumbnailUrl)" :alt="item.code" fit="contain" /></a></div>
    <Dialog v-model:open="removeOpen"><DialogContent><DialogHeader><DialogTitle>{{ t('wishlist.remove') }}</DialogTitle><DialogDescription>{{ t('wishlist.removeHint') }}</DialogDescription></DialogHeader><DialogFooter><Button variant="outline" @click="removeOpen = false">{{ t('wishlist.cancel') }}</Button><Button @click="remove">{{ t('wishlist.remove') }}</Button></DialogFooter></DialogContent></Dialog>
  </section>
</template>
