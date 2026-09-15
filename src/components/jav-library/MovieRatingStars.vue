<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Star } from "lucide-vue-next"

const props = defineProps<{
  /** 当前分 0–5，步进 0.5（用于高亮） */
  modelValue: number
  /** 忙碌或只读时禁用半星按钮 */
  disabled?: boolean
}>()

const emit = defineEmits<{
  commit: [value: number]
}>()

const stars = [1, 2, 3, 4, 5] as const
const { t } = useI18n()

/** 整星是否点亮。 */
function starFilled(s: number) {
  return props.modelValue >= s
}

/** 是否显示该星的半星状态。 */
function starHalf(s: number) {
  return props.modelValue >= s - 0.5 && props.modelValue < s
}

/** 按左半/右半提交 0.5 步进分数。 */
function pick(s: number, side: "left" | "right") {
  if (props.disabled) return
  const v = side === "left" ? s - 0.5 : s
  emit("commit", v)
}
</script>

<template>
  <div
    class="inline-flex h-6 items-center gap-px"
    role="group"
    :aria-label="t('rating.ariaLabel')"
  >
    <span
      v-for="s in stars"
      :key="s"
      class="relative inline-flex size-4 shrink-0"
    >
      <!-- 左半：s-0.5 -->
      <button
        type="button"
        class="absolute inset-y-0 left-0 z-[2] w-1/2 cursor-pointer rounded-l-sm border-0 bg-transparent p-0 outline-none ring-offset-background hover:bg-primary/10 focus-visible:ring-2 focus-visible:ring-ring"
        :aria-label="t('rating.score', { s: s - 0.5 })"
        :disabled="disabled"
        @click="pick(s, 'left')"
      />
      <!-- 右半：s -->
      <button
        type="button"
        class="absolute inset-y-0 right-0 z-[2] w-1/2 cursor-pointer rounded-r-sm border-0 bg-transparent p-0 outline-none ring-offset-background hover:bg-primary/10 focus-visible:ring-2 focus-visible:ring-ring"
        :aria-label="t('rating.score', { s })"
        :disabled="disabled"
        @click="pick(s, 'right')"
      />

      <span class="pointer-events-none relative z-0 flex size-4 items-center justify-center" aria-hidden="true">
        <Star
          v-if="starFilled(s)"
          class="size-4 fill-primary text-primary"
        />
        <template v-else-if="starHalf(s)">
          <Star class="absolute size-4 text-muted-foreground/50" />
          <span class="absolute inset-0 w-1/2 overflow-hidden">
            <Star class="size-4 fill-primary text-primary" />
          </span>
        </template>
        <Star
          v-else
          class="size-4 text-muted-foreground/50"
        />
      </span>
    </span>
  </div>
</template>
