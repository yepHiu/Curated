<script setup lang="ts">
import { onMounted, ref, useId } from "vue"
import { useI18n } from "vue-i18n"
import { LoaderCircle } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

defineProps<{ busy: boolean }>()
const name = defineModel<string>({ required: true })
const emit = defineEmits<{ submit: []; cancel: [] }>()
const { t } = useI18n()
const inputId = useId()
const inputRef = ref<{ $el?: HTMLElement } | null>(null)

function focus() {
  const input = inputRef.value?.$el
  if (input instanceof HTMLInputElement) input.focus()
}

onMounted(focus)
defineExpose({ focus })
</script>

<template>
  <form
    class="flex flex-col gap-3"
    :aria-busy="busy"
    @submit.prevent="emit('submit')"
    @keydown.stop
    @keydown.esc.prevent="emit('cancel')"
  >
    <FieldGroup>
      <Field class="gap-2" :data-disabled="busy || undefined">
        <FieldLabel :for="inputId">{{ t("library.savedViewName") }}</FieldLabel>
        <Input
          :id="inputId"
          ref="inputRef"
          v-model="name"
          maxlength="40"
          autocomplete="off"
          class="rounded-xl"
          data-library-saved-view-create-name
          :placeholder="t('library.savedViewNamePlaceholder')"
          :disabled="busy"
          @pointerdown.stop
        />
      </Field>
    </FieldGroup>
    <div class="flex justify-end">
      <Button
        type="submit"
        size="sm"
        class="min-h-11 rounded-full px-4 sm:min-h-8"
        :disabled="busy || !name.trim()"
      >
        <LoaderCircle v-if="busy" data-icon="inline-start" class="animate-spin" aria-hidden="true" />
        {{ t("library.savedViewSave") }}
      </Button>
    </div>
  </form>
</template>
