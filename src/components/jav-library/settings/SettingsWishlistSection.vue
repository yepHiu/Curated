<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useLibraryService } from "@/services/library-service"
import type { WishlistToken } from "@/domain/wishlist/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Field, FieldLabel } from "@/components/ui/field"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
const { t } = useI18n(), service = useLibraryService().wishlist
const items = ref<WishlistToken[]>([]), secret = ref(""), busy = ref(false), error = ref(false)
/** 从本机后端读取可撤销凭证清单。 */
async function load() { if (!service.integrationsAvailable) return; try { items.value = await service.tokens(); error.value = false } catch { error.value = true } }
/** 创建一次展示的凭证，离开页面即不再保留明文。 */
async function create() { busy.value = true; try { const token = await service.createToken("Curated Plugin"); secret.value = token.token ?? ""; await load() } catch { error.value = true } finally { busy.value = false } }
/** 显式撤销，插件必须重新配置凭证。 */
async function revoke(id: string) { busy.value = true; try { await service.revokeToken(id); secret.value = ""; await load() } catch { error.value = true } finally { busy.value = false } }
onMounted(load)
</script>
<template>
  <Card><CardHeader><CardTitle>{{ t('wishlist.connect') }}</CardTitle><CardDescription>{{ t('wishlist.tokenHint') }}</CardDescription></CardHeader><CardContent class="flex flex-col gap-4">
    <p v-if="!service.integrationsAvailable" class="text-sm text-muted-foreground">{{ t('wishlist.mockHint') }}</p>
    <template v-else><p v-if="error" role="alert">{{ t('wishlist.tokenError') }}</p><Button class="self-start" :disabled="busy" @click="create">{{ t('wishlist.createToken') }}</Button><Field v-if="secret"><FieldLabel for="wishlist-token">{{ t('wishlist.tokenOnce') }}</FieldLabel><Input id="wishlist-token" :model-value="secret" readonly autocomplete="off" @focus="($event.target as HTMLInputElement).select()" /></Field><div v-for="token in items" :key="token.id" class="flex items-center justify-between gap-3"><span class="truncate text-sm">{{ token.name }} · {{ token.createdAt.slice(0, 10) }}</span><Button variant="outline" :disabled="busy" @click="revoke(token.id)">{{ t('wishlist.revoke') }}</Button></div></template>
  </CardContent></Card>
</template>
