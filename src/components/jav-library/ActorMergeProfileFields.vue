<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { ActorMergePreviewDTO, ActorMergeProfileSelection } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"

const props = defineProps<{
  preview: ActorMergePreviewDTO
  decisions: Record<string, ActorMergeProfileSelection>
  avatars: { source?: string; target?: string }
  disabled: boolean
}>()
const emit = defineEmits<{ select: [fields: string[], selection: ActorMergeProfileSelection] }>()
const { t } = useI18n()
const sides = ["target", "source"] as const
const groups = computed(() => {
  const pairs: Record<string, string[]> = {
    avatar: ["avatarRemoteUrl", "avatarLocalPath"],
    providerIdentity: ["provider", "providerActorId"],
  }
  const paired = new Set(Object.values(pairs).flat())
  const definitions = [
    ...Object.entries(pairs).map(([key, fields]) => ({ key, fields })),
    ...props.preview.profileFields.filter((field) => !paired.has(field.field)).map((field) => ({ key: field.field, fields: [field.field] })),
  ]
  return definitions.map((definition) => {
    const fields = props.preview.profileFields.filter((field) => definition.fields.includes(field.field))
    return { ...definition, values: fields, conflict: fields.some((field) => field.conflict) }
  }).filter((group) => group.values.some((field) => field.sourceValue || field.targetValue))
})

function value(group: (typeof groups.value)[number], side: ActorMergeProfileSelection) {
  return group.values.map((field) => side === "source" ? field.sourceValue : field.targetValue).filter(Boolean).join(" · ") || t("actors.merge.emptyValue")
}

function selectedSide(group: (typeof groups.value)[number]) {
  return props.decisions[group.fields[0]!] ?? (group.conflict ? undefined : group.values[0]?.defaultSelection)
}
</script>

<template>
  <section v-if="groups.length" class="flex min-w-0 flex-col gap-3">
    <h3 class="text-sm font-semibold">{{ t("actors.merge.resultProfile") }}</h3>
    <p v-if="groups.some((group) => group.conflict)" class="text-xs text-muted-foreground">{{ t("actors.merge.conflictsDescription") }}</p>
    <fieldset v-for="group in groups" :key="group.key" class="min-w-0 rounded-xl border border-border/70 p-3" :disabled="disabled">
      <legend class="px-1 text-sm font-medium">{{ t(`actors.merge.profileField.${group.key}`) }}</legend>
      <div v-if="group.conflict" class="grid min-w-0 gap-2 sm:grid-cols-2">
        <label v-for="side in sides" :key="side" class="flex min-h-11 min-w-0 cursor-pointer items-center gap-3 rounded-lg border border-border/60 p-3 has-[:checked]:border-primary has-[:checked]:bg-primary/10">
          <input type="radio" :name="`actor-merge-${group.key}`" :value="side" :checked="selectedSide(group) === side" class="shrink-0 accent-primary" @change="emit('select', group.fields, side)" />
          <span class="flex min-w-0 flex-col gap-2 text-sm">
            <span class="break-words font-medium">{{ preview[side].name }}</span>
            <Avatar v-if="group.key === 'avatar'" class="size-16 rounded-lg">
              <AvatarImage v-if="avatars[side]" :src="avatars[side]" :alt="preview[side].name" class="object-cover" />
              <AvatarFallback class="rounded-lg">{{ preview[side].name.slice(0, 2) }}</AvatarFallback>
            </Avatar>
            <span v-else class="whitespace-pre-wrap break-words text-muted-foreground">{{ value(group, side) }}</span>
          </span>
        </label>
      </div>
      <template v-else>
        <Avatar v-if="group.key === 'avatar'" class="size-16 rounded-lg">
          <AvatarImage v-if="avatars[selectedSide(group) ?? 'target']" :src="avatars[selectedSide(group) ?? 'target'] || ''" :alt="preview.target.name" class="object-cover" />
          <AvatarFallback class="rounded-lg">{{ preview.target.name.slice(0, 2) }}</AvatarFallback>
        </Avatar>
        <p v-else class="whitespace-pre-wrap break-words text-sm text-muted-foreground">{{ value(group, selectedSide(group) ?? 'target') }}</p>
      </template>
    </fieldset>
  </section>
</template>
