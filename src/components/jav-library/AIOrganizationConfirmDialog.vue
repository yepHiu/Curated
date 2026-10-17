<script setup lang="ts">
import { computed, ref, useId, watch } from "vue"
import { useI18n } from "vue-i18n"
import { LoaderCircle } from "lucide-vue-next"
import { useTagOrganization } from "@/composables/use-tag-organization"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import type { OrganizationTask } from "@/services/contracts/topic-service"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Field, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

const { t } = useI18n(), state = useTagOrganization(), { writeEnabled } = useExperimentalAgent()
const id = useId()
const options: OrganizationTask[] = ["tags", "title", "summary"]
const selected = ref<OrganizationTask[]>(["tags"])
watch([state.dialogOpen, state.selection], () => { if (state.dialogOpen.value) selected.value = ["tags"] })
const invalid = computed(() => !state.selection.value?.movieIds.length || state.selection.value.movieIds.length > 600)
function toggle(task: OrganizationTask, checked: boolean | "indeterminate") {
  selected.value = checked === true ? [...new Set([...selected.value, task])] : selected.value.filter(item => item !== task)
}
function close(open: boolean) { if (!state.busy.value) state.dialogOpen.value = open }
</script>

<template>
  <Dialog :open="state.dialogOpen.value" @update:open="close">
    <DialogContent class="flex max-h-[85dvh] flex-col sm:max-w-lg" :show-close-button="!state.busy.value" @escape-key-down="state.busy.value && $event.preventDefault()" @interact-outside="state.busy.value && $event.preventDefault()">
      <DialogHeader class="shrink-0 pr-6">
        <DialogTitle>{{ t('organizationConfirm.title') }}</DialogTitle>
        <DialogDescription>{{ t('organizationConfirm.description') }}</DialogDescription>
      </DialogHeader>
      <div class="flex min-h-0 flex-col gap-5 overflow-y-auto overscroll-contain" data-organization-confirm>
        <section class="flex min-h-0 flex-col gap-2" :aria-label="t('organizationConfirm.movies')">
          <p class="text-sm font-medium">{{ t('topics.selectedMovies', { count: state.selection.value?.movieIds.length ?? 0 }) }}</p>
          <ul class="flex max-h-52 shrink-0 flex-col divide-y divide-border overflow-y-auto rounded-lg border" data-organization-movies>
            <li v-for="movie in state.selection.value?.movies" :key="movie.id" class="flex min-w-0 flex-col gap-1 p-3">
              <span class="break-words text-sm font-medium">{{ movie.title }}</span>
              <span class="break-all text-xs text-muted-foreground">{{ t('organizationConfirm.code') }} <span class="font-mono">{{ movie.code }}</span></span>
            </li>
          </ul>
        </section>
        <FieldSet :disabled="state.busy.value">
          <FieldLegend variant="label">{{ t('organizationConfirm.tasks') }}</FieldLegend>
          <FieldGroup class="gap-1">
            <Field v-for="task in options" :key="task" orientation="horizontal" :data-disabled="state.busy.value" class="min-h-11">
              <Checkbox :id="`${id}-${task}`" :model-value="selected.includes(task)" :disabled="state.busy.value" :data-organization-task="task" @update:model-value="toggle(task, $event)" />
              <FieldLabel :for="`${id}-${task}`" class="min-h-11 cursor-pointer">{{ t(`organizationConfirm.options.${task}`) }}</FieldLabel>
            </Field>
          </FieldGroup>
        </FieldSet>
        <p v-if="selected.some(task => task !== 'tags')" class="text-xs text-muted-foreground">{{ t('organizationConfirm.translationHint') }}</p>
        <Alert v-if="state.error.value || invalid || !writeEnabled" variant="destructive"><AlertDescription>{{ state.error.value || (invalid ? t('topics.selectionLimit') : t('topics.errors.AI_PERMISSION_REQUIRED')) }}</AlertDescription></Alert>
      </div>
      <DialogFooter class="shrink-0 gap-2">
        <Button variant="outline" class="min-h-11 rounded-full sm:min-h-9" :disabled="state.busy.value" @click="close(false)">{{ t('common.cancel') }}</Button>
        <Button class="min-h-11 rounded-full sm:min-h-9" :disabled="state.busy.value || invalid || !writeEnabled || !selected.length" data-confirm-organization @click="state.startSelected(selected)">
          <LoaderCircle v-if="state.busy.value" class="animate-spin" data-icon="inline-start" />
          {{ t(state.busy.value ? 'organizationConfirm.submitting' : 'organizationConfirm.confirm') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
