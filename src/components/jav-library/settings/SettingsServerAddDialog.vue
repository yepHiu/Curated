<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Loader2, Plus } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"

defineProps<{ disabled?: boolean }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()
const open = ref(false)
const name = ref("")
const url = ref("")
const saving = ref(false)
const error = ref("")
const canSave = computed(() => Boolean(name.value.trim() && url.value.trim()) && !saving.value)

function setOpen(value: boolean) {
  if (saving.value) return
  open.value = value
  if (value) { name.value = ""; url.value = ""; error.value = "" }
}

async function save() {
  if (!canSave.value) return
  error.value = ""
  const bridge = window.javLibrary
  if (!bridge?.addServer) { error.value = t("settings.serverConnections.restartRequired"); return }
  let address: URL
  try {
    const value = url.value.trim()
    address = new URL(value.includes("://") ? value : `http://${value}`)
    if (!["http:", "https:"].includes(address.protocol) || address.username || address.password || address.pathname !== "/" || address.search || address.hash) throw new Error()
  } catch { error.value = t("settings.serverConnections.invalidAddress"); return }
  saving.value = true
  try {
    const saved = await bridge.getServerConnections?.()
    if (saved?.servers.some(server => server.url === address.origin)) {
      error.value = t("settings.serverConnections.duplicateAddress")
      return
    }
    await bridge.addServer({ name: name.value.trim(), url: address.origin })
    open.value = false
    emit("saved")
  } catch (cause) {
    error.value = t(cause instanceof Error && cause.message.includes("No handler registered")
      ? "settings.serverConnections.restartRequired" : "settings.serverConnections.saveError")
  } finally { saving.value = false }
}
</script>

<template>
  <Dialog :open="open" @update:open="setOpen">
    <DialogTrigger as-child>
      <Button type="button" class="rounded-full" :disabled="disabled">
        <Plus class="size-4" aria-hidden="true" />
        {{ t('settings.serverConnections.add') }}
      </Button>
    </DialogTrigger>
    <DialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md" :show-close-button="!saving" @escape-key-down="saving && $event.preventDefault()" @interact-outside="saving && $event.preventDefault()">
      <DialogHeader>
        <DialogTitle>{{ t('settings.serverConnections.add') }}</DialogTitle>
        <DialogDescription class="sr-only">{{ t('settings.serverConnections.addDescription') }}</DialogDescription>
      </DialogHeader>
      <form class="flex min-w-0 flex-col gap-4" :aria-busy="saving" @submit.prevent="save">
        <div class="flex flex-col gap-2">
          <label for="new-server-name" class="text-sm font-medium">{{ t('settings.serverConnections.nameLabel') }}</label>
          <Input id="new-server-name" v-model="name" :maxlength="80" required autocomplete="off" class="rounded-xl border-input bg-muted/30" :placeholder="t('settings.serverConnections.namePlaceholder')" :disabled="saving" />
        </div>
        <div class="flex flex-col gap-2">
          <label for="new-server-url" class="text-sm font-medium">{{ t('settings.serverConnections.addressLabel') }}</label>
          <Input id="new-server-url" v-model="url" :maxlength="2048" required autocomplete="off" :spellcheck="false" autocapitalize="none" class="rounded-xl border-input bg-muted/30" placeholder="http://192.168.1.10:8081" :disabled="saving" :aria-invalid="!!error" :aria-describedby="error ? 'new-server-error' : undefined" />
        </div>
        <p v-if="error" id="new-server-error" role="alert" class="text-sm text-destructive">{{ error }}</p>
        <DialogFooter class="gap-2 pt-2">
          <DialogClose as-child>
            <Button type="button" variant="outline" class="min-h-11 rounded-full sm:min-h-9" :disabled="saving">{{ t('common.cancel') }}</Button>
          </DialogClose>
          <Button type="submit" class="min-h-11 rounded-full sm:min-h-9" :disabled="!canSave" data-save-server>
            <Loader2 v-if="saving" class="size-4 motion-safe:animate-spin" aria-hidden="true" />
            {{ t(saving ? 'common.saving' : 'settings.serverConnections.save') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
