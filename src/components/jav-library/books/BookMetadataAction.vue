<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { useI18n } from "vue-i18n"
import { Sparkles, LoaderCircle } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useBookOrganizationService } from "@/services/book-organization-service"
import { pushAppToast } from "@/composables/use-app-toast"
import { useServerLocalAccess } from "@/composables/use-server-local-access"
import { taskRoute } from "@/lib/task-center"

const props = defineProps<{ kind: "comic" | "photo"; books: readonly { id: string; title: string; sourceFileName: string }[] }>()
const { t, locale } = useI18n(), { writeEnabled } = useExperimentalAgent()
const { isServerLocal } = useServerLocalAccess()
const canSubmit = computed(() => writeEnabled.value && isServerLocal.value)
const open = ref(false), busy = ref(false), error = ref(""), submitted = ref("")
const selection = ref<{ id: string; title: string; sourceFileName: string }[]>([])
let requestId = "", targetLocale = ""
const invalid = computed(() => selection.value.length === 0 || selection.value.length > 600)
watch(open, value => {
  if (value) { selection.value = props.books.map(book => ({ id: book.id, title: book.title, sourceFileName: book.sourceFileName })); requestId = crypto.randomUUID(); targetLocale = locale.value; error.value = "" }
})
async function start() {
  if (busy.value || invalid.value || !canSubmit.value) return
  busy.value = true; error.value = ""
  try {
    const job = await useBookOrganizationService().start({ kind: props.kind, bookIds: selection.value.map(b => b.id), locale: targetLocale, requestId })
    submitted.value = `book-organize:${job.id}`; open.value = false
    window.dispatchEvent(new Event("curated:task-center-updated"))
    pushAppToast(t("organizationConfirm.submitted"), { variant: "success" })
  } catch { error.value = t("bookAI.submitFailed") }
  finally { busy.value = false }
}
</script>

<template>
  <Button v-if="canSubmit" variant="outline" class="min-h-11 shrink-0 rounded-full sm:min-h-8" :disabled="!books.length" @click="open = true"><Sparkles data-icon="inline-start" />{{ t('bookAI.action') }}</Button>
  <Button v-if="submitted" as-child variant="link" size="sm"><RouterLink :to="taskRoute(submitted)">{{ t('bookAI.viewTask') }}</RouterLink></Button>
  <Dialog :open="open" @update:open="value => { if (!busy) open = value }">
    <DialogContent class="flex max-h-[85dvh] flex-col sm:max-w-lg" :show-close-button="!busy" @escape-key-down="busy && $event.preventDefault()" @interact-outside="busy && $event.preventDefault()">
      <DialogHeader><DialogTitle>{{ t('bookAI.action') }}</DialogTitle><DialogDescription>{{ t('bookAI.description') }}</DialogDescription></DialogHeader>
      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto">
        <p class="text-sm font-medium">{{ t('bookAI.count', { count: selection.length }) }}</p>
        <ul class="max-h-52 divide-y divide-border overflow-y-auto rounded-lg border"><li v-for="book in selection" :key="book.id" class="flex flex-col gap-1 p-3"><span class="break-words text-sm font-medium">{{ book.title }}</span><span class="break-all text-xs text-muted-foreground">{{ book.sourceFileName }}</span></li></ul>
        <p class="text-sm text-muted-foreground">{{ t(`bookAI.scope.${kind}`) }}</p>
        <Alert v-if="error || invalid" variant="destructive"><AlertDescription>{{ error || t('bookAI.selectionLimit') }}</AlertDescription></Alert>
      </div>
      <DialogFooter><Button variant="outline" :disabled="busy" @click="open = false">{{ t('common.cancel') }}</Button><Button :disabled="busy || invalid || !canSubmit" @click="start"><LoaderCircle v-if="busy" class="animate-spin" data-icon="inline-start" />{{ t(busy ? 'organizationConfirm.submitting' : 'organizationConfirm.confirm') }}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
</template>
