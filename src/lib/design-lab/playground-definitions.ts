export type PlaygroundComponentId = "button" | "input" | "tag" | "card"
export type ViewportPresetId = "mobile" | "tablet" | "desktop" | "custom"
export type RadiusPreset = "md" | "xl" | "full"

export type ButtonPlaygroundState = {
  variant: "default" | "secondary" | "outline" | "destructive" | "ghost"
  size: "default" | "sm" | "lg"
  radius: RadiusPreset
  label: string
  leadingIcon: boolean
  trailingIcon: boolean
  loading: boolean
  disabled: boolean
  fullWidth: boolean
}

export type InputPlaygroundState = {
  density: "default" | "compact"
  radius: RadiusPreset
  value: string
  placeholder: string
  prefixLabel: boolean
  suffixText: boolean
  invalid: boolean
  disabled: boolean
  readonly: boolean
}

export type TagPlaygroundState = {
  variant: "default" | "secondary" | "outline" | "destructive"
  radius: RadiusPreset
  label: string
  leadingIcon: boolean
}

export type CardPlaygroundState = {
  padding: "md" | "lg"
  radius: "lg" | "xl"
  shadow: "sm" | "md" | "lg"
  bordered: boolean
  title: string
  description: string
  dense: boolean
}

export type DesignLabPlaygroundState = {
  componentId: PlaygroundComponentId
  viewportPreset: ViewportPresetId
  customWidth: number
  button: ButtonPlaygroundState
  input: InputPlaygroundState
  tag: TagPlaygroundState
  card: CardPlaygroundState
}

export type PlaygroundCodeOutput = {
  vueSnippet: string
  tokenNotes: string[]
}

export const PLAYGROUND_COMPONENT_OPTIONS: { value: PlaygroundComponentId; label: string }[] = [
  { value: "button", label: "Button" },
  { value: "input", label: "Input" },
  { value: "tag", label: "Tag" },
  { value: "card", label: "Card" },
]

export const VIEWPORT_PRESETS: { value: ViewportPresetId; label: string; width: number | null }[] = [
  { value: "mobile", label: "Mobile", width: 375 },
  { value: "tablet", label: "Tablet", width: 768 },
  { value: "desktop", label: "Desktop", width: 1280 },
  { value: "custom", label: "Custom", width: null },
]

function radiusClass(radius: RadiusPreset): string {
  switch (radius) {
    case "xl":
      return "rounded-[18px]"
    case "full":
      return "rounded-full"
    default:
      return "rounded-[10px]"
  }
}

function cardRadiusClass(radius: CardPlaygroundState["radius"]): string {
  return radius === "xl" ? "rounded-[18px]" : "rounded-[14px]"
}

function cardShadowClass(shadow: CardPlaygroundState["shadow"]): string {
  switch (shadow) {
    case "md":
      return "shadow-[var(--shadow-md)]"
    case "lg":
      return "shadow-[var(--shadow-lg)]"
    default:
      return "shadow-[var(--shadow-sm)]"
  }
}

function compactInputClass(density: InputPlaygroundState["density"]): string {
  return density === "compact" ? "min-h-9 py-1.5 text-sm" : ""
}

export function getViewportWidth(state: DesignLabPlaygroundState): number {
  const preset = VIEWPORT_PRESETS.find((item) => item.value === state.viewportPreset)
  if (!preset || preset.width === null) {
    return state.customWidth
  }
  return preset.width
}

export function createDefaultDesignLabPlaygroundState(): DesignLabPlaygroundState {
  return {
    componentId: "button",
    viewportPreset: "desktop",
    customWidth: 960,
    button: {
      variant: "default",
      size: "default",
      radius: "xl",
      label: "Save changes",
      leadingIcon: true,
      trailingIcon: false,
      loading: false,
      disabled: false,
      fullWidth: false,
    },
    input: {
      density: "default",
      radius: "xl",
      value: "",
      placeholder: "Search title or code",
      prefixLabel: true,
      suffixText: false,
      invalid: false,
      disabled: false,
      readonly: false,
    },
    tag: {
      variant: "secondary",
      radius: "full",
      label: "Queued",
      leadingIcon: true,
    },
    card: {
      padding: "lg",
      radius: "xl",
      shadow: "md",
      bordered: true,
      title: "Prototype card",
      description: "Use cards to test information density before wiring business data.",
      dense: false,
    },
  }
}

export function renderPlaygroundOutput(state: DesignLabPlaygroundState): PlaygroundCodeOutput {
  switch (state.componentId) {
    case "input": {
      const inputClasses = [radiusClass(state.input.radius), compactInputClass(state.input.density)]
        .filter(Boolean)
        .join(" ")
      const prefix = state.input.prefixLabel ? '<span class="text-sm text-muted-foreground">Q</span>\n  ' : ""
      const suffix = state.input.suffixText
        ? '\n  <span class="text-xs text-muted-foreground">⌘K</span>'
        : ""
      return {
        vueSnippet: `<div class="flex items-center gap-3 rounded-2xl border border-border/70 bg-surface px-3 py-2">\n  ${prefix}<Input class="${inputClasses}" placeholder="${state.input.placeholder}"${state.input.invalid ? ' aria-invalid="true"' : ""}${state.input.disabled ? " disabled" : ""}${state.input.readonly ? " readonly" : ""} />${suffix}\n</div>`,
        tokenNotes: [
          "Input backgrounds should stay on muted or surface tokens instead of hard-coded fills.",
          "Focus-visible and invalid states should remain distinct in both light and dark mode.",
        ],
      }
    }
    case "tag":
      return {
        vueSnippet: `<Badge variant="${state.tag.variant}" class="${radiusClass(state.tag.radius)}">${state.tag.leadingIcon ? "\n  <Check />\n  " : ""}${state.tag.label}\n</Badge>`,
        tokenNotes: [
          "Prefer semantic badge variants over ad-hoc status colors.",
          "Use fully rounded tags for compact metadata labels and queue states.",
        ],
      }
    case "card":
      return {
        vueSnippet: `<Card class="${cardRadiusClass(state.card.radius)} ${cardShadowClass(state.card.shadow)}${state.card.bordered ? " border border-border/70" : " border-transparent"} bg-surface">\n  <CardHeader class="${state.card.padding === "lg" ? "gap-3" : "gap-2"}">\n    <CardTitle>${state.card.title}</CardTitle>\n    <CardDescription>${state.card.description}</CardDescription>\n  </CardHeader>\n  <CardContent class="${state.card.dense ? "pt-0 text-sm" : "text-sm leading-6"}">\n    Prototype content area\n  </CardContent>\n</Card>`,
        tokenNotes: [
          "Use surface and border tokens to keep cards aligned with app chrome.",
          "Card elevation should come from the shared shadow scale instead of local box-shadow values.",
        ],
      }
    case "button":
    default: {
      const buttonClasses = [radiusClass(state.button.radius), state.button.fullWidth ? "w-full" : ""]
        .filter(Boolean)
        .join(" ")
      const leadingIcon = state.button.leadingIcon ? '\n  <Sparkles data-icon="inline-start" />' : ""
      const trailingIcon = state.button.trailingIcon ? '\n  <ChevronRight data-icon="inline-end" />' : ""
      const label = state.button.loading ? "Saving" : state.button.label
      const spinner = state.button.loading ? '\n  <Loader2 data-icon="inline-start" class="animate-spin" />' : ""
      const disabled = state.button.disabled || state.button.loading ? " disabled" : ""
      return {
        vueSnippet: `<Button variant="${state.button.variant}" size="${state.button.size}" class="${buttonClasses}"${disabled}>${spinner}${!state.button.loading ? leadingIcon : ""}\n  ${label}${trailingIcon}\n</Button>`,
        tokenNotes: [
          "Primary and destructive buttons should map to semantic action color tokens.",
          "Use loading plus disabled together for async actions that would otherwise be double-submitted.",
        ],
      }
    }
  }
}
