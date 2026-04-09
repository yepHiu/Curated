<script setup lang="ts">
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import {
  PLAYGROUND_COMPONENT_OPTIONS,
  VIEWPORT_PRESETS,
  type DesignLabPlaygroundState,
} from "@/lib/design-lab/playground-definitions"

const state = defineModel<DesignLabPlaygroundState>({ required: true })
</script>

<template>
  <div class="rounded-2xl border border-border/70 bg-surface-elevated p-4 lg:p-5">
    <div class="mb-4">
      <h3 class="text-sm font-semibold text-foreground">Inspector</h3>
      <p class="text-xs leading-5 text-muted-foreground">
        Adjust the current specimen and keep output aligned with in-repo usage patterns.
      </p>
    </div>

    <div class="grid gap-4">
      <label class="grid gap-2">
        <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Component</span>
        <Select v-model="state.componentId">
          <SelectTrigger class="w-full rounded-xl">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="option in PLAYGROUND_COMPONENT_OPTIONS"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label class="grid gap-2">
        <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Viewport</span>
        <Select v-model="state.viewportPreset">
          <SelectTrigger class="w-full rounded-xl">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="preset in VIEWPORT_PRESETS"
              :key="preset.value"
              :value="preset.value"
            >
              {{ preset.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </label>

      <label v-if="state.viewportPreset === 'custom'" class="grid gap-2">
        <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Custom Width</span>
        <input v-model="state.customWidth" type="range" min="280" max="1440" step="10" />
        <span class="font-mono text-xs text-foreground">{{ state.customWidth }}px</span>
      </label>

      <template v-if="state.componentId === 'button'">
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Label</span>
          <Input v-model="state.button.label" />
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="grid gap-2">
            <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Variant</span>
            <Select v-model="state.button.variant">
              <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="default">Default</SelectItem>
                <SelectItem value="secondary">Secondary</SelectItem>
                <SelectItem value="outline">Outline</SelectItem>
                <SelectItem value="destructive">Destructive</SelectItem>
                <SelectItem value="ghost">Ghost</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <label class="grid gap-2">
            <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Size</span>
            <Select v-model="state.button.size">
              <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="default">Default</SelectItem>
                <SelectItem value="sm">Small</SelectItem>
                <SelectItem value="lg">Large</SelectItem>
              </SelectContent>
            </Select>
          </label>
        </div>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Radius</span>
          <Select v-model="state.button.radius">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="md">Medium</SelectItem>
              <SelectItem value="xl">Extra Large</SelectItem>
              <SelectItem value="full">Full</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Leading icon</span>
          <Switch v-model="state.button.leadingIcon" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Trailing icon</span>
          <Switch v-model="state.button.trailingIcon" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Loading</span>
          <Switch v-model="state.button.loading" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Disabled</span>
          <Switch v-model="state.button.disabled" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Full width</span>
          <Switch v-model="state.button.fullWidth" />
        </label>
      </template>

      <template v-else-if="state.componentId === 'input'">
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Value</span>
          <Input v-model="state.input.value" />
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Placeholder</span>
          <Input v-model="state.input.placeholder" />
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Density</span>
          <Select v-model="state.input.density">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="default">Default</SelectItem>
              <SelectItem value="compact">Compact</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Radius</span>
          <Select v-model="state.input.radius">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="md">Medium</SelectItem>
              <SelectItem value="xl">Extra Large</SelectItem>
              <SelectItem value="full">Full</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Prefix label</span>
          <Switch v-model="state.input.prefixLabel" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Suffix text</span>
          <Switch v-model="state.input.suffixText" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Invalid</span>
          <Switch v-model="state.input.invalid" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Disabled</span>
          <Switch v-model="state.input.disabled" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Readonly</span>
          <Switch v-model="state.input.readonly" />
        </label>
      </template>

      <template v-else-if="state.componentId === 'tag'">
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Label</span>
          <Input v-model="state.tag.label" />
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Variant</span>
          <Select v-model="state.tag.variant">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="default">Default</SelectItem>
              <SelectItem value="secondary">Secondary</SelectItem>
              <SelectItem value="outline">Outline</SelectItem>
              <SelectItem value="destructive">Destructive</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Radius</span>
          <Select v-model="state.tag.radius">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="md">Medium</SelectItem>
              <SelectItem value="xl">Extra Large</SelectItem>
              <SelectItem value="full">Full</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Leading icon</span>
          <Switch v-model="state.tag.leadingIcon" />
        </label>
      </template>

      <template v-else>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Title</span>
          <Input v-model="state.card.title" />
        </label>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Description</span>
          <Input v-model="state.card.description" />
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="grid gap-2">
            <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Padding</span>
            <Select v-model="state.card.padding">
              <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="md">Medium</SelectItem>
                <SelectItem value="lg">Large</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <label class="grid gap-2">
            <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Radius</span>
            <Select v-model="state.card.radius">
              <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="lg">Large</SelectItem>
                <SelectItem value="xl">Extra Large</SelectItem>
              </SelectContent>
            </Select>
          </label>
        </div>
        <label class="grid gap-2">
          <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Shadow</span>
          <Select v-model="state.card.shadow">
            <SelectTrigger class="w-full rounded-xl"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="sm">Small</SelectItem>
              <SelectItem value="md">Medium</SelectItem>
              <SelectItem value="lg">Large</SelectItem>
            </SelectContent>
          </Select>
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Bordered</span>
          <Switch v-model="state.card.bordered" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-foreground">
          <span>Dense content</span>
          <Switch v-model="state.card.dense" />
        </label>
      </template>
    </div>
  </div>
</template>
