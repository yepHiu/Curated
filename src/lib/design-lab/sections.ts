export const DESIGN_LAB_SECTIONS = [
  { id: "tokens", label: "Tokens" },
  { id: "components", label: "Components" },
  { id: "playground", label: "Playground" },
  { id: "motion", label: "Motion" },
  { id: "accessibility", label: "Accessibility" },
] as const

export type DesignLabSectionId = (typeof DESIGN_LAB_SECTIONS)[number]["id"]
